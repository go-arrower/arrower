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
