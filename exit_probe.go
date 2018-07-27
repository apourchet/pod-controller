package controller

import (
	"sync"
)

// ExitProbe will essentially return HEALTHY while it's running,
// and HEALTHY if the exit code of the process was 0. If the exit
// code was not 0 it will return UNHEALTHY forever.
// The exitprobe will never go back to a healthy state after
// reaching an unhealthy state and stopping.
// Since we cannot preempt a goroutine, the Stop method is a no-op.
type ExitProbe struct {
	sync.Mutex

	Check     Check
	isRunning bool
	success   bool
	err       error
}

func NewExitProbe(check Check) *ExitProbe {
	return &ExitProbe{
		Check: check,
	}
}

func (p *ExitProbe) Start() {
	p.Lock()
	p.isRunning = true
	p.Unlock()

	go func() {
		success, err := p.Check.Run()
		p.Lock()
		p.success, p.err = success, err
		p.isRunning = false
		p.Unlock()
	}()
}

func (p *ExitProbe) Healthy() (bool, error) {
	p.Lock()
	defer p.Unlock()
	if p.isRunning {
		return true, nil
	}
	return p.success, p.err
}

func (p *ExitProbe) Running() bool {
	p.Lock()
	defer p.Unlock()
	return p.isRunning
}

func (p *ExitProbe) Stop() {}
