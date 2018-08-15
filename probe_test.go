package controller

import (
	"runtime"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/require"
)

type mockCheck struct {
	Clock    clock.Clock
	Duration time.Duration
	Success  bool
	Err      error
}

func newMockCheck(clock clock.Clock, duration time.Duration, success bool, err error) mockCheck {
	return mockCheck{
		Clock:    clock,
		Duration: duration,
		Success:  success,
		Err:      err,
	}
}

func (c mockCheck) Run() (bool, error) {
	c.Clock.Sleep(c.Duration)
	return c.Success, c.Err
}

type mockMultiCheck struct {
	checks []Check
	index  int
}

func newMockMultiCheck() *mockMultiCheck {
	return &mockMultiCheck{
		checks: []Check{},
		index:  0,
	}
}

func (c *mockMultiCheck) Add(check Check) *mockMultiCheck {
	c.checks = append(c.checks, check)
	return c
}

func (c *mockMultiCheck) Run() (bool, error) {
	if c.index >= len(c.checks) {
		panic("mockMultiCheck ran out of checks")
	}
	check := c.checks[c.index]
	defer func() { c.index += 1 }()
	return check.Run()
}

func gosched() {
	time.Sleep(1 * time.Millisecond)
	runtime.Gosched()
}

func timeTravel(clock *clock.Mock, count int, step time.Duration) {
	for i := 0; i < count; i++ {
		clock.Add(step)
		gosched()
	}
}

func TestLongLivedProbe(t *testing.T) {
	t.Run("unhealthy_check", func(t *testing.T) {
		clock := clock.NewMock()
		check := newMockCheck(clock, 0*time.Second, false, nil)
		multicheck := newMockMultiCheck().Add(check)

		probe := newLongLivedProbe(multicheck)
		probe.FailureThreshold = 1
		probe.InitialDelay = 0 * time.Second
		probe.Period = 2 * time.Second
		probe.Clock = clock

		probe.Start()
		gosched()

		timeTravel(clock, 2, time.Second)

		healthy, err := probe.Healthy()
		require.False(t, healthy)
		require.NoError(t, err)
	})
	t.Run("timeout_check", func(t *testing.T) {
		clock := clock.NewMock()
		check := newMockCheck(clock, 10*time.Second, true, nil)
		multicheck := newMockMultiCheck().Add(check)

		probe := newLongLivedProbe(multicheck)
		probe.SuccessThreshold = 2
		probe.FailureThreshold = 2
		probe.InitialDelay = 0 * time.Second
		probe.Period = 2 * time.Second
		probe.Timeout = 1 * time.Second
		probe.Clock = clock

		probe.Start()
		gosched()

		timeTravel(clock, 2, time.Second)

		healthy, err := probe.Healthy()
		require.False(t, healthy)
		require.NoError(t, err)
	})
	t.Run("healthy_check", func(t *testing.T) {
		clock := clock.NewMock()
		check := newMockCheck(clock, 0*time.Second, true, nil)
		multicheck := newMockMultiCheck().Add(check).Add(check)

		probe := newLongLivedProbe(multicheck)
		probe.SuccessThreshold = 2
		probe.FailureThreshold = 2
		probe.InitialDelay = 0 * time.Second
		probe.Period = 2 * time.Second
		probe.Clock = clock

		probe.Start()
		gosched()

		timeTravel(clock, 2, time.Second)

		healthy, err := probe.Healthy()
		require.True(t, healthy)
		require.NoError(t, err)
	})
	t.Run("failed_then_healthy", func(t *testing.T) {
		clock := clock.NewMock()
		multicheck := newMockMultiCheck().
			Add(newMockCheck(clock, 0*time.Second, false, nil)).
			Add(newMockCheck(clock, 0*time.Second, true, nil)).
			Add(newMockCheck(clock, 0*time.Second, true, nil))

		probe := newLongLivedProbe(multicheck)
		probe.SuccessThreshold = 2
		probe.FailureThreshold = 2
		probe.InitialDelay = 0 * time.Second
		probe.Period = 2 * time.Second
		probe.Clock = clock

		probe.Start()
		gosched()

		timeTravel(clock, 5, time.Second)

		healthy, err := probe.Healthy()
		require.True(t, healthy)
		require.NoError(t, err)
	})
	t.Run("healthy_then_failed", func(t *testing.T) {
		clock := clock.NewMock()
		multicheck := newMockMultiCheck().
			Add(newMockCheck(clock, 0*time.Second, true, nil)).
			Add(newMockCheck(clock, 0*time.Second, false, nil)).
			Add(newMockCheck(clock, 0*time.Second, false, nil))

		probe := newLongLivedProbe(multicheck)
		probe.SuccessThreshold = 2
		probe.FailureThreshold = 2
		probe.InitialDelay = 0 * time.Second
		probe.Period = 2 * time.Second
		probe.Clock = clock

		probe.Start()
		gosched()

		timeTravel(clock, 5, time.Second)

		healthy, err := probe.Healthy()
		require.False(t, healthy)
		require.NoError(t, err)
	})
	t.Run("stop_probe", func(t *testing.T) {
		clock := clock.NewMock()
		multicheck := newMockMultiCheck().
			Add(newMockCheck(clock, 0*time.Second, true, nil)).
			Add(newMockCheck(clock, 0*time.Second, true, nil)).
			Add(newMockCheck(clock, 0*time.Second, true, nil))

		probe := newLongLivedProbe(multicheck)
		probe.InitialDelay = 2 * time.Second
		probe.Period = 1 * time.Second
		probe.Clock = clock

		probe.Start()
		gosched()

		for i := 0; i < 4; i++ {
			clock.Add(1 * time.Second)
			gosched()
		}

		probe.Stop()

		// Since the probe is stopped the next check that would panic will not
		// be run.
		timeTravel(clock, 5, time.Second)

		running := probe.Running()
		require.False(t, running)
	})
}
