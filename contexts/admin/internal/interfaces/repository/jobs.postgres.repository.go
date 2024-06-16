package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/go-arrower/arrower/contexts/admin/internal/domain/jobs"
	"github.com/go-arrower/arrower/contexts/admin/internal/interfaces/repository/models"
	"github.com/go-arrower/arrower/postgres"
)

func NewPostgresJobsRepository(pg *pgxpool.Pool) *PostgresJobsRepository {
	return &PostgresJobsRepository{
		postgres.NewPostgresBaseRepository(models.New(pg)),
	}
}

type PostgresJobsRepository struct {
	postgres.BaseRepository[*models.Queries]
}

var _ jobs.Repository = (*PostgresJobsRepository)(nil)

func (repo *PostgresJobsRepository) Queues(ctx context.Context) (jobs.QueueNames, error) {
	queues, err := repo.Conn().GetQueues(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not query queues: %w: %v", postgres.ErrQueryFailed, err) //nolint:errorlint,lll // prevent err in api
	}

	queueNames := make(jobs.QueueNames, len(queues))

	for i, q := range queues {
		if q == "" {
			q = string(jobs.DefaultQueueName)
		}

		queueNames[i] = jobs.QueueName(q)
	}

	return queueNames, nil
}

func (repo *PostgresJobsRepository) PendingJobs(ctx context.Context, queue jobs.QueueName) ([]jobs.Job, error) {
	name := queueNameFromDomain(queue)

	jobs, err := repo.Conn().GetPendingJobs(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("%w: could not get pending jobs: %v", postgres.ErrQueryFailed, err) //nolint:errorlint,lll // prevent err in api
	}

	return jobsToDomain(jobs), nil
}

func jobsToDomain(j []models.ArrowerGueJob) []jobs.Job {
	jobs := make([]jobs.Job, len(j))

	for i := 0; i < len(j); i++ {
		jobs[i] = jobToDomain(j[i])
	}

	return jobs
}

func jobToDomain(job models.ArrowerGueJob) jobs.Job {
	return jobs.Job{
		ID:         job.JobID,
		Priority:   job.Priority,
		RunAt:      job.RunAt.Time,
		Type:       jobs.JobType(job.JobType),
		Payload:    string(job.Args),
		ErrorCount: job.ErrorCount,
		LastError:  job.LastError,
		Queue:      jobs.QueueName(job.Queue),
		CreatedAt:  job.CreatedAt.Time,
		UpdatedAt:  job.UpdatedAt.Time,
	}
}

