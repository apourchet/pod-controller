package controller

import (
	"sync"
	"time"
)

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
	sync.Mutex

	Name         string
	States       []ContainerState
	LatestErrors []*ProbeError
	Restarts     int

	ctn  Container
	spec ContainerSpec
}

type ProbeError struct {
	Message   string
	Timestamp time.Time
}

// LastState returns the last state of the container status.
func (status *ContainerStatus) LastState() ContainerState {
	status.Lock()
	defer status.Unlock()
	return status.States[len(status.States)-1]
}

// LatestError returns the latest error that the container has received, or the
// empty string if there are none.
func (status *ContainerStatus) LatestError() *ProbeError {
	status.Lock()
	defer status.Unlock()
	if len(status.LatestErrors) == 0 {
		return &ProbeError{Message: ""}
	}
	return status.LatestErrors[len(status.LatestErrors)-1]
}

func (status *ContainerStatus) AddError(err *ProbeError) {
	status.Lock()
	defer status.Unlock()
	status.LatestErrors = append(status.LatestErrors, err)
}

func (status *ContainerStatus) AddState(state ContainerState) {
	status.Lock()
	defer status.Unlock()
	status.States = append(status.States, state)
}

func (status *ContainerStatus) RecordRestart() {
	status.Lock()
	defer status.Unlock()
	status.Restarts++
}

// Healthy returns true if the container is in one of the 3 states:
// Started, Healthy, Unhealthy
// A status of Unhealthy means that the liveness probe has failed but has not reached the
// failureThreshold. So in essence the container is still in a valid state, but most likely
// transitioning into a failed state soon if the liveness probe keeps failing.
func (status *ContainerStatus) Healthy() bool {
	lastState := status.LastState()
	return lastState == Started || lastState == Healthy || lastState == Unhealthy
}
