package retry

import (
	"math"
	"time"

	"github.com/manmathbh/distributed-job-scheduler/internal/models"
)

// Policy is a snapshot of retry configuration used to compute backoff delays.
type Policy struct {
	Strategy     models.RetryStrategy
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// Default returns the default retry policy.
func Default() Policy {
	return Policy{
		Strategy:     models.RetryStrategyExponential,
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     60 * time.Second,
		Multiplier:   2.0,
	}
}

// NextDelay computes the delay before the next attempt (1-indexed attempt
// number, i.e. the attempt that just failed). The result is deterministic and
// bounded by MaxDelay.
func (p Policy) NextDelay(attempt int) time.Duration {
	delay := p.compute(attempt)
	if delay > p.MaxDelay {
		return p.MaxDelay
	}
	if delay < 0 {
		return 0
	}
	return delay
}

func (p Policy) compute(attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}
	switch p.Strategy {
	case models.RetryStrategyFixed:
		return p.InitialDelay
	case models.RetryStrategyLinear:
		return time.Duration(int64(attempt)) * p.InitialDelay
	case models.RetryStrategyExponential:
		mult := p.Multiplier
		if mult < 1 {
			mult = 2.0
		}
		backoff := math.Pow(mult, float64(attempt-1))
		return time.Duration(float64(p.InitialDelay) * backoff)
	default:
		return p.InitialDelay
	}
}

// ShouldRetry reports whether another attempt is allowed after `attempts`
// failed attempts.
func (p Policy) ShouldRetry(attempts int) bool {
	return attempts < p.MaxAttempts
}
