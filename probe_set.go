package controller

import (
	"time"
)

type ProbeSet struct {
	Exit      *ExitProbe
	Liveness  Probe
	Readiness Probe

	Sleeper func(time.Duration)
}

func NewProbeSet(exit *ExitProbe, liveness, readiness Probe) *ProbeSet {
	return &ProbeSet{
		Exit:      exit,
		Liveness:  liveness,
		Readiness: readiness,
		Sleeper:   time.Sleep,
	}
}

func (pset *ProbeSet) Start() {
	pset.Exit.Start()
	go func() {
		for !pset.Exit.Waiting() {
			pset.Sleeper(1 * time.Second)
		}
		pset.Liveness.Start()
		pset.Readiness.Start()
	}()
}
