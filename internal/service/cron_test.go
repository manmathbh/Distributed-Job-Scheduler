package service

import (
	"testing"
	"time"
)

func TestNextCronRun(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	everyMinute, err := NextCronRun("* * * * *", "UTC", now)
	if err != nil {
		t.Fatal(err)
	}
	if !everyMinute.After(now) {
		t.Errorf("next run must be after now")
	}

	// 0 0 * * * = daily at midnight.
	daily, err := NextCronRun("0 0 * * *", "UTC", now)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	if !daily.Equal(want) {
		t.Errorf("daily next run = %v, want %v", daily, want)
	}

	// Invalid expression.
	if _, err := NextCronRun("invalid", "UTC", now); err == nil {
		t.Error("expected error for invalid cron")
	}
}
