package models

import "testing"

func TestCanTransition(t *testing.T) {
	valid := []struct{ from, to JobStatus }{
		{JobStatusQueued, JobStatusClaimed},
		{JobStatusQueued, JobStatusScheduled},
		{JobStatusQueued, JobStatusCancelled},
		{JobStatusScheduled, JobStatusQueued},
		{JobStatusClaimed, JobStatusRunning},
		{JobStatusClaimed, JobStatusQueued},
		{JobStatusRunning, JobStatusCompleted},
		{JobStatusRunning, JobStatusFailed},
		{JobStatusRunning, JobStatusQueued},
		{JobStatusRunning, JobStatusScheduled},
	}
	for _, c := range valid {
		if !CanTransition(c.from, c.to) {
			t.Errorf("expected transition %s -> %s to be valid", c.from, c.to)
		}
	}

	invalid := []struct{ from, to JobStatus }{
		{JobStatusCompleted, JobStatusRunning},
		{JobStatusFailed, JobStatusQueued},
		{JobStatusCancelled, JobStatusRunning},
		{JobStatusQueued, JobStatusCompleted}, // cannot complete before running
		{JobStatusClaimed, JobStatusCompleted},
		{JobStatusQueued, JobStatusRunning}, // must go through claimed
	}
	for _, c := range invalid {
		if CanTransition(c.from, c.to) {
			t.Errorf("expected transition %s -> %s to be invalid", c.from, c.to)
		}
	}
}

func TestTransition_NoSelfTransition(t *testing.T) {
	if err := Transition(JobStatusQueued, JobStatusQueued); err == nil {
		t.Error("expected self transition to be rejected")
	}
}

func TestTransition_Valid(t *testing.T) {
	if err := Transition(JobStatusRunning, JobStatusCompleted); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestJobStatus_IsTerminal(t *testing.T) {
	for _, s := range []JobStatus{JobStatusCompleted, JobStatusFailed, JobStatusCancelled} {
		if !s.IsTerminal() {
			t.Errorf("expected %s to be terminal", s)
		}
	}
	for _, s := range []JobStatus{JobStatusQueued, JobStatusScheduled, JobStatusClaimed, JobStatusRunning} {
		if s.IsTerminal() {
			t.Errorf("expected %s to be non-terminal", s)
		}
	}
}
