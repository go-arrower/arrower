package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-arrower/arrower/jobs/models"
	"github.com/go-arrower/arrower/postgres"
)

var ErrQueryFailed = errors.New("query failed")

type (
	PendingJob struct {
		CreatedAt  time.Time
		UpdatedAt  time.Time
		RunAt      time.Time
		ID         string
		Type       string
		Queue      string
		Payload    string
		LastError  string
		ErrorCount int32
		Priority   int16
	}

	// Repository manages the data access to the underlying Jobs implementation.
	Repository interface {
		PendingJobs(ctx context.Context, queue string) ([]PendingJob, error)
	}
)

type PostgresJobsRepository struct {
	db postgres.BaseRepository[*models.Queries]
}

var _ Repository = (*PostgresJobsRepository)(nil)

func NewPostgresJobsRepository(queries *models.Queries) *PostgresJobsRepository {
	return &PostgresJobsRepository{db: postgres.NewPostgresBaseRepository(queries)}
}

func (repo *PostgresJobsRepository) Queues(ctx context.Context) ([]string, error) {
	q, err := repo.db.ConnOrTX(ctx).GetQueues(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQueryFailed, err)
	}

	return q, nil
}

func (repo *PostgresJobsRepository) PendingJobs(ctx context.Context, queue string) ([]PendingJob, error) {
	jobs, err := repo.db.ConnOrTX(ctx).GetPendingJobs(ctx, queue)
	if err != nil {
		return nil, fmt.Errorf("%w: could not get pending jobs: %v", ErrQueryFailed, err)
	}

	return jobsFromModelToDomain(jobs), nil
}

func jobsFromModelToDomain(j []models.GueJob) []PendingJob {
	jobs := make([]PendingJob, len(j))

	for i := 0; i < len(j); i++ {
		jobs[i] = jobFromModelToDomain(j[i])
	}

	return jobs
}

func jobFromModelToDomain(job models.GueJob) PendingJob {
	return PendingJob{
		ID:         job.JobID,
		Priority:   job.Priority,
		RunAt:      job.RunAt.Time,
		Type:       job.JobType,
		Payload:    string(job.Args),
		ErrorCount: job.ErrorCount,
		LastError:  job.LastError.String,
		Queue:      job.Queue,
		CreatedAt:  job.CreatedAt.Time,
		UpdatedAt:  job.UpdatedAt.Time,
	}
}
