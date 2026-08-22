package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/manmathbh/distributed-job-scheduler/internal/models"
	"github.com/manmathbh/distributed-job-scheduler/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFireDueScheduledJob_AdvancesAndClaims(t *testing.T) {
	s := store.NewMemoryStore()
	ctx := context.Background()

	now := time.Now()
	due := now.Add(-time.Minute)
	sj := &models.ScheduledJob{
		ID: "sched-1", ProjectID: "p", QueueID: "q", Name: "n",
		CronExpr: "* * * * *", Timezone: "UTC", Payload: []byte(`{}`),
		Enabled: true, NextRunAt: &due, CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, s.CreateScheduledJob(ctx, sj))

	calls := 0
	fired, err := s.FireDueScheduledJob(ctx, now, func(sj *models.ScheduledJob) (time.Time, error) {
		calls++
		return now.Add(time.Hour), nil
	})
	require.NoError(t, err)
	require.NotNil(t, fired)
	assert.Equal(t, "sched-1", fired.ID)
	assert.Equal(t, 1, calls)
	assert.Equal(t, now.Add(time.Hour), *fired.NextRunAt)
	assert.Equal(t, now, *fired.LastRunAt)

	// Not due anymore.
	again, err := s.FireDueScheduledJob(ctx, now, func(*models.ScheduledJob) (time.Time, error) { return now, nil })
	require.NoError(t, err)
	assert.Nil(t, again, "schedule must not fire twice")
}

func TestFireDueScheduledJob_SkipsDisabled(t *testing.T) {
	s := store.NewMemoryStore()
	ctx := context.Background()
	now := time.Now()
	due := now.Add(-time.Minute)
	sj := &models.ScheduledJob{
		ID: "sched-1", ProjectID: "p", QueueID: "q", CronExpr: "* * * * *",
		Timezone: "UTC", Enabled: false, NextRunAt: &due, CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, s.CreateScheduledJob(ctx, sj))

	fired, err := s.FireDueScheduledJob(ctx, now, func(*models.ScheduledJob) (time.Time, error) { return now, nil })
	require.NoError(t, err)
	assert.Nil(t, fired)
}
