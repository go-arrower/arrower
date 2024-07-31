package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/go-arrower/arrower/postgres"
)

var ErrJobLockedAlready = fmt.Errorf("%w: job might be processing already", postgres.ErrQueryFailed)

// Repository manages the data access to the underlying jobs' implementation.
type Repository interface {
	FindAllQueueNames(ctx context.Context) (QueueNames, error)
	PendingJobs(ctx context.Context, queue QueueName) ([]Job, error)
	QueueKPIs(ctx context.Context, queue QueueName) (QueueKPIs, error)
	DeleteByID(ctx context.Context, jobID string) error
	RunJobAt(ctx context.Context, jobID string, runAt time.Time) error
	WorkerPools(ctx context.Context) ([]WorkerPool, error)
	Schedules(ctx context.Context) ([]Schedule, error)
	FinishedJobs(ctx context.Context, f Filter) ([]Job, error)
	FinishedJobsTotal(ctx context.Context, f Filter) (int64, error)
}

type Filter struct {
	Queue   QueueName
	JobType JobType
}
