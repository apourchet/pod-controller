package controller

import (
	"time"

	"github.com/benbjohnson/clock"
)

func NewReadinessProbe(check Check) *LongLivedProbe {
	return &LongLivedProbe{
		// TODO: Check defaults
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
