package models

import "fmt"

// transitionMap defines the valid job lifecycle transitions.
// Centralized here so no handler scatters ad-hoc state changes.
var transitionMap = map[JobStatus]map[JobStatus]bool{
	JobStatusQueued: {
		JobStatusScheduled: true, // delayed/scheduled jobs wait for available_at
		JobStatusClaimed:   true,
		JobStatusCancelled: true,
		JobStatusFailed:    true, // cancelled/failed before claim is allowed
	},
	JobStatusScheduled: {
		JobStatusQueued:    true, // scheduler promotes when available_at <= now
		JobStatusCancelled: true,
	},
	JobStatusClaimed: {
		JobStatusRunning: true, // worker confirms execution started
		JobStatusQueued:  true, // lease expired / recovered
		JobStatusFailed:  true,
	},
	JobStatusRunning: {
		JobStatusCompleted: true, // success
		JobStatusFailed:    true, // permanent failure -> DLQ
		JobStatusQueued:    true, // retry scheduled (attempt < max_attempts)
		JobStatusScheduled: true, // retry scheduled with a future backoff delay
	},
	// Terminal states have no outgoing transitions.
	JobStatusCompleted: {},
	JobStatusFailed:    {},
	JobStatusCancelled: {},
}

// CanTransition reports whether a job may move from one status to another.
func CanTransition(from, to JobStatus) bool {
	if from == to {
		return false
	}
	allowed, ok := transitionMap[from]
	if !ok {
		return false
	}
	return allowed[to]
}

// Transition validates and applies a state change.
func Transition(from, to JobStatus) error {
	if !CanTransition(from, to) {
		return fmt.Errorf("invalid state transition: %s -> %s", from, to)
	}
	return nil
}
