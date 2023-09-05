//go:build integration

package jobs_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/jobs"
	"github.com/go-arrower/arrower/jobs/models"
)

func TestPostgresGueRepository_Queues(t *testing.T) {
	t.Parallel()

	t.Run("get all gue queues", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()

		// Given a job queue
		jq0, _ := jobs.NewPostgresJobs(logger, noop.NewMeterProvider(), trace.NewNoopTracerProvider(),
			pg, jobs.WithPollInterval(time.Nanosecond),
		)
		_ = jq0.RegisterJobFunc(func(ctx context.Context, job simpleJob) error { return nil })
		_ = jq0.Enqueue(ctx, simpleJob{})

		// And Given a different job queue run in the future
		jq1, _ := jobs.NewPostgresJobs(alog.NewNoopLogger(), noop.NewMeterProvider(), trace.NewNoopTracerProvider(),
			pg, jobs.WithPollInterval(time.Nanosecond), jobs.WithQueue("some_queue"),
		)
		_ = jq1.RegisterJobFunc(func(ctx context.Context, job simpleJob) error { return nil })
		_ = jq1.Enqueue(ctx, simpleJob{}, jobs.WithRunAt(time.Now().Add(1*time.Hour)))

		time.Sleep(100 * time.Millisecond) // wait for job to finish

		repo := jobs.NewPostgresJobsRepository(models.New(pg))

		q, err := repo.Queues(ctx)
		assert.NoError(t, err)
		assert.Len(t, q, 2)
	})
}

func TestPostgresJobsRepository_PendingJobs(t *testing.T) {
	t.Parallel()

	t.Run("", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo := jobs.NewPostgresJobsRepository(models.New(pg))

		pendingJobs, err := repo.PendingJobs(ctx, "")
		assert.NoError(t, err)
		assert.Len(t, pendingJobs, 0, "queue needs to be empty, as no jobs got enqueued yet")

		jq, _ := jobs.NewPostgresJobs(logger, noop.NewMeterProvider(), trace.NewNoopTracerProvider(), pg)
		_ = jq.Enqueue(ctx, simpleJob{})

		pendingJobs, err = repo.PendingJobs(ctx, "")
		assert.NoError(t, err)
		assert.Len(t, pendingJobs, 1, "one Job is enqueued")
	})
}

func TestPostgresJobsRepository_QueueKPIs(t *testing.T) {
	t.Parallel()

	t.Run("empty gue tables", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()

		repo := jobs.NewPostgresJobsRepository(models.New(pg))

		stats, err := repo.QueueKPIs(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, stats.PendingJobs, 0)
		assert.Equal(t, stats.FailedJobs, 0)
		assert.Equal(t, stats.ProcessedJobs, 0)
		assert.Equal(t, stats.AverageTimePerJob, time.Duration(0))
		assert.Equal(t, stats.AvailableWorkers, 0)
		assert.Empty(t, stats.PendingJobsPerType)
	})

	t.Run("queue KPIs from gue tables", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase("testdata/fixtures/queue_kpis.yaml")

		repo := jobs.NewPostgresJobsRepository(models.New(pg))

		stats, err := repo.QueueKPIs(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, stats.PendingJobs, 3)
		assert.Equal(t, stats.FailedJobs, 1)
		assert.Equal(t, stats.ProcessedJobs, 2)
		assert.Equal(t, stats.AverageTimePerJob, time.Duration(1500)*time.Millisecond)
		assert.Equal(t, stats.AvailableWorkers, 1337)
		assert.Equal(t, stats.PendingJobsPerType, map[string]int{
			"type_0": 2,
			"type_1": 1,
		})
	})
}

func TestPostgresJobsRepository_RegisterWorkerPool(t *testing.T) {
	t.Parallel()

	t.Run("register worker pool", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()

		repo := jobs.NewPostgresJobsRepository(models.New(pg))

		workerPool := jobs.WorkerPool{
			ID:       "1337",
			Queue:    "",
			Workers:  1337,
			LastSeen: time.Now(),
		}

		// register new worker pool
		err := repo.RegisterWorkerPool(context.Background(), workerPool)
		assert.NoError(t, err)

		// ensure worker pool got registered
		wp, err := repo.WorkerPools(context.Background())
		assert.NoError(t, err)
		assert.Len(t, wp, 1)
		wpLastSeen := wp[0].LastSeen

		// ensure same pool updates it's registration
		workerPool.Workers = 1
		err = repo.RegisterWorkerPool(context.Background(), workerPool)
		assert.NoError(t, err)
		wp, err = repo.WorkerPools(context.Background())
		assert.NoError(t, err)
		assert.Len(t, wp, 1)
		assert.True(t, wp[0].LastSeen.After(wpLastSeen))
		assert.Equal(t, 1, wp[0].Workers)
	})
}

