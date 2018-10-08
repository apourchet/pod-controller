package controller

import (
	"fmt"
	"time"

	"github.com/benbjohnson/clock"
	oci "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
)

// The pod controller will need to keep track of the states of the containeres it manages. It will
// mostly only really need the set of probes that match each container and will simply aggregate
// the information from the probes into a singular state per container.
// A second pass will take the states of each of the containeres and aggregate those into a single
// HEALTHY bit of information.
type PodController interface {
	Start() error
	Status() []*ContainerStatus

	// Kill tries to send the signal to the containers and returns the status
	// of the containers.
	Kill(signal int) []error

	// This is the healthy bit that the pod controller should aim to get right
	// as it will determine when the pod should get rescheduled.
	Healthy() bool
}

type PodSpec struct {
	InitContainers []InitContainerSpec
	Containers     []ContainerSpec
}

type InitContainerSpec struct {
	Name     string
	Spec     oci.Spec
	Metadata map[string]interface{}
}

type ContainerSpec struct {
	Name           string
	Spec           oci.Spec
	LivenessProbe  LivenessProbeSpec
	ReadinessProbe ReadinessProbeSpec

	Metadata map[string]interface{}
}

type ContainerInfo struct {
	ctn    Container
	probes *ProbeSet
	status *ContainerStatus
}

// controller implements the PodController interface.
type controller struct {
	// A map from container ID/name to container status
	InitInfos map[string]ContainerInfo
	MainInfos map[string]ContainerInfo

	InitOrder []string
	MainOrder []string

	Clock clock.Clock
}

func NewPodController(spec PodSpec, runtimePath string) (*controller, error) {
	runtime, err := LoadPlugin(runtimePath)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return WithBootstrapper(spec, runtime.Bootstrapper)
}

func WithBootstrapper(spec PodSpec, bootstrapper ContainerBootstrapper) (*controller, error) {
	initContainers, mainContainers, err := materializeContainers(spec, bootstrapper)
	if err != nil {
		return nil, err
	}
	return WithContainers(spec, initContainers, mainContainers)
}

func WithContainers(spec PodSpec, initContainers, mainContainers []Container) (*controller, error) {
	if len(spec.InitContainers)+len(spec.Containers) != len(initContainers)+len(mainContainers) {
		return nil, fmt.Errorf("Missing names for some of the containers")
	}
	c := &controller{
		InitInfos: map[string]ContainerInfo{},
		MainInfos: map[string]ContainerInfo{},
		Clock:     clock.New(),
	}
	for i, ctn := range initContainers {
		ctnSpec := spec.InitContainers[i]
		status := NewContainerStatus(ctnSpec.Name)
		c.InitInfos[ctnSpec.Name] = ContainerInfo{
			ctn:    ctn,
			status: status,
		}
		c.InitOrder = append(c.InitOrder, ctnSpec.Name)
	}
	for i, ctn := range mainContainers {
		ctnSpec := spec.Containers[i]
		status := NewContainerStatus(ctnSpec.Name)
		probeSet, err := c.getProbeSet(ctnSpec, ctn)
		if err != nil {
			return c, err
		}
		c.MainInfos[ctnSpec.Name] = ContainerInfo{
			ctn:    ctn,
			status: status,
			probes: probeSet,
		}
		c.MainOrder = append(c.MainOrder, ctnSpec.Name)
	}
	return c, nil
}

// Start goes through the spec of the controller and starts the init containers
// as well as regular containers. It also starts a background thread that watches
// the probes to update the container statuses.
func (c *controller) Start() error {
	// First run the InitContainers of the pod. These containers will be
	// run in sequence, and not healthchecked.
	for _, name := range c.InitOrder {
		info := c.InitInfos[name]
		// Start that init container, then wait for it to run to completion.
		if err := info.ctn.Start(); err != nil {
			return errors.WithStack(err)
		} else if err := info.ctn.Wait(); err != nil {
			return errors.WithStack(err)
		}
		// TODO: timeout these init containers.
	}
	go c.watch()
	return nil
}

// Status gathers the statuses of all the containers and appends them to a list
// for display.
func (c *controller) Status() []*ContainerStatus {
	statuses := []*ContainerStatus{}
	for _, name := range c.MainOrder {
		info := c.MainInfos[name]
		statuses = append(statuses, info.status)
	}
	return statuses
}

