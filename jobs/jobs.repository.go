package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

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

	QueueKPIs struct {
		PendingJobsPerType map[string]int
		PendingJobs        int
		FailedJobs         int
		AvailableWorkers   int
		AverageTimePerJob  time.Duration
	}

	// Repository manages the data access to the underlying Jobs implementation.
	Repository interface {
		PendingJobs(ctx context.Context, queue string) ([]PendingJob, error)
		QueueKPIs(ctx context.Context, queue string) (QueueKPIs, error)
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

	return jobsToDomain(jobs), nil
}

func jobsToDomain(j []models.GueJob) []PendingJob {
	jobs := make([]PendingJob, len(j))

	for i := 0; i < len(j); i++ {
		jobs[i] = jobToDomain(j[i])
	}

	return jobs
}

func jobToDomain(job models.GueJob) PendingJob {
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

func (repo *PostgresJobsRepository) QueueKPIs(ctx context.Context, queue string) (QueueKPIs, error) {
	var kpis QueueKPIs

	kpis.AvailableWorkers = 10

	jp, err := repo.db.ConnOrTX(ctx).StatsPendingJobs(ctx, queue)
	if err != nil {
		return QueueKPIs{}, fmt.Errorf("%w: could not query pending jobs: %v", ErrQueryFailed, err)
	}

	kpis.PendingJobs = int(jp)

	jf, err := repo.db.ConnOrTX(ctx).StatsFailedJobs(ctx, queue)
	if err != nil {
		return QueueKPIs{}, fmt.Errorf("%w: could not query failed jobs: %v", ErrQueryFailed, err)
	}

	kpis.FailedJobs = int(jf)

	avg, err := repo.db.ConnOrTX(ctx).StatsAvgDurationOfJobs(ctx, queue)
	if err != nil && !errors.As(err, &pgx.ScanArgError{}) { //nolint:exhaustruct // Scan() fails if history table is empty
		return QueueKPIs{}, fmt.Errorf("%w: could not query average job durration: %v", ErrQueryFailed, err)
	}

	// StatsAvgDurationOfJobs returns microseconds but time.Duration() accepts ns.
	kpis.AverageTimePerJob = time.Duration(avg) * time.Microsecond

	nt, err := repo.db.ConnOrTX(ctx).StatsPendingJobsPerType(ctx, queue)
	if err != nil {
		return QueueKPIs{}, fmt.Errorf("%w: cound not query pending job_types: %v", ErrQueryFailed, err)
	}

	kpis.PendingJobsPerType = pendingJobTypesToDomain(nt)

	return kpis, nil
}

func pendingJobTypesToDomain(jobTypes []models.StatsPendingJobsPerTypeRow) map[string]int {
	ret := map[string]int{}

	for _, v := range jobTypes {
		ret[v.JobType] = int(v.Count)
	}

	return ret
}
