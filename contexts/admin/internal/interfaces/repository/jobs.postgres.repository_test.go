//go:build integration

package repository_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-arrower/arrower/alog"
	ajobs "github.com/go-arrower/arrower/jobs"
	"github.com/go-arrower/arrower/tests"
	"github.com/stretchr/testify/assert"
	mnoop "go.opentelemetry.io/otel/metric/noop"
	tnoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/go-arrower/arrower/contexts/admin/internal/domain/jobs"
	"github.com/go-arrower/arrower/contexts/admin/internal/interfaces/repository"
	"github.com/go-arrower/arrower/contexts/admin/internal/interfaces/repository/testdata"
)

var (
	ctx       = context.Background()
	pgHandler *tests.PostgresDocker
)

func TestMain(m *testing.M) {
	pgHandler = tests.GetPostgresDockerForIntegrationTestingInstance()

	//
	// Run tests
	code := m.Run()

	pgHandler.Cleanup()
	os.Exit(code)
}
func TestPostgresGueRepository_Queues(t *testing.T) {
	t.Parallel()

	t.Run("get all gue queues", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()

		// Given a job queue
		jq0, _ := ajobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(),
			pg, ajobs.WithPollInterval(time.Nanosecond),
		)
		_ = jq0.RegisterJobFunc(func(ctx context.Context, job testdata.SimpleJob) error { return nil })
		_ = jq0.Enqueue(ctx, testdata.SimpleJob{})

		// And given a different job queue run in the future, meaning: this queue does not have a history yet
		jq1, _ := ajobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(),
			pg, ajobs.WithPollInterval(time.Nanosecond), ajobs.WithQueue("some_queue"),
		)
		_ = jq1.RegisterJobFunc(func(ctx context.Context, job testdata.SimpleJob) error { return nil })
		_ = jq1.Enqueue(ctx, testdata.SimpleJob{}, ajobs.WithRunAt(time.Now().Add(1*time.Hour)))

		time.Sleep(100 * time.Millisecond) // wait for job to finish

		repo := repository.NewPostgresJobsRepository(pg)

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
		repo := repository.NewPostgresJobsRepository(pg)

		pendingJobs, err := repo.PendingJobs(ctx, jobs.DefaultQueueName)
		assert.NoError(t, err)
		assert.Empty(t, pendingJobs, "queue needs to be empty, as no jobs got enqueued yet")

		jq, _ := ajobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg)
		_ = jq.Enqueue(ctx, testdata.SimpleJob{})

		pendingJobs, err = repo.PendingJobs(ctx, jobs.DefaultQueueName)
		assert.NoError(t, err)
		assert.Len(t, pendingJobs, 1, "one job is enqueued")
	})
}

func TestPostgresJobsRepository_QueueKPIs(t *testing.T) {
	t.Parallel()

	t.Run("empty gue tables", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()

		repo := repository.NewPostgresJobsRepository(pg)

		stats, err := repo.QueueKPIs(context.Background(), jobs.DefaultQueueName)
		assert.NoError(t, err)
		assert.Equal(t, 0, stats.PendingJobs)
		assert.Equal(t, 0, stats.FailedJobs)
		assert.Equal(t, 0, stats.ProcessedJobs)
		assert.Equal(t, time.Duration(0), stats.AverageTimePerJob)
		assert.Equal(t, 0, stats.AvailableWorkers)
		assert.Empty(t, stats.PendingJobsPerType)
	})

	t.Run("queue KPIs from gue tables", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase("testdata/fixtures/queue_kpis.yaml")

		repo := repository.NewPostgresJobsRepository(pg)

		stats, err := repo.QueueKPIs(context.Background(), jobs.DefaultQueueName)
		assert.NoError(t, err)
		assert.Equal(t, 3, stats.PendingJobs)
		assert.Equal(t, 1, stats.FailedJobs)
		assert.Equal(t, 2, stats.ProcessedJobs)
		assert.Equal(t, time.Duration(1500)*time.Millisecond, stats.AverageTimePerJob)
		assert.Equal(t, 1337, stats.AvailableWorkers)
		assert.Equal(t, map[string]int{
			"type_0": 2,
			"type_1": 1,
		}, stats.PendingJobsPerType)
	})
}

func TestPostgresJobsRepository_Delete(t *testing.T) {
	t.Parallel()

	t.Run("delete job", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()

		repo := repository.NewPostgresJobsRepository(pg)
		jq, _ := ajobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg)

		_ = jq.Enqueue(ctx, testdata.SimpleJob{})

		pending, _ := repo.PendingJobs(ctx, "")
		assert.Len(t, pending, 1)

		err := repo.Delete(ctx, pending[0].ID)
		assert.NoError(t, err)

		pending, _ = repo.PendingJobs(ctx, "")
		assert.Empty(t, pending)
	})

	t.Run("delete already running job", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()

		repo := repository.NewPostgresJobsRepository(pg)
		jq, _ := ajobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			ajobs.WithPollInterval(time.Nanosecond),
		)

		_ = jq.RegisterJobFunc(func(ctx context.Context, job testdata.SimpleJob) error {
			time.Sleep(1 * time.Minute) // simulate a long-running job
			assert.Fail(t, "this should never be called, job continues to run but tests aborts")

			return nil
		})
		_ = jq.Enqueue(ctx, testdata.SimpleJob{})

		pending, _ := repo.PendingJobs(ctx, "")
		assert.Len(t, pending, 1)

		time.Sleep(100 * time.Millisecond) // start the worker

		err := repo.Delete(ctx, pending[0].ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, jobs.ErrJobLockedAlready)

		pending, _ = repo.PendingJobs(ctx, "")
		assert.Len(t, pending, 1+1, "delete should fail, as the job is currently processed and thus locked by the db") // +1 is gueron job
	})
}

func TestPostgresJobsRepository_RunJobAt(t *testing.T) {
	t.Parallel()

	t.Run("reschedule job", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo := repository.NewPostgresJobsRepository(pg)
		jq, _ := ajobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			ajobs.WithPollInterval(time.Nanosecond),
		)

		newJobTime := time.Now().Add(time.Minute)

		_ = jq.Enqueue(ctx, testdata.SimpleJob{})

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
		repo := repository.NewPostgresJobsRepository(pg)
		jq, _ := ajobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			ajobs.WithPollInterval(time.Nanosecond),
		)

		_ = jq.RegisterJobFunc(func(ctx context.Context, job testdata.SimpleJob) error {
			time.Sleep(1 * time.Minute) // simulate a long-running job
			assert.Fail(t, "this should never be called, job continues to run but tests aborts")

			return nil
		})

		_ = jq.Enqueue(ctx, testdata.SimpleJob{})
		pending, _ := repo.PendingJobs(ctx, "")

		time.Sleep(100 * time.Millisecond) // start the worker

		newJobTime := time.Now().Add(time.Minute)

		err := repo.RunJobAt(ctx, pending[0].ID, newJobTime)
		assert.Error(t, err)
		assert.ErrorIs(t, err, jobs.ErrJobLockedAlready)
	})
}
