package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/go-arrower/arrower/jobs/models"
	"github.com/go-arrower/arrower/postgres"
)

var (
	ErrQueryFailed  = errors.New("query failed")
	ErrDeleteFailed = fmt.Errorf("%w: could not delete job. it might be processing already", ErrQueryFailed)
)

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
		ProcessedJobs      int
		AvailableWorkers   int
		AverageTimePerJob  time.Duration
	}

	WorkerPool struct {
		LastSeen time.Time
		ID       string
		Queue    string
		Workers  int
	}

	// Repository manages the data access to the underlying Jobs implementation.
	Repository interface {
		Queues(ctx context.Context) ([]string, error)
		PendingJobs(ctx context.Context, queue string) ([]PendingJob, error)
		QueueKPIs(ctx context.Context, queue string) (QueueKPIs, error)
		WorkerPools(ctx context.Context) ([]WorkerPool, error)
		RegisterWorkerPool(ctx context.Context, wp WorkerPool) error
		Delete(ctx context.Context, jobID string) error
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

	jt, err := repo.db.ConnOrTX(ctx).StatsProcessedJobs(ctx, queue)
	if err != nil {
		return QueueKPIs{}, fmt.Errorf("%w: could not query processed jobs: %v", ErrQueryFailed, err)
	}

	kpis.ProcessedJobs = int(jt)

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

	w, err := repo.db.ConnOrTX(ctx).StatsQueueWorkerPoolSize(ctx, queue)
	if err != nil {
		return QueueKPIs{}, fmt.Errorf("%w: could not query total queue worker size: %v", ErrQueryFailed, err)
	}

	kpis.AvailableWorkers = int(w)

	return kpis, nil
}

func pendingJobTypesToDomain(jobTypes []models.StatsPendingJobsPerTypeRow) map[string]int {
	ret := map[string]int{}

	for _, v := range jobTypes {
		ret[v.JobType] = int(v.Count)
	}

	return ret
}

func (repo *PostgresJobsRepository) WorkerPools(ctx context.Context) ([]WorkerPool, error) {
	w, err := repo.db.ConnOrTX(ctx).GetWorkerPools(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQueryFailed, err)
	}

	return workersToDomain(w), nil
}

func workersToDomain(w []models.GueJobsWorkerPool) []WorkerPool {
	workers := make([]WorkerPool, len(w))

	for i, w := range w {
		workers[i] = WorkerPool{
			ID:       w.ID,
			Queue:    w.Queue,
			Workers:  int(w.Workers),
			LastSeen: w.UpdatedAt.Time,
		}
	}

	return workers
}

func (repo *PostgresJobsRepository) RegisterWorkerPool(ctx context.Context, wp WorkerPool) error {
	err := repo.db.ConnOrTX(ctx).UpsertWorkerToPool(ctx, models.UpsertWorkerToPoolParams{
		ID:        wp.ID,
		Queue:     wp.Queue,
		Workers:   int16(wp.Workers),
		UpdatedAt: pgtype.Timestamptz{Time: wp.LastSeen, Valid: true, InfinityModifier: pgtype.Finite},
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrQueryFailed, err)
	}

	return nil
}

// Delete attempts to delete a Job with the given jobID.
// If a Job is currently processed by a worker, the row in the db gets locked until the worker succeeds or fails.
// In this case the delete command would block until the lock is freed (which potentially could take a long time).
//
// Delete will time out after one second, assuming that if the database needs longer to execute the query, it means the
// row is locked.
func (repo *PostgresJobsRepository) Delete(ctx context.Context, jobID string) error {
	ctx, _ = context.WithTimeout(ctx, time.Second)

	err := repo.db.ConnOrTX(ctx).DeleteJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeleteFailed, err)
	}

	return nil
}
