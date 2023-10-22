package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"

	"github.com/go-arrower/arrower/jobs/models"
	"github.com/go-arrower/arrower/postgres"
)

var (
	ErrQueryFailed      = errors.New("query failed")
	ErrJobLockedAlready = fmt.Errorf("%w: job might be processing already", ErrQueryFailed)
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
		RunJobAt(ctx context.Context, jobID string, runAt time.Time) error
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
		return nil, fmt.Errorf("%w: %v", ErrQueryFailed, err) //nolint:errorlint // prevent err in api
	}

	return q, nil
}

func (repo *PostgresJobsRepository) PendingJobs(ctx context.Context, queue string) ([]PendingJob, error) {
	jobs, err := repo.db.ConnOrTX(ctx).GetPendingJobs(ctx, queue)
	if err != nil {
		return nil, fmt.Errorf("%w: could not get pending jobs: %v", ErrQueryFailed, err) //nolint:errorlint,lll // prevent err in api
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

func (repo *PostgresJobsRepository) QueueKPIs(ctx context.Context, queue string) (QueueKPIs, error) { //nolint:funlen
	var kpis QueueKPIs

	group, newCtx := errgroup.WithContext(ctx)

	group.Go(func() error {
		jp, err := repo.db.ConnOrTX(newCtx).StatsPendingJobs(newCtx, queue)
		if err != nil {
			return fmt.Errorf("%w: could not query pending jobs: %v", ErrQueryFailed, err) //nolint:errorlint,lll // prevent err in api
		}

		kpis.PendingJobs = int(jp)

		return nil
	})

	group.Go(func() error {
		jf, err := repo.db.ConnOrTX(newCtx).StatsFailedJobs(newCtx, queue)
		if err != nil {
			return fmt.Errorf("%w: could not query failed jobs: %v", ErrQueryFailed, err) //nolint:errorlint,lll // prevent err in api
		}

		kpis.FailedJobs = int(jf)

		return nil
	})

	group.Go(func() error {
		jt, err := repo.db.ConnOrTX(newCtx).StatsProcessedJobs(newCtx, queue)
		if err != nil {
			return fmt.Errorf("%w: could not query processed jobs: %v", ErrQueryFailed, err) //nolint:errorlint,lll // prevent err in api
		}

		kpis.ProcessedJobs = int(jt)

		return nil
	})

	group.Go(func() error {
		avg, err := repo.db.ConnOrTX(newCtx).StatsAvgDurationOfJobs(newCtx, queue)
		if err != nil && !errors.As(err, &pgx.ScanArgError{}) { //nolint:exhaustruct // Scan() fails if history table is empty
			return fmt.Errorf("%w: could not query average job durration: %v", ErrQueryFailed, err) //nolint:errorlint,lll // prevent err in api
		}

		// StatsAvgDurationOfJobs returns microseconds but time.Duration() accepts ns.
		kpis.AverageTimePerJob = time.Duration(avg) * time.Microsecond

		return nil
	})

	group.Go(func() error {
		nt, err := repo.db.ConnOrTX(newCtx).StatsPendingJobsPerType(newCtx, queue)
		if err != nil {
			return fmt.Errorf("%w: cound not query pending job_types: %v", ErrQueryFailed, err) //nolint:errorlint,lll // prevent err in api
		}

		kpis.PendingJobsPerType = pendingJobTypesToDomain(nt)

		return nil
	})

	group.Go(func() error {
		w, err := repo.db.ConnOrTX(newCtx).StatsQueueWorkerPoolSize(newCtx, queue)
		if err != nil {
			return fmt.Errorf("%w: could not query total queue worker size: %v", ErrQueryFailed, err) //nolint:errorlint,lll // prevent err in api
		}

		kpis.AvailableWorkers = int(w)

		return nil
	})

	err := group.Wait()

	return kpis, err //nolint:wrapcheck // false positive, as error is nil or the first failing goroutine
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
		return nil, fmt.Errorf("%w: %v", ErrQueryFailed, err) //nolint:errorlint // prevent err in api
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
		return fmt.Errorf("%w: %v", ErrQueryFailed, err) //nolint:errorlint // prevent err in api
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
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	err := repo.db.ConnOrTX(ctx).DeleteJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("%w: could not delete: %v", ErrJobLockedAlready, err) //nolint:errorlint // prevent err in api
	}

	return nil
}

// RunJobAt attempts to reschedule a Job with the given runAt time.
//
// RunJobAt will time out after one second, assuming that if the database needs longer to execute the query,
// it means the row is locked by an active worker processing the job.
func (repo *PostgresJobsRepository) RunJobAt(ctx context.Context, jobID string, runAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	err := repo.db.ConnOrTX(ctx).UpdateRunAt(ctx, models.UpdateRunAtParams{
		JobID: jobID,
		RunAt: pgtype.Timestamptz{Time: runAt, Valid: true, InfinityModifier: pgtype.Finite},
	})
	if err != nil {
		return fmt.Errorf("%w: could not reschedule: %v", ErrJobLockedAlready, err) //nolint:errorlint,lll // prevent err in api
	}

	return nil
}

func NewTracedJobsRepository(repo Repository) *TracedJobsRepository {
	return &TracedJobsRepository{repo: repo}
}

type TracedJobsRepository struct {
	repo Repository
}

var _ Repository = (*TracedJobsRepository)(nil)

func (repo *TracedJobsRepository) Queues(ctx context.Context) ([]string, error) {
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("arrower.jobs").
		Start(ctx, "repo", trace.WithAttributes(attribute.String("method", "Queues")))
	defer span.End()

	return repo.repo.Queues(ctx) //nolint:wrapcheck // this is decorator
}

func (repo *TracedJobsRepository) PendingJobs(ctx context.Context, queue string) ([]PendingJob, error) {
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("arrower.jobs").
		Start(ctx, "repo", trace.WithAttributes(
			attribute.String("method", "PendingJobs"),
			attribute.String("queue", queue),
		))
	defer span.End()

	return repo.repo.PendingJobs(ctx, queue) //nolint:wrapcheck // this is decorator
}

func (repo *TracedJobsRepository) QueueKPIs(ctx context.Context, queue string) (QueueKPIs, error) {
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("arrower.jobs").
		Start(ctx, "repo", trace.WithAttributes(
			attribute.String("method", "QueueKPIs"),
			attribute.String("queue", queue),
		))
	defer span.End()

	return repo.repo.QueueKPIs(ctx, queue) //nolint:wrapcheck // this is decorator
}

func (repo *TracedJobsRepository) WorkerPools(ctx context.Context) ([]WorkerPool, error) {
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("arrower.jobs").
		Start(ctx, "repo", trace.WithAttributes(attribute.String("method", "WorkerPools")))
	defer span.End()

	return repo.repo.WorkerPools(ctx) //nolint:wrapcheck // this is decorator
}

func (repo *TracedJobsRepository) RegisterWorkerPool(ctx context.Context, wp WorkerPool) error {
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("arrower.jobs").
		Start(ctx, "repo", trace.WithAttributes(
			attribute.String("method", "RegisterWorkerPool"),
			attribute.String("wp_id", wp.ID),
			attribute.String("wp_queue", wp.Queue),
		))
	defer span.End()

	return repo.repo.RegisterWorkerPool(ctx, wp) //nolint:wrapcheck // this is decorator
}

func (repo *TracedJobsRepository) Delete(ctx context.Context, jobID string) error {
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("arrower.jobs").
		Start(ctx, "repo", trace.WithAttributes(
			attribute.String("method", "Delete"),
			attribute.String("jobID", jobID),
		))
	defer span.End()

	return repo.repo.Delete(ctx, jobID) //nolint:wrapcheck // this is decorator
}

func (repo *TracedJobsRepository) RunJobAt(ctx context.Context, jobID string, runAt time.Time) error {
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("arrower.jobs").
		Start(ctx, "repo", trace.WithAttributes(
			attribute.String("method", "RunJobAt"),
			attribute.String("jobID", jobID),
			attribute.String("runAt", runAt.String()),
		))
	defer span.End()

	return repo.repo.RunJobAt(ctx, jobID, runAt) //nolint:wrapcheck // this is decorator
}