func (repo *PostgresJobsRepository) QueueKPIs(ctx context.Context, queue jobs.QueueName) (jobs.QueueKPIs, error) { //nolint:funlen,lll
	var kpis jobs.QueueKPIs

	queueName := queueNameFromDomain(queue)

	group, newCtx := errgroup.WithContext(ctx)

	group.Go(func() error {
		jp, err := repo.Conn().StatsPendingJobs(newCtx, queueName)
		if err != nil {
			return fmt.Errorf("%w: could not query pending jobs: %v", postgres.ErrQueryFailed, err) //nolint:errorlint,lll // prevent err in api
		}

		kpis.PendingJobs = int(jp)

		return nil
	})

	group.Go(func() error {
		jf, err := repo.Conn().StatsFailedJobs(newCtx, queueName)
		if err != nil {
			return fmt.Errorf("%w: could not query failed jobs: %v", postgres.ErrQueryFailed, err) //nolint:errorlint,lll // prevent err in api
		}

		kpis.FailedJobs = int(jf)

		return nil
	})

	group.Go(func() error {
		jt, err := repo.Conn().StatsProcessedJobs(newCtx, queueName)
		if err != nil {
			return fmt.Errorf("%w: could not query processed jobs: %v", postgres.ErrQueryFailed, err) //nolint:errorlint,lll // prevent err in api
		}

		kpis.ProcessedJobs = int(jt)

		return nil
	})

	group.Go(func() error {
		avg, err := repo.Conn().StatsAvgDurationOfJobs(newCtx, queueName)
		if err != nil && !errors.As(err, &pgx.ScanArgError{}) { //nolint:exhaustruct // Scan() fails if history table is empty
			return fmt.Errorf("%w: could not query average job durration: %v", postgres.ErrQueryFailed, err) //nolint:errorlint,lll // prevent err in api
		}

		// StatsAvgDurationOfJobs returns microseconds but time.Duration() accepts ns.
		kpis.AverageTimePerJob = time.Duration(avg) * time.Microsecond

		return nil
	})

	group.Go(func() error {
		nt, err := repo.Conn().StatsPendingJobsPerType(newCtx, queueName)
		if err != nil {
			return fmt.Errorf("%w: cound not query pending job_types: %v", postgres.ErrQueryFailed, err) //nolint:errorlint,lll // prevent err in api
		}

		kpis.PendingJobsPerType = pendingJobTypesToDomain(nt)

		return nil
	})

	group.Go(func() error {
		w, err := repo.Conn().StatsQueueWorkerPoolSize(newCtx, queueName)
		if err != nil {
			return fmt.Errorf("%w: could not query total queue worker size: %v", postgres.ErrQueryFailed, err) //nolint:errorlint,lll // prevent err in api
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

// Delete attempts to delete a Job with the given jobID.
// If a Job is currently processed by a worker, the row in the db gets locked until the worker succeeds or fails.
// In this case the delete command would block until the lock is freed (which potentially could take a long time).
//
// Delete will time out after one second, assuming that if the database needs longer to execute the query, it means the
// row is locked.
func (repo *PostgresJobsRepository) Delete(ctx context.Context, jobID string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	err := repo.ConnOrTX(ctx).DeleteJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("%w: could not delete: %v", jobs.ErrJobLockedAlready, err) //nolint:errorlint // prevent err in api
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

	err := repo.ConnOrTX(ctx).UpdateRunAt(ctx, models.UpdateRunAtParams{
		JobID: jobID,
		RunAt: pgtype.Timestamptz{Time: runAt, Valid: true, InfinityModifier: pgtype.Finite},
	})
	if err != nil {
		return fmt.Errorf("%w: could not reschedule: %v", jobs.ErrJobLockedAlready, err) //nolint:errorlint,lll // prevent err in api
	}

	return nil
}

func (repo *PostgresJobsRepository) WorkerPools(ctx context.Context) ([]jobs.WorkerPool, error) {
	w, err := repo.Conn().GetWorkerPools(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", postgres.ErrQueryFailed, err) //nolint:errorlint // prevent err in api
	}

	return workersToDomain(w), nil
}

func workersToDomain(w []models.ArrowerGueJobsWorkerPool) []jobs.WorkerPool {
	workers := make([]jobs.WorkerPool, len(w))

	for i, w := range w { //nolint:varnamelen // fp as loop index vars are allowed by config
		jt := []jobs.JobType{}

		for _, t := range w.JobTypes {
			jt = append(jt, jobs.JobType(t))
		}

		workers[i] = jobs.WorkerPool{
			ID:       w.ID,
			Queue:    queueNameToDomain(w.Queue),
			Version:  w.GitHash,
			JobTypes: jt,
			Workers:  int(w.Workers),
			LastSeen: w.UpdatedAt.Time,
		}
	}

	return workers
}

func (repo *PostgresJobsRepository) FinishedJobs(ctx context.Context, filter jobs.Filter) ([]jobs.Job, error) {
	queue := queueNameFromDomain(filter.Queue)

	if filter.Queue != "" && filter.JobType != "" {
		jobs, err := repo.Conn().GetFinishedJobsByQueueAndType(ctx, models.GetFinishedJobsByQueueAndTypeParams{
			Queue:   queue,
			JobType: string(filter.JobType),
		})
		if err != nil {
			return nil, fmt.Errorf("%w: could not get finished jobs by queue and job type: %v", postgres.ErrQueryFailed, err) //nolint:errorlint,lll // prevent err in api
		}

		return historyJobsToDomain(jobs), nil
	}

	if filter.Queue != "" {
		jobs, err := repo.Conn().GetFinishedJobsByQueue(ctx, queue)
		if err != nil {
			return nil, fmt.Errorf("%w: could not get finished jobs by queue: %v", postgres.ErrQueryFailed, err) //nolint:errorlint,lll // prevent err in api
		}

		return historyJobsToDomain(jobs), nil
	}

	jobs, err := repo.Conn().GetFinishedJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: could not get finished jobs: %v", postgres.ErrQueryFailed, err) //nolint:errorlint,lll // prevent err in api
	}

	return historyJobsToDomain(jobs), nil
}

func (repo *PostgresJobsRepository) FinishedJobsTotal(ctx context.Context, filter jobs.Filter) (int64, error) {
	queue := queueNameFromDomain(filter.Queue)

	if filter.Queue != "" && filter.JobType != "" {
		total, err := repo.Conn().TotalFinishedJobsByQueueAndType(ctx, models.TotalFinishedJobsByQueueAndTypeParams{
			Queue:   queue,
			JobType: string(filter.JobType),
		})
		if err != nil {
			return 0, fmt.Errorf("%w: %v", postgres.ErrQueryFailed, err) //nolint:errorlint // prevent err in api
		}

		return total, nil
	}

	if filter.Queue != "" {
		total, err := repo.Conn().TotalFinishedJobsByQueue(ctx, queue)
		if err != nil {
			return 0, fmt.Errorf("%w: %v", postgres.ErrQueryFailed, err) //nolint:errorlint // prevent err in api
		}

		return total, nil
	}

	total, err := repo.Conn().TotalFinishedJobs(ctx)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", postgres.ErrQueryFailed, err) //nolint:errorlint // prevent err in api
	}

	return total, nil
}

func historyJobsToDomain(j []models.ArrowerGueJobsHistory) []jobs.Job {
	jobs := make([]jobs.Job, len(j))

	for i := 0; i < len(j); i++ {
		jobs[i] = historyJobToDomain(j[i])
	}

	return jobs
}

func historyJobToDomain(job models.ArrowerGueJobsHistory) jobs.Job {
	return jobs.Job{
		ID:         job.JobID,
		Priority:   job.Priority,
		RunAt:      job.RunAt.Time,
		Type:       jobs.JobType(job.JobType),
		Payload:    string(job.Args),
		ErrorCount: job.RunCount,
		LastError:  job.RunError,
		Queue:      queueNameToDomain(job.Queue),
		CreatedAt:  job.CreatedAt.Time,
		UpdatedAt:  job.UpdatedAt.Time,
	}
}

func queueNameFromDomain(name jobs.QueueName) string {
	if name == jobs.DefaultQueueName {
		name = ""
	}

	return string(name)
}

func queueNameToDomain(name string) jobs.QueueName {
	if name == "" {
		name = string(jobs.DefaultQueueName)
	}

	return jobs.QueueName(name)
}
