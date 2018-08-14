package controller

import (
	"fmt"
	"time"
)

type ProbeSet struct {
	Exit      Probe
	Liveness  Probe
	Readiness Probe
}

func (pset ProbeSet) Start() {
	pset.Exit.Start()
	pset.Liveness.Start()
	pset.Readiness.Start()
}

type ProbeSpec struct {
	Exec    *[]string
	HTTPGet *struct {
		Host   string
		Path   string
		Port   int
		Scheme string
		// TODO: add headers
	}
	// TODO: add TCP socket

	InitialDelaySeconds int
	PeriodSeconds       int
	SuccessThreshold    int
	FailureThreshold    int
	TimeoutSeconds      int
}

func (p ProbeSpec) GetBaseProbe() BaseProbe {
	return BaseProbe{
		InitialDelay:     time.Duration(p.InitialDelaySeconds) * time.Second,
		Period:           time.Duration(p.PeriodSeconds) * time.Second,
		Timeout:          time.Duration(p.TimeoutSeconds) * time.Second,
		SuccessThreshold: p.SuccessThreshold,
		FailureThreshold: p.FailureThreshold,
	}
}

func (p ProbeSpec) GetCheck(ctn Container) Check {
	if p.HTTPGet != nil {
		host := fmt.Sprintf("%s:%v", p.HTTPGet.Host, p.HTTPGet.Port)
		httpcheck := NewHTTPCheck(host, p.HTTPGet.Path)
		httpcheck.Scheme = p.HTTPGet.Scheme
		return httpcheck
	} else if p.Exec != nil {
		return RunnerCheck{
			Runner: func() error {
				code, err := ctn.Exec((*p.Exec)[0], (*p.Exec)[1:]...)
				if err != nil {
					return err
				} else if code != 0 {
					return fmt.Errorf("non-0 exit code on exec check: %d", code)
				}
				return nil
			},
		}
	}

	// By default a check will constantly return healthy.
	return HealthyCheck{}
}

type LivenessProbeSpec struct{ ProbeSpec }

type ReadinessProbeSpec struct{ ProbeSpec }

func (p LivenessProbeSpec) Materialize(ctn Container) (Probe, error) {
	check := p.GetCheck(ctn)
	probe := NewLivenessProbe(check)
	probe.BaseProbe = p.GetBaseProbe()
	return probe, nil
}

func (p ReadinessProbeSpec) Materialize(ctn Container) (Probe, error) {
	check := p.GetCheck(ctn)
	probe := NewReadinessProbe(check)
	probe.BaseProbe = p.GetBaseProbe()
	return probe, nil
}
