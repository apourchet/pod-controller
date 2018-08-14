package controller

import (
	"fmt"
	"os/exec"
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

func (p ProbeSpec) GetCheck(runtime RuntimeStrategy) Check {
	if p.Exec != nil {
		cmd := exec.Command((*p.Exec)[0], (*p.Exec)[1:]...)
		shellcheck := NewShellCheck(cmd)
		return shellcheck
	} else if p.HTTPGet != nil {
		host := fmt.Sprintf("%s:%v", p.HTTPGet.Host, p.HTTPGet.Port)
		httpcheck := NewHTTPCheck(host, p.HTTPGet.Path)
		httpcheck.Scheme = p.HTTPGet.Scheme
		return httpcheck
	}
	return HealthyCheck{}
}

type LivenessProbeSpec struct{ ProbeSpec }

type ReadinessProbeSpec struct{ ProbeSpec }

func (p LivenessProbeSpec) Materialize(runtime RuntimeStrategy) (Probe, error) {
	check := p.GetCheck(runtime)
	probe := NewLivenessProbe(check)
	probe.BaseProbe = p.GetBaseProbe()
	return probe, nil
}

func (p ReadinessProbeSpec) Materialize(runtime RuntimeStrategy) (Probe, error) {
	check := p.GetCheck(runtime)
	probe := NewReadinessProbe(check)
	probe.BaseProbe = p.GetBaseProbe()
	return probe, nil
}