func TestPostgresJobsRepository_Delete(t *testing.T) {
	t.Parallel()

	t.Run("delete job", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()

		repo := jobs.NewPostgresJobsRepository(models.New(pg))
		jq, _ := jobs.NewPostgresJobs(alog.NewNoopLogger(), noop.NewMeterProvider(), trace.NewNoopTracerProvider(), pg)

		_ = jq.Enqueue(ctx, simpleJob{})

		pending, _ := repo.PendingJobs(ctx, "")
		assert.Len(t, pending, 1)

		err := repo.Delete(ctx, pending[0].ID)
		assert.NoError(t, err)

		pending, _ = repo.PendingJobs(ctx, "")
		assert.Len(t, pending, 0)
	})

	t.Run("delete already running job", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()

		repo := jobs.NewPostgresJobsRepository(models.New(pg))
		jq, _ := jobs.NewPostgresJobs(alog.NewNoopLogger(), noop.NewMeterProvider(), trace.NewNoopTracerProvider(), pg,
			jobs.WithPollInterval(time.Nanosecond),
		)

		_ = jq.RegisterJobFunc(func(ctx context.Context, job simpleJob) error {
			time.Sleep(1 * time.Minute) // simulate a long-running job
			assert.Fail(t, "this should never be called, job continues to run but tests aborts")

			return nil
		})
		_ = jq.Enqueue(ctx, simpleJob{})

		pending, _ := repo.PendingJobs(ctx, "")
		assert.Len(t, pending, 1)

		time.Sleep(100 * time.Millisecond) // start the worker

		err := repo.Delete(ctx, pending[0].ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, jobs.ErrJobLockedAlready)

		pending, _ = repo.PendingJobs(ctx, "")
		assert.Len(t, pending, 1, "delete should fail, as the job is currently processed and thus locked by the db")
	})
}

func TestPostgresJobsRepository_RunJobAt(t *testing.T) {
	t.Parallel()

	t.Run("reschedule job", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo := jobs.NewPostgresJobsRepository(models.New(pg))
		jq, _ := jobs.NewPostgresJobs(alog.NewNoopLogger(), noop.NewMeterProvider(), trace.NewNoopTracerProvider(), pg,
			jobs.WithPollInterval(time.Nanosecond),
		)

		newJobTime := time.Now().Add(time.Minute)

		_ = jq.Enqueue(ctx, simpleJob{})

		pending, _ := repo.PendingJobs(ctx, "")
		assert.Len(t, pending, 1)
		assert.NotEqual(t, newJobTime.Format(time.RFC3339), pending[0].RunAt.Format(time.RFC3339))

		err := repo.RunJobAt(ctx, pending[0].ID, newJobTime)
		assert.NoError(t, err)

		pending, _ = repo.PendingJobs(ctx, "")
		assert.Equal(t, newJobTime.Format(time.RFC3339), pending[0].RunAt.Format(time.RFC3339))
	})

	t.Run("reschedule already started job", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo := jobs.NewPostgresJobsRepository(models.New(pg))
		jq, _ := jobs.NewPostgresJobs(alog.NewNoopLogger(), noop.NewMeterProvider(), trace.NewNoopTracerProvider(), pg,
			jobs.WithPollInterval(time.Nanosecond),
		)

		_ = jq.RegisterJobFunc(func(ctx context.Context, job simpleJob) error {
			time.Sleep(1 * time.Minute) // simulate a long-running job
			assert.Fail(t, "this should never be called, job continues to run but tests aborts")

			return nil
		})

		_ = jq.Enqueue(ctx, simpleJob{})
		pending, _ := repo.PendingJobs(ctx, "")

		time.Sleep(100 * time.Millisecond) // start the worker

		newJobTime := time.Now().Add(time.Minute)

		err := repo.RunJobAt(ctx, pending[0].ID, newJobTime)
		assert.Error(t, err)
		assert.ErrorIs(t, err, jobs.ErrJobLockedAlready)
	})
}
