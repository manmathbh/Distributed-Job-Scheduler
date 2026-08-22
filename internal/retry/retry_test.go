package retry

import (
	"testing"
	"time"

	"github.com/manmathbh/distributed-job-scheduler/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNextDelay_Fixed(t *testing.T) {
	p := Policy{
		Strategy:     models.RetryStrategyFixed,
		MaxAttempts:  5,
		InitialDelay: 3 * time.Second,
		MaxDelay:     60 * time.Second,
	}
	for i := 1; i <= 5; i++ {
		assert.Equal(t, 3*time.Second, p.NextDelay(i), "attempt %d", i)
	}
}

func TestNextDelay_Linear(t *testing.T) {
	p := Policy{
		Strategy:     models.RetryStrategyLinear,
		MaxAttempts:  5,
		InitialDelay: 2 * time.Second,
		MaxDelay:     60 * time.Second,
	}
	expected := []time.Duration{2 * time.Second, 4 * time.Second, 6 * time.Second, 8 * time.Second, 10 * time.Second}
	for i, want := range expected {
		assert.Equal(t, want, p.NextDelay(i+1), "attempt %d", i+1)
	}
}

func TestNextDelay_Exponential(t *testing.T) {
	p := Policy{
		Strategy:     models.RetryStrategyExponential,
		MaxAttempts:  5,
		InitialDelay: 1 * time.Second,
		MaxDelay:     60 * time.Second,
		Multiplier:   2,
	}
	expected := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second}
	for i, want := range expected {
		assert.Equal(t, want, p.NextDelay(i+1), "attempt %d", i+1)
	}
}

func TestNextDelay_ExponentialCapped(t *testing.T) {
	p := Policy{
		Strategy:     models.RetryStrategyExponential,
		MaxAttempts:  10,
		InitialDelay: 10 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2,
	}
	// 10s, 20s, 40s(capped->30s), 80s(capped->30s)...
	assert.Equal(t, 10*time.Second, p.NextDelay(1))
	assert.Equal(t, 20*time.Second, p.NextDelay(2))
	assert.Equal(t, 30*time.Second, p.NextDelay(3))
	assert.Equal(t, 30*time.Second, p.NextDelay(4))
}

func TestShouldRetry(t *testing.T) {
	p := Policy{MaxAttempts: 3}
	assert.True(t, p.ShouldRetry(0))
	assert.True(t, p.ShouldRetry(2))
	assert.False(t, p.ShouldRetry(3))
	assert.False(t, p.ShouldRetry(4))
}

func TestDefault(t *testing.T) {
	p := Default()
	assert.Equal(t, models.RetryStrategyExponential, p.Strategy)
	assert.Equal(t, 3, p.MaxAttempts)
	assert.Equal(t, 1*time.Second, p.InitialDelay)
}
