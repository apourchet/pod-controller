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
	Status() []ContainerStatus

	// Kill tries to send the signal to the containers and returns the status
	// of the containers.
	Kill(signal int) []ContainerStatus

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

// controller implements the PodController interface.
type controller struct {
	// TODO: concurrent access

	Spec PodSpec

	// The container launching strategy
	Runtime RuntimeStrategy

	// A map from container ID/name to container status
	Statuses    map[string]*ContainerStatus
	StatusSlice []*ContainerStatus

	// A map from container ID/name to its probes
	Probes map[string]ProbeSet

	Clock clock.Clock
}

func NewPodController(spec PodSpec, runtimePath string) (*controller, error) {
	runtime, err := LoadPlugin(runtimePath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return WithBootstrapper(spec, runtime.Bootstrapper), nil
}

func WithBootstrapper(spec PodSpec, bootstrapper ContainerBootstrapper) *controller {
	return &controller{
		Spec:        spec,
		Runtime:     RuntimeStrategy{bootstrapper},
		Statuses:    map[string]*ContainerStatus{},
		StatusSlice: []*ContainerStatus{},
		Probes:      map[string]ProbeSet{},
		Clock:       clock.New(),
	}
}

// Start goes through the spec of the controller and starts the init containers
// as well as regular containers. It also starts a background thread that watches
// the probes to update the container statuses.
func (c *controller) Start() error {
	// First run the InitContainers of the pod. These containers will be
	// run in sequence, and not healthchecked.
	for _, spec := range c.Spec.InitContainers {
		cmd, err := c.Runtime.Bootstrapper(spec.Spec, spec.Metadata)
		if err != nil {
			return errors.WithStack(err)
		}

		// Start that init container, then wait for it to run to completion.
		if err := cmd.Start(); err != nil {
			return errors.WithStack(err)
		} else if err := cmd.Wait(); err != nil {
			return errors.WithStack(err)
		}

		// TODO: timeout these init containers.
	}

	// For each one of the containers in the spec, we create a status and we
	// start that container and start the probes related to it.
	for _, spec := range c.Spec.Containers {
		status := &ContainerStatus{
			Name:         spec.Name,
			States:       []ContainerState{Started},
			LatestErrors: []*ProbeError{},
			spec:         spec,
		}
		c.Statuses[spec.Name] = status
		c.StatusSlice = append(c.StatusSlice, status)

		ctn, err := c.Runtime.Bootstrapper(spec.Spec, spec.Metadata)
		if err != nil {
			return errors.WithStack(err)
		}
		c.Statuses[spec.Name].ctn = ctn

		exitCheck := ExitCheck(ctn)
		exitProbe := NewExitProbe(exitCheck)

		livenessProbe, err := spec.LivenessProbe.Materialize(ctn)
		if err != nil {
			return errors.WithStack(err)
		}

		readinessProbe, err := spec.ReadinessProbe.Materialize(ctn)
		if err != nil {
			return errors.WithStack(err)
		}

		c.Probes[spec.Name] = ProbeSet{
			Exit:      exitProbe,
			Liveness:  livenessProbe,
			Readiness: readinessProbe,
		}
	}

	// This watch will actually start all the probes, the exitProbes will start the
	// container by themselves.
	go c.watch()

	return nil
}

// Status gathers the statuses of all the containers and appends them to a list
// for display.
func (c *controller) Status() []ContainerStatus {
	statuses := []ContainerStatus{}
	for _, status := range c.StatusSlice {
		statuses = append(statuses, *status)
	}
	return statuses
}

// Kill sends the kill signal to all of the containers in the pod.
func (c *controller) Kill(signal int) []ContainerStatus {
	for cname, status := range c.Statuses {
		if err := status.ctn.Kill(signal); err != nil {
			err = fmt.Errorf("failed to send kill signal %d to container %s: %v",
				signal, cname, err)
			status.AddError(&ProbeError{
				Message:   err.Error(),
				Timestamp: c.Clock.Now(),
			})
		}
	}
	return c.Status()
}

// Healthy only looks through the container statuses to determine the health of the pod,
// it relies on the eventual consistency provided by the background thread that the controller
// spawns with `watch`.
// For now if a single container within the pod is neither Started nor Healthy we deem the pod
// to be unhealthy and should be rescheduled.
func (c *controller) Healthy() bool {
	for _, status := range c.Statuses {
		if !status.Healthy() {
			return false
		}
	}
	return true
}

// watch starts the probes for all of its containers, then
// goes through all of the probes for all the containers and updates the statuses of
// the containers within the pod. It does this pass every second.
func (c *controller) watch() {
	for _, probeset := range c.Probes {
		probeset.Start()
	}

	for {
		for cname, probeset := range c.Probes {
			status := c.Statuses[cname]
			lastState := status.LastState()

			// If we get an error we havent seen before we will append it to our list
			// of latest errors.
			state, mustRestart, errs := c.nextState(lastState, probeset)
			errs = filterErrors(errs)
			if len(errs) > 0 {
				if status.LatestError().Message != errs[0].Error() {
					for _, msg := range stringifyErrors(errs) {
						err := &ProbeError{Message: msg, Timestamp: c.Clock.Now()}
						c.Statuses[cname].AddError(err)
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
			c.Statuses[cname].AddState(state)

			if mustRestart {
				c.Statuses[cname].RecordRestart()
				// TODO: restart container and change newStatus
			}

			// TODO: Prune that new status so that memory never explodes.
		}
		c.Clock.Sleep(1 * time.Second)
		// TODO stopping mechanism
	}
}

// nextState computes the next state for the container from its status. It also computes
// whether or not the container needs to be restarted and returns some of the errors the probes
// might have run into.
func (c *controller) nextState(state ContainerState, probes ProbeSet) (next ContainerState, restart bool, errs []error) {
	exitHealth, exitErr := probes.Exit.Healthy()
	exitRunning := probes.Exit.Running()

	liveHealth, liveErr := probes.Liveness.Healthy()
	liveRunning := probes.Liveness.Running()

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
