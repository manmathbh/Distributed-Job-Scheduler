package seed

import (
	"context"
	"log"
	"time"

	"github.com/manmathbh/distributed-job-scheduler/internal/models"
	"github.com/manmathbh/distributed-job-scheduler/internal/service"
)

// SeedDemo creates a demo project and queues for a given owner if none exist.
func SeedDemo(ctx context.Context, svc *service.Service, ownerID string) error {
	projects, _, err := svc.ListProjects(ctx, ownerID, 10, "")
	if err != nil {
		return err
	}
	if len(projects) > 0 {
		log.Printf("seed: owner %q already has %d project(s), skipping demo seed", ownerID, len(projects))
		return nil
	}

	p, err := svc.CreateProject(ctx, ownerID, "Demo Project", "Seeded demo project for the dashboard")
	if err != nil {
		return err
	}
	log.Printf("seed: created project %s (%s)", p.Name, p.ID)

	defaultQueue, err := svc.CreateQueue(ctx, p.ID, service.QueueConfig{
		Name:          "default",
		Description:   "Default queue",
		Priority:      0,
		Concurrency:   4,
		RetryStrategy: models.RetryStrategyExponential,
		MaxAttempts:   3,
		InitialDelay:  time.Second,
		MaxDelay:      time.Minute,
		Multiplier:    2,
	})
	if err != nil {
		return err
	}
	log.Printf("seed: created queue %q (%s)", defaultQueue.Name, defaultQueue.ID)

	_, err = svc.CreateQueue(ctx, p.ID, service.QueueConfig{
		Name:          "notifications",
		Description:   "High-priority notifications",
		Priority:      10,
		Concurrency:   2,
		RetryStrategy: models.RetryStrategyLinear,
		MaxAttempts:   5,
		InitialDelay:  2 * time.Second,
		MaxDelay:      30 * time.Second,
		Multiplier:    1,
	})
	if err != nil {
		return err
	}
	log.Printf("seed: created queue %q", "notifications")
	return nil
}
