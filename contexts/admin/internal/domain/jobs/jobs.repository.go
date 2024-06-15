package jobs

import (
	"context"
	"time"
)

// Repository manages the data access to the underlying jobs' implementation.
type Repository interface {
	Queues(ctx context.Context) (QueueNames, error)
	PendingJobs(ctx context.Context, queue QueueName) ([]PendingJob, error)
	QueueKPIs(ctx context.Context, queue QueueName) (QueueKPIs, error)
	Delete(ctx context.Context, jobID string) error
	RunJobAt(ctx context.Context, jobID string, runAt time.Time) error
	WorkerPools(ctx context.Context) ([]WorkerPool, error)
	FinishedJobs(ctx context.Context, f Filter) ([]PendingJob, error)
	FinishedJobsTotal(ctx context.Context, f Filter) (int64, error)
}

type Filter struct {
	Queue   QueueName
	JobType JobType
}
