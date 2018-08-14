package controller

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	oci "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/require"
)

func TestController(t *testing.T) {
	t.Run("single_healthy", func(t *testing.T) {
		spec := PodSpec{
			InitContainers: []oci.Spec{
				{
					Process: &oci.Process{
						Args: []string{"true"},
					},
				},
			},
			Containers: []ContainerSpec{
				{
					Name: "main",
					Spec: oci.Spec{
						Process: &oci.Process{
							Args: []string{"sleep", "1000"},
						},
					},
					LivenessProbe: LivenessProbeSpec{
						ProbeSpec{
							PeriodSeconds:    5,
							TimeoutSeconds:   1,
							SuccessThreshold: 1,
							FailureThreshold: 1,
						},
					},
					ReadinessProbe: ReadinessProbeSpec{
						ProbeSpec{
							PeriodSeconds:    5,
							TimeoutSeconds:   1,
							SuccessThreshold: 1,
							FailureThreshold: 1,
						},
					},
				},
			},
		}
		controller, err := NewPodController(spec, "./bins/testing.so")
		require.NoError(t, err)

		clock := clock.NewMock()
		controller.Clock = clock
		err = controller.Start()
		require.NoError(t, err)

		for i := 0; i < 10; i++ {
			clock.Add(1 * time.Second)
			gosched()
		}

		healthy := controller.Healthy()
		require.True(t, healthy)

		statuses := controller.Status()
		require.Lenf(t, statuses, 1, "should only have 1 status")
		require.Equal(t, Healthy, statuses[0].LastState())
	})
	t.Run("single_unhealthy", func(t *testing.T) {
		spec := PodSpec{
			Containers: []ContainerSpec{
				{
					Name: "main",
					Spec: oci.Spec{
						Process: &oci.Process{
							Args: []string{"false"},
						},
					},
					LivenessProbe: LivenessProbeSpec{
						ProbeSpec{
							PeriodSeconds:    5,
							TimeoutSeconds:   1,
							SuccessThreshold: 1,
							FailureThreshold: 1,
						},
					},
					ReadinessProbe: ReadinessProbeSpec{
						ProbeSpec{
							PeriodSeconds:    5,
							TimeoutSeconds:   1,
							SuccessThreshold: 1,
							FailureThreshold: 1,
						},
					},
				},
			},
		}
		controller, err := NewPodController(spec, "./bins/testing.so")
		require.NoError(t, err)

		clock := clock.NewMock()
		controller.Clock = clock
		err = controller.Start()
		require.NoError(t, err)

		for i := 0; i < 10; i++ {
			clock.Add(1 * time.Second)
			gosched()
		}

		healthy := controller.Healthy()
		require.False(t, healthy)

		statuses := controller.Status()
		require.Lenf(t, statuses, 1, "should only have 1 status")
		require.Equal(t, Failed, statuses[0].LastState())
	})
	t.Run("single_healthy_then_unhealthy", func(t *testing.T) {
		// TODO: write tests
	})
	t.Run("double_healthy", func(t *testing.T) {
		// TODO: write tests
	})
	t.Run("double_healthy_and_unhealthy", func(t *testing.T) {
		// TODO: write tests
	})
}
