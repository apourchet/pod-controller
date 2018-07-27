package controller

import (
	"fmt"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/require"
)

func TestExitProbe(t *testing.T) {
	t.Run("unhealthy_check", func(t *testing.T) {
		clock := clock.NewMock()
		check := newMockCheck(clock, 0*time.Second, false, nil)
		probe := NewExitProbe(check)
		probe.Start()
		gosched()

		for probe.Running() {
			clock.Add(1 * time.Millisecond)
			gosched()
		}

		healthy, err := probe.Healthy()
		require.False(t, healthy)
		require.NoError(t, err)
	})
	t.Run("errored_check", func(t *testing.T) {
		clock := clock.NewMock()
		check := newMockCheck(clock, 0*time.Second, false, fmt.Errorf("ERROR"))
		probe := NewExitProbe(check)
		probe.Start()
		gosched()

		for probe.Running() {
			clock.Add(1 * time.Millisecond)
			gosched()
		}

		healthy, err := probe.Healthy()
		require.False(t, healthy)
		require.Error(t, err)
	})
	t.Run("healthy_check", func(t *testing.T) {
		clock := clock.NewMock()
		check := newMockCheck(clock, 0*time.Second, true, nil)
		probe := NewExitProbe(check)
		probe.Start()
		gosched()

		for probe.Running() {
			clock.Add(1 * time.Millisecond)
			gosched()
		}

		healthy, err := probe.Healthy()
		require.True(t, healthy)
		require.NoError(t, err)
	})
	t.Run("is_running", func(t *testing.T) {
		clock := clock.NewMock()
		check := newMockCheck(clock, 5*time.Second, true, nil)
		probe := NewExitProbe(check)
		probe.Start()
		gosched()
		require.True(t, probe.Running())
		for i := 0; i < 10; i++ {
			clock.Add(1 * time.Second)
			gosched()
		}
		require.False(t, probe.Running())
	})
	t.Run("healthy_while_running", func(t *testing.T) {
		clock := clock.NewMock()
		check := newMockCheck(clock, 10*time.Second, true, nil)
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
