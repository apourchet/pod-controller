package controller

import (
	"runtime"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
)

// Probes will run a Check every so often depending on what type of probe
// we are dealing with. Probes will also have the retry logic and the timeout
// logic in them so that the Check interface remains stupid simple.
// Any calls to Healthy() after Stop() has undefined behavior.
// Healthy can also return an error, this will typically correspond to
// the last error encountered by the probe, so even a `true` healthy bit
// might return a non-nil error.
type Probe interface {
	Start()
	Healthy() (healthy bool, err error)
	Started() bool
	Running() bool
	Stop()
}

var _ Probe = NewLivenessProbe(nil)
var _ Probe = NewReadinessProbe(nil)
var _ Probe = &ExitProbe{}

type BaseProbe struct {
	sync.Mutex

	InitialDelay     time.Duration
	Period           time.Duration
	Timeout          time.Duration
	SuccessThreshold int
	FailureThreshold int

	consecutiveSuccesses int
	consecutiveFailures  int
	hasFailed            bool
	hasSucceeded         bool
	err                  error
}

func (p *BaseProbe) onTickResult(success bool, err error) {
	p.Lock()
	defer p.Unlock()
	p.hasFailed = p.hasFailed || !success
	p.hasSucceeded = p.hasSucceeded || success
	if err != nil {
		p.err = err
	}

	if !success {
		p.consecutiveFailures += 1
		p.consecutiveSuccesses = 0
	} else {
		p.consecutiveFailures = 0
		p.consecutiveSuccesses += 1
	}
	return
}

func (p *BaseProbe) onTimeout() {
	p.Lock()
	defer p.Unlock()
	p.hasFailed = true
	p.consecutiveFailures += 1
	p.consecutiveSuccesses = 0
}

// A liveness probe will continue performing the same operation at an
// interval, and will only stop if FailureThreshold is reached.
type LongLivedProbe struct {
	BaseProbe
	sync.Mutex

	Check Check
	Clock clock.Clock

	isRunning  bool
	isHealthy  bool
	hasStarted bool
}

func newLongLivedProbe(check Check) *LongLivedProbe {
	return &LongLivedProbe{
		// TODO: Check defaults in the k8s reference.
		BaseProbe: BaseProbe{
			InitialDelay:     1 * time.Second,
			Period:           5 * time.Second,
			Timeout:          1 * time.Second,
			SuccessThreshold: 1,
			FailureThreshold: 1,
		},
		Check: check,
		Clock: clock.New(),
	}
}

func (p *LongLivedProbe) Start() {
	p.Lock()
	p.isRunning = true
	p.hasStarted = true
	p.Unlock()

	go func() {
		p.Clock.Sleep(p.InitialDelay)
		done := make(chan bool)
		for {
			// If stop was called by another goroutine then we return from the
			// run loop.
			if !p.Running() {
				return
			}

			// We know the probe is running and hasnt failed out yet, so we run
			// a single tick.
			var success bool
			var err error
			go func() {
				success, err = p.Check.Run()
				done <- true
			}()
			runtime.Gosched()

			// Check for timeout of that tick here, and process the result of
			// the tick.
			select {
			case <-p.Clock.After(p.Timeout):
				p.onTimeout()
			case <-done:
				p.onTickResult(success, err)
			}

			// Check for max successes and failures. If the max failures in a row
			// has been reached we stop the probe and set its state to UNHEALTHY.
			p.Lock()
			if p.consecutiveFailures >= p.FailureThreshold {
				p.isHealthy = false
				p.Unlock()
				p.Stop()
				return
			} else if p.consecutiveSuccesses >= p.SuccessThreshold ||
				(!p.hasFailed && success) || (!p.hasSucceeded && success) {
				p.isHealthy = true
			} else if !success {
				p.isHealthy = false
			}
			p.Unlock()

			// This line is hit if we have not hit either of the thresholds.
			p.Clock.Sleep(p.Period)
		}
	}()
}

func (p *LongLivedProbe) Healthy() (bool, error) {
	p.Lock()
	defer p.Unlock()
	return p.isHealthy, p.err
}

func (p *LongLivedProbe) Running() bool {
	p.Lock()
	defer p.Unlock()
	return p.isRunning
}

func (p *LongLivedProbe) Started() bool {
	p.Lock()
	defer p.Unlock()
	return p.hasStarted
}

func (p *LongLivedProbe) Stop() {
	p.Lock()
	defer p.Unlock()
	p.isRunning = false
}
