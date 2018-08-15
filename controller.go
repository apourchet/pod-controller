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
type ContainerState int

const (
	Started   ContainerState = iota
	Healthy                  // When the liveness probe is healthy
	Unhealthy                // When the liveness probe last failed after a healthy state
	Terminal                 // When the liveness probe has given up
	Finished                 // When the container exited with a 0 status code
	Failed                   // When the container exited with non-0 status code
)

type ContainerStatus struct {
	States       []ContainerState
	LatestErrors []error
	Restarts     int

	spec ContainerSpec
}

type PodController interface {
	Start() error
	Status() []ContainerStatus
	Healthy() bool // False should cause the pod to be restarted
}

type PodSpec struct {
	InitContainers []oci.Spec
	Containers     []ContainerSpec
}

type ContainerSpec struct {
	oci.Spec

	Name           string
	LivenessProbe  LivenessProbeSpec
	ReadinessProbe ReadinessProbeSpec
}

// controller implements the PodController interface.
type controller struct {
	// TODO: concurrent access

	Spec PodSpec

	// The container launching strategy
	Runtime RuntimeStrategy

	// A map from container ID/name to container status
	Statuses map[string]ContainerStatus

	// A map from container ID/name to its probes
	Probes map[string]ProbeSet

	Clock clock.Clock
}

func NewPodController(spec PodSpec, runtimePath string) (*controller, error) {
	runtime, err := LoadPlugin(runtimePath)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &controller{
		Spec:     spec,
		Runtime:  runtime,
		Statuses: map[string]ContainerStatus{},
		Probes:   map[string]ProbeSet{},
		Clock:    clock.New(),
	}, nil
}

// Start goes through the spec of the controller and starts the init containers
// as well as regular containers. It also starts a background thread that watches
// the probes to update the container statuses.
func (c *controller) Start() error {
	// First run the InitContainers of the pod. These containers will be
	// run in sequence, and not healthchecked.
	for _, spec := range c.Spec.InitContainers {
		cmd, err := c.Runtime.Bootstrapper(spec)
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
		c.Statuses[spec.Name] = ContainerStatus{
			States:       []ContainerState{Started},
			LatestErrors: []error{},
		}

		ctn, err := c.Runtime.Bootstrapper(spec.Spec)
		if err != nil {
			return errors.WithStack(err)
		}

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
	for _, status := range c.Statuses {
		statuses = append(statuses, status)
	}
	return statuses
}

// Healthy only looks through the container statuses to determine the health of the pod,
// it relies on the eventual consistency provided by the background thread that the controller
// spawns with `watch`.
// For now if a single container within the pod is neither Started nor Healthy we deem the pod
// to be unhealthy and should be rescheduled.
func (c *controller) Healthy() bool {
	for _, status := range c.Statuses {
		lastState := status.States[len(status.States)-1]
		if lastState == Started || lastState == Healthy || lastState == Unhealthy {
			continue
		}
		return false
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
			state, mustRestart, errs := c.nextState(status, probeset)
			errs = filterErrors(errs)

			newStatus := ContainerStatus{
				States:       append(status.States, state),
				LatestErrors: append(status.LatestErrors, errs...),
			}
			if mustRestart {
				newStatus.Restarts += 1
				// TODO: restart container and change newStatus
			}

			c.Statuses[cname] = newStatus
		}
		c.Clock.Sleep(1 * time.Second)
		// TODO stopping mechanism
	}
}

// nextState computes the next state for the container from its status. It also computes
// whether or not the container needs to be restarted and returns some of the errors the probes
// might have run into.
func (c *controller) nextState(status ContainerStatus, probes ProbeSet) (next ContainerState, restart bool, errs []error) {
	state := status.States[len(status.States)-1]

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
	case Started, Healthy, Unhealthy:
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
		return Unhealthy, restart, errs
	default:
		panic(fmt.Sprintf("unrecognized state: %v", state))
	}
	return
}

// LastState returns the last state of the container status.
func (status ContainerStatus) LastState() ContainerState {
	return status.States[len(status.States)-1]
}
