package controller

import (
	"fmt"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/require"
)

func newMockAsyncCheck(clock clock.Clock, startDuration, waitDuration time.Duration, startError, waitError error) *AsyncCheck {
	start := func() error {
		clock.Sleep(startDuration)
		return startError
	}
	wait := func() error {
		clock.Sleep(waitDuration)
		return waitError
	}
	return NewAsyncCheck(start, wait)
}

func TestExitProbe(t *testing.T) {
	t.Run("unhealthy_check", func(t *testing.T) {
		clock := clock.NewMock()
		check := newMockAsyncCheck(clock, 0*time.Second, 0, fmt.Errorf("ERROR"), nil)
		probe := NewExitProbe(check)
		probe.Start()
		gosched()

		for probe.Running() {
			timeTravel(clock, 1, 1*time.Millisecond)
		}

		healthy, err := probe.Healthy()
		require.False(t, healthy)
		require.Error(t, err)
	})
	t.Run("healthy_check", func(t *testing.T) {
		clock := clock.NewMock()
		check := newMockAsyncCheck(clock, 0*time.Second, 0*time.Second, nil, nil)
		probe := NewExitProbe(check)
		probe.Start()
		gosched()

		for probe.Running() {
			timeTravel(clock, 1, 1*time.Millisecond)
		}

		healthy, err := probe.Healthy()
		require.True(t, healthy)
		require.NoError(t, err)
	})
	t.Run("is_running", func(t *testing.T) {
		clock := clock.NewMock()
		check := newMockAsyncCheck(clock, 5*time.Second, 1*time.Second, nil, nil)
		probe := NewExitProbe(check)
		probe.Start()
		gosched()
		require.True(t, probe.Running())
		timeTravel(clock, 10, 1*time.Second)
		require.False(t, probe.Running())
	})
	t.Run("healthy_while_running", func(t *testing.T) {
		clock := clock.NewMock()
		check := newMockAsyncCheck(clock, 10*time.Second, 10*time.Second, nil, nil)
		probe := NewExitProbe(check)
		probe.Start()
		gosched()

		clock.Add(1 * time.Second)
		gosched()

		healthy, err := probe.Healthy()
		require.True(t, healthy)
		require.NoError(t, err)
	})
}