// Kill sends the kill signal to all of the containers in the pod.
func (c *controller) Kill(signal int) []error {
	errs := []error{}
	for _, name := range c.MainOrder {
		info := c.MainInfos[name]
		if err := info.ctn.Kill(signal); err != nil {
			err = fmt.Errorf("failed to send kill signal %d to container %s: %v",
				signal, name, err)
			errs = append(errs, err)
			info.status.AddError(&ProbeError{
				Message:   err.Error(),
				Timestamp: c.Clock.Now(),
			})
		}
	}
	return errs
}

// Healthy only looks through the container statuses to determine the health of the pod,
// it relies on the eventual consistency provided by the background thread that the controller
// spawns with `watch`.
// For now if a single container within the pod is neither Started nor Healthy we deem the pod
// to be unhealthy and should be rescheduled.
func (c *controller) Healthy() bool {
	for _, name := range c.MainOrder {
		info := c.MainInfos[name]
		if !info.status.Healthy() {
			return false
		}
	}
	return true
}

// watch starts the probes for all of its containers, then
// goes through all of the probes for all the containers and updates the statuses of
// the containers within the pod. It does this pass every second.
func (c *controller) watch() {
	for _, info := range c.MainInfos {
		info.probes.Start()
	}
	for {
		c.Clock.Sleep(1 * time.Second)
		for _, info := range c.MainInfos {
			status, probeset := info.status, info.probes
			lastState := status.LastState()

			// If we get an error we havent seen before we will append it to our list
			// of latest errors.
			state, mustRestart, errs := c.nextState(lastState, probeset)
			errs = filterErrors(errs)
			if len(errs) > 0 {
				if status.LatestError().Message != errs[0].Error() {
					for _, msg := range stringifyErrors(errs) {
						err := &ProbeError{Message: msg, Timestamp: c.Clock.Now()}
						status.AddError(err)
					}
				} else {
					status.LatestError().Timestamp = c.Clock.Now()
				}
			}

			// If the state does not change, just continue to the next set
			// of probes.
			if state == status.LastState() {
				continue
			}
			status.AddState(state)

			if mustRestart {
				status.RecordRestart()
				// TODO: restart container and change newStatus
			}

			// TODO: Prune that new status so that memory never explodes.
		}
		// TODO stopping mechanism
	}
}

// nextState computes the next state for the container from its status. It also computes
// whether or not the container needs to be restarted and returns some of the errors the probes
// might have run into.
func (c *controller) nextState(state ContainerState, probes *ProbeSet) (next ContainerState, restart bool, errs []error) {
	exitHealth, exitErr := probes.Exit.Healthy()
	exitRunning := probes.Exit.Running()

	liveHealth, liveErr := probes.Liveness.Healthy()
	liveStarted, liveRunning := probes.Liveness.Started(), probes.Liveness.Running()

	errs = []error{exitErr, liveErr}

	// TODO: restart computation
	restart = false
	switch state {
	case Failed, Finished, Terminal:
		return state, restart, errs
	case Started, Healthy, Failing:
		// If the container exited we can get the next state easily.
		if !exitRunning {
			if exitHealth {
				return Finished, restart, errs
			}
			return Failed, restart, errs
		}

		// If the liveness has not started yet then it means the exit probe is still
		// in its starting phase.
		if !liveStarted {
			return Started, restart, errs
		}

		// If the container did not exit yet we need to check that the liveness
		// probe has not given up.
		if !liveRunning {
			return Terminal, restart, errs
		}

		// If the liveness probe is still running we just return healthy or not
		// depending on its bit.
		if liveHealth {
			return Healthy, restart, errs
		}
		return Failing, restart, errs
	default:
		panic(fmt.Sprintf("unrecognized state: %v", state))
	}
	return
}

func materializeContainers(spec PodSpec, bootstrapper ContainerBootstrapper) ([]Container, []Container, error) {
	initContainers, mainContainers := []Container{}, []Container{}
	for _, spec := range spec.InitContainers {
		ctn, err := bootstrapper(spec.Spec, spec.Metadata)
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}
		initContainers = append(initContainers, ctn)
	}

	for _, spec := range spec.Containers {
		ctn, err := bootstrapper(spec.Spec, spec.Metadata)
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}
		mainContainers = append(mainContainers, ctn)
	}
	return initContainers, mainContainers, nil
}

func (c *controller) getProbeSet(spec ContainerSpec, ctn Container) (*ProbeSet, error) {
	livenessProbe, err := spec.LivenessProbe.Materialize(ctn)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	readinessProbe, err := spec.ReadinessProbe.Materialize(ctn)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	exitProbe := NewExitProbe(ExitCheck(ctn))
	pset := NewProbeSet(exitProbe, livenessProbe, readinessProbe)
	pset.Sleeper = func(d time.Duration) { c.Clock.Sleep(d) }
	return pset, nil
}
