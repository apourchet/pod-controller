package controller

import (
	"time"

	"github.com/benbjohnson/clock"
)

func NewLivenessProbe(check Check) *LongLivedProbe {
	return &LongLivedProbe{
		// TODO: Check defaults in the k8s reference.
		BaseProbe: BaseProbe{
			InitialDelay:     1 * time.Second,
			Period:           5 * time.Second,
			Timeout:          1 * time.Second,
			SuccessThreshold: 1,
			FailureThreshold: 1,
		},
		Check:     check,
		Clock:     clock.New(),
		isHealthy: true,
	}
}
