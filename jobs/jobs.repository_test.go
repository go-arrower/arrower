//go:build integration

package jobs_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/jobs"
	"github.com/go-arrower/arrower/jobs/models"
	"github.com/go-arrower/arrower/tests"
)

func TestPostgresGueRepository_Queues(t *testing.T) {
	t.Parallel()

	t.Run("get all gue queues", func(t *testing.T) {
		t.Parallel()

		pg := tests.PrepareTestDatabase(pgHandler)

		// Given a job queue
		jq0, _ := jobs.NewGueJobs(pg.PGx, jobs.WithPollInterval(time.Nanosecond))
		_ = jq0.RegisterWorker(func(ctx context.Context, job simpleJob) error { return nil })
		_ = jq0.Enqueue(ctx, simpleJob{})
		_ = jq0.StartWorkers()

		// And Given a different job queue run in the future
		jq1, _ := jobs.NewGueJobs(pg.PGx, jobs.WithPollInterval(time.Nanosecond), jobs.WithQueue("some_queue"))
		_ = jq1.RegisterWorker(func(ctx context.Context, job simpleJob) error { return nil })
		_ = jq1.Enqueue(ctx, simpleJob{}, jobs.WithRunAt(time.Now().Add(1*time.Hour)))
		_ = jq1.StartWorkers()

		time.Sleep(100 * time.Millisecond) // wait for job to finish

		repo := jobs.NewPostgresJobsRepository(models.New(pg.PGx))

		q, err := repo.Queues(ctx)
		assert.NoError(t, err)
		assert.Len(t, q, 2)
	})
}

func TestPostgresJobsRepository_PendingJobs(t *testing.T) {
	t.Parallel()

	t.Run("", func(t *testing.T) {
		t.Parallel()

		pg := tests.PrepareTestDatabase(pgHandler)

		repo := jobs.NewPostgresJobsRepository(models.New(pg.PGx))

		pendingJobs, err := repo.PendingJobs(ctx, "")
		assert.NoError(t, err)
		assert.Equal(t, 0, len(pendingJobs))

		jq0, _ := jobs.NewGueJobs(pg.PGx, jobs.WithPollInterval(time.Nanosecond))
		_ = jq0.RegisterWorker(func(ctx context.Context, job simpleJob) error { return nil })
		_ = jq0.Enqueue(ctx, simpleJob{})

		pendingJobs, err = repo.PendingJobs(ctx, "")
		assert.NoError(t, err)
		assert.Equal(t, 1, len(pendingJobs))
	})
}

func TestPostgresJobsRepository_QueueKPIs(t *testing.T) {
	t.Parallel()

	t.Run("empty gue tables", func(t *testing.T) {
		t.Parallel()

		pg := tests.PrepareTestDatabase(pgHandler)

		repo := jobs.NewPostgresJobsRepository(models.New(pg.PGx))

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

		pg := tests.PrepareTestDatabase(pgHandler, "testdata/fixtures/queue_kpis.yaml")

		repo := jobs.NewPostgresJobsRepository(models.New(pg.PGx))

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

		pg := tests.PrepareTestDatabase(pgHandler)

		repo := jobs.NewPostgresJobsRepository(models.New(pg.PGx))

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
