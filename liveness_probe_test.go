package controller

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/require"
)

func TestLivenessProbe(t *testing.T) {
	t.Run("healthy_start", func(t *testing.T) {
		clock := clock.NewMock()
		check := newMockCheck(clock, 0*time.Second, false, nil)
		multicheck := newMockMultiCheck().Add(check)

		probe := NewLivenessProbe(multicheck)
		healthy, err := probe.Healthy()
		require.True(t, healthy)
		require.NoError(t, err)
	})
}
