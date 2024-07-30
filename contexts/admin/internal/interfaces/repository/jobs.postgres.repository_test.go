//go:build integration

package repository_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	mnoop "go.opentelemetry.io/otel/metric/noop"
	tnoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/contexts/admin/internal/domain/jobs"
	"github.com/go-arrower/arrower/contexts/admin/internal/interfaces/repository"
	"github.com/go-arrower/arrower/contexts/admin/internal/interfaces/repository/testdata"
	ajobs "github.com/go-arrower/arrower/jobs"
	"github.com/go-arrower/arrower/tests"
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

func TestPostgresJobsRepository_FindAllQueueNames(t *testing.T) {
	t.Parallel()

	t.Run("find all gue queues", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()

		// Given a job queue
		jq0, err := ajobs.NewPostgresJobs(alog.NewNoop(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(),
			pg, ajobs.WithPollInterval(time.Nanosecond),
		)
		assert.NoError(t, err)
		err = jq0.RegisterJobFunc(func(_ context.Context, _ testdata.SimpleJob) error { return nil })
		assert.NoError(t, err)
		err = jq0.Enqueue(ctx, testdata.SimpleJob{})
		assert.NoError(t, err)

		// And given a different job queue run in the future, meaning: this queue does not have a history yet
		jq1, err := ajobs.NewPostgresJobs(alog.NewNoop(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(),
			pg, ajobs.WithPollInterval(time.Nanosecond), ajobs.WithQueue("some_queue"),
		)
		assert.NoError(t, err)
		err = jq1.RegisterJobFunc(func(_ context.Context, _ testdata.SimpleJob) error { return nil })
		assert.NoError(t, err)
		err = jq1.Enqueue(ctx, testdata.SimpleJob{}, ajobs.WithRunAt(time.Now().Add(1*time.Hour)))
		assert.NoError(t, err)

		time.Sleep(100 * time.Millisecond) // wait for job to finish

		repo := repository.NewPostgresJobsRepository(pg)

		q, err := repo.FindAllQueueNames(ctx)
		assert.NoError(t, err)
		assert.Len(t, q, 2, "should have two different queues")
		assert.Equal(t, jobs.DefaultQueueName, q[0])
		assert.Equal(t, jobs.QueueName("some_queue"), q[1])
	})

	t.Run("deduplicate queues", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase("testdata/fixtures/findAllQueueNames.yaml")
		repo := repository.NewPostgresJobsRepository(pg)

		q, err := repo.FindAllQueueNames(ctx)
		assert.NoError(t, err)
		assert.EqualValues(t, jobs.QueueNames{"Default", "Q0", "Q1"}, q)
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

		jq, _ := ajobs.NewPostgresJobs(alog.NewNoop(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg)
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

func TestPostgresJobsRepository_DeleteByID(t *testing.T) {
	t.Parallel()

	t.Run("job", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo := repository.NewPostgresJobsRepository(pg)
		jq, err := ajobs.NewPostgresJobs(alog.NewNoop(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg)
		assert.NoError(t, err)

		err = jq.Enqueue(ctx, testdata.SimpleJob{})
		assert.NoError(t, err)

		pending, err := repo.PendingJobs(ctx, "")
		assert.NoError(t, err)
		assert.Len(t, pending, 1, "a job should be enqueued")

		err = repo.DeleteByID(ctx, pending[0].ID)
		assert.NoError(t, err)

		pending, err = repo.PendingJobs(ctx, "")
		assert.NoError(t, err)
		assert.Empty(t, pending, "should be empty")
	})

	t.Run("non-existing", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo := repository.NewPostgresJobsRepository(pg)

		err := repo.DeleteByID(ctx, "non-existing-id")
		assert.NoError(t, err)

		pending, err := repo.PendingJobs(ctx, "")
		assert.NoError(t, err)
		assert.Empty(t, pending)
	})

	t.Run("already running job", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo := repository.NewPostgresJobsRepository(pg)
		jq, err := ajobs.NewPostgresJobs(alog.NewNoop(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			ajobs.WithPollInterval(time.Nanosecond),
		)
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(_ context.Context, _ testdata.SimpleJob) error {
			time.Sleep(1 * time.Minute) // simulate a long-running job
			assert.Fail(t, "this should never be called, job continues to run but tests aborts")

			return nil
		})
		assert.NoError(t, err)
		err = jq.Enqueue(ctx, testdata.SimpleJob{})
		assert.NoError(t, err)

		pending, err := repo.PendingJobs(ctx, "")
		assert.NoError(t, err)
		assert.Len(t, pending, 1)

		time.Sleep(100 * time.Millisecond) // start the worker

		err = repo.DeleteByID(ctx, pending[0].ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, jobs.ErrJobLockedAlready, "started jobs can not be deleted")

		pending, err = repo.PendingJobs(ctx, "")
		assert.NoError(t, err)
		assert.Len(t, pending, 1+1, "delete should fail, as the job is currently processed and thus locked by the db") // +1 is gueron job
	})
}

func TestPostgresJobsRepository_RunJobAt(t *testing.T) {
	t.Parallel()

	t.Run("reschedule job", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo := repository.NewPostgresJobsRepository(pg)
		jq, _ := ajobs.NewPostgresJobs(alog.NewNoop(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
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
		jq, _ := ajobs.NewPostgresJobs(alog.NewNoop(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			ajobs.WithPollInterval(time.Nanosecond),
		)

		_ = jq.RegisterJobFunc(func(_ context.Context, _ testdata.SimpleJob) error {
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
