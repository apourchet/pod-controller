package controller

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/require"
)

func TestReadinessProbe(t *testing.T) {
	t.Run("unhealthy_start", func(t *testing.T) {
		clock := clock.NewMock()
		check := newMockCheck(clock, 0*time.Second, false, nil)
		multicheck := newMockMultiCheck().Add(check)

		probe := NewReadinessProbe(multicheck)
		healthy, err := probe.Healthy()
		require.False(t, healthy)
		require.NoError(t, err)
	})
}
