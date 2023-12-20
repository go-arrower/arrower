//go:build integration

package jobs_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	mnoop "go.opentelemetry.io/otel/metric/noop"
	tnoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/jobs"
	"github.com/go-arrower/arrower/jobs/models"
	"github.com/go-arrower/arrower/postgres"
	"github.com/go-arrower/arrower/tests"
)

var pgHandler *tests.PostgresDocker

func TestMain(m *testing.M) {
	pgHandler = tests.GetPostgresDockerForIntegrationTestingInstance()

	//
	// Run tests
	code := m.Run()

	pgHandler.Cleanup()
	os.Exit(code)
}

func TestNewGueJobs(t *testing.T) {
	t.Parallel()

	t.Run("initialise a new PostgresJobsHandler", func(t *testing.T) {
		t.Parallel()

		jq, err := jobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pgHandler.PGx())
		assert.NoError(t, err)
		assert.NotEmpty(t, jq)
	})

	t.Run("initialise a new PostgresJobsHandler with queue", func(t *testing.T) {
		t.Parallel()

		jq, err := jobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pgHandler.PGx(),
			jobs.WithQueue("name"),
		)
		assert.NoError(t, err)
		assert.NotEmpty(t, jq)
	})

	t.Run("initialise a new PostgresJobsHandler with poll interval", func(t *testing.T) {
		t.Parallel()

		jq, err := jobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pgHandler.PGx(),
			jobs.WithPollInterval(time.Nanosecond),
		)
		assert.NoError(t, err)
		assert.NotEmpty(t, jq)
	})

	t.Run("initialise a new PostgresJobsHandler with pool size", func(t *testing.T) {
		t.Parallel()

		jq, err := jobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pgHandler.PGx(),
			jobs.WithPoolSize(1),
		)
		assert.NoError(t, err)
		assert.NotEmpty(t, jq)
	})

	t.Run("initialise a new PostgresJobsHandler with pool name", func(t *testing.T) {
		t.Parallel()

		buf := &syncBuffer{}
		logger := alog.NewTest(buf)
		alog.Unwrap(logger).SetLevel(alog.LevelDebug)

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(logger, mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPoolName(argName), jobs.WithPollInterval(time.Nanosecond),
		)
		assert.NoError(t, err)
		assert.NotEmpty(t, jq)

		_ = jq.RegisterJobFunc(func(context.Context, simpleJob) error { return nil }) // register JobFunc to start workers
		_ = jq.Enqueue(ctx, simpleJob{})
		time.Sleep(500 * time.Millisecond) // wait for worker to start and process the job

		queries := models.New(pg)
		wp, err := queries.GetWorkerPools(ctx)
		assert.NoError(t, err)

		// expect one worker with right name to be registered in the db
		assert.Len(t, wp, 1)
		assert.Equal(t, argName, wp[0].ID)
		// expect the gue-logs to contain the right name as client-id
		assert.Contains(t, buf.String(), "jobs.gue.client-id="+argName)
	})
}

func TestGueHandler_RegisterJobFunc(t *testing.T) {
	t.Parallel()

	t.Run("ensure JobFunc is of type func and signature has two arguments with first ctx and returns error", func(t *testing.T) {
		t.Parallel()

		jq, err := jobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pgHandler.PGx())
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(nil)
		assert.ErrorIs(t, err, jobs.ErrInvalidJobFunc)

		err = jq.RegisterJobFunc("")
		assert.ErrorIs(t, err, jobs.ErrInvalidJobFunc)

		err = jq.RegisterJobFunc(func() {})
		assert.ErrorIs(t, err, jobs.ErrInvalidJobFunc)

		err = jq.RegisterJobFunc(func(id int, data []byte) error { return nil })
		assert.ErrorIs(t, err, jobs.ErrInvalidJobFunc)

		// primitive types do not implement a JobType interface and have no struct name.
		err = jq.RegisterJobFunc(func(ctx context.Context, data []byte) error { return nil })
		assert.ErrorIs(t, err, jobs.ErrInvalidJobFunc)

		err = jq.RegisterJobFunc(func(ctx context.Context, job simpleJob) {})
		assert.ErrorIs(t, err, jobs.ErrInvalidJobFunc)

		err = jq.RegisterJobFunc(func(ctx context.Context, job simpleJob) int { return 0 })
		assert.ErrorIs(t, err, jobs.ErrInvalidJobFunc)

		err = jq.RegisterJobFunc(func(ctx context.Context, job simpleJob) error { return nil })
		assert.NoError(t, err)
	})

	t.Run("ensure only one worker can register for a JobType name", func(t *testing.T) {
		t.Parallel()

		jq, err := jobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pgHandler.PGx())
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(ctx context.Context, job simpleJob) error { return nil })
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(ctx context.Context, job simpleJob) error { return nil })
		assert.ErrorIs(t, err, jobs.ErrInvalidJobFunc)

		err = jq.RegisterJobFunc(func(ctx context.Context, job jobWithSameNameAsSimpleJob) error { return nil })
		assert.ErrorIs(t, err, jobs.ErrInvalidJobFunc)
	})

	t.Run("ensure register takes a jobType from the interface implementation", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(time.Nanosecond),
		)
		assert.NoError(t, err)

		wg.Add(1)
		err = jq.RegisterJobFunc(func(ctx context.Context, j jobWithJobType) error {
			assert.Equal(t, jobWithJobType{Name: argName}, j)
			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		err = jq.Enqueue(ctx, jobWithJobType{Name: argName})
		assert.NoError(t, err)

		wg.Wait()
		_ = jq.Shutdown(ctx)
	})

	t.Run("ensure retries are done on an error", func(t *testing.T) {
		t.Parallel()

		var (
			wg         sync.WaitGroup
			mu         sync.Mutex
			retryCount int
		)

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(time.Nanosecond),
		)
		assert.NoError(t, err)

		wg.Add(1)
		err = jq.RegisterJobFunc(func(ctx context.Context, job jobWithJobType) error {
			mu.Lock()
			defer mu.Unlock()

			if retryCount < 1 {
				retryCount++

				return fmt.Errorf("some failure inside a job worker: %d", retryCount) //nolint:goerr113
			}

			assert.Equal(t, jobWithJobType{Name: argName}, job)
			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		err = jq.Enqueue(ctx, jobWithJobType{Name: argName})
		assert.NoError(t, err)

		wg.Wait()
		_ = jq.Shutdown(ctx)
	})
}

func TestGueHandler_Enqueue(t *testing.T) {
	t.Parallel()

	t.Run("invalid job", func(t *testing.T) {
		t.Parallel()

		jq, err := jobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pgHandler.PGx(),
			jobs.WithPollInterval(time.Nanosecond),
		)
		assert.NoError(t, err)

		err = jq.Enqueue(ctx, "")
		assert.ErrorIs(t, err, jobs.ErrInvalidJobType)

		err = jq.Enqueue(ctx, []byte(""))
		assert.ErrorIs(t, err, jobs.ErrInvalidJobType)

		err = jq.Enqueue(ctx, 0)
		assert.ErrorIs(t, err, jobs.ErrInvalidJobType)

		err = jq.Enqueue(ctx, nil)
		assert.ErrorIs(t, err, jobs.ErrInvalidJobType)

		err = jq.Enqueue(ctx, func() {})
		assert.ErrorIs(t, err, jobs.ErrInvalidJobType)
	})

	t.Run("single job", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(time.Nanosecond),
		)
		assert.NoError(t, err)

		wg.Add(1)
		err = jq.RegisterJobFunc(func(ctx context.Context, j jobWithArgs) error {
			assert.Equal(t, payloadJob, j)
			assert.Equal(t, argName, j.Name)
			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		err = jq.Enqueue(ctx, payloadJob)
		assert.NoError(t, err)

		wg.Wait()
		_ = jq.Shutdown(ctx)
	})

	t.Run("slice of same job type, struct name and custom", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(time.Nanosecond),
		)
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(ctx context.Context, j jobWithArgs) error {
			assert.Equal(t, argName, j.Name)
			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(ctx context.Context, j jobWithJobType) error {
			assert.Equal(t, argName, j.Name)
			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		wg.Add(2)
		err = jq.Enqueue(ctx, []jobWithArgs{{Name: argName}, {Name: argName}})
		assert.NoError(t, err)

		wg.Add(2)
		err = jq.Enqueue(ctx, []jobWithJobType{{Name: argName}, {Name: argName}})
		assert.NoError(t, err)

		wg.Wait()                          // all workers are done, and now:
		time.Sleep(100 * time.Millisecond) // wait until gue finishes with the underlying transaction

		ensureJobTableRows(t, pg, 0)        // all jobs are processed
		ensureJobHistoryTableRows(t, pg, 4) // history has all jobs

		_ = jq.Shutdown(ctx)
	})

	t.Run("slice of different job types", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(time.Nanosecond),
		)
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(ctx context.Context, j jobWithArgs) error {
			assert.Equal(t, argName, j.Name)
			wg.Done()

			return nil
		})
		assert.NoError(t, err)
		err = jq.RegisterJobFunc(func(ctx context.Context, j jobWithJobType) error {
			assert.Equal(t, argName, j.Name)
			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		wg.Add(2)
		err = jq.Enqueue(ctx, []any{jobWithArgs{Name: argName}, jobWithJobType{Name: argName}})
		assert.NoError(t, err)

		wg.Wait()                          // all workers are done, and now:
		time.Sleep(100 * time.Millisecond) // wait until gue finishes with the underlying transaction

		ensureJobTableRows(t, pg, 0)        // all jobs are processed
		ensureJobHistoryTableRows(t, pg, 2) // history has all jobs

		_ = jq.Shutdown(ctx)
	})

	t.Run("ensure priority is set", func(t *testing.T) {
		t.Parallel()

		var (
			wg    sync.WaitGroup
			mu    sync.Mutex
			order int
		)

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(time.Nanosecond), jobs.WithPoolSize(1),
		)
		assert.NoError(t, err)

		// enforce the same runAt time, so the priority is considered by the queue as the second argument to order with.
		runAt := jobs.WithRunAt(time.Now().UTC())

		// First queued job has default priority
		err = jq.Enqueue(ctx, jobWithArgs{Name: "1"}, runAt)
		assert.NoError(t, err)

		// Second queued job has higher priority and should be processed first
		err = jq.Enqueue(ctx, jobWithArgs{Name: "0"}, runAt, jobs.WithPriority(-1))
		assert.NoError(t, err)

		wg.Add(2)
		err = jq.RegisterJobFunc(func(ctx context.Context, job jobWithArgs) error {
			mu.Lock()
			defer mu.Unlock()

			if job.Name == "0" { // expect job with Name 0 to be run first (^= 0) in order
				assert.Equal(t, 0, order)
			}
			if job.Name == "1" { // expect job with Name 1 to be run afterwards ^= higher order
				assert.Equal(t, 1, order)
			}

			order++
			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		wg.Wait()
		_ = jq.Shutdown(ctx)
	})
}

func TestGueHandler_StartWorkers(t *testing.T) {
	t.Parallel()

	t.Run("restart queue if RegisterJobFunc is called after start", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(time.Nanosecond),
		)
		assert.NoError(t, err)

		_ = jq.RegisterJobFunc(func(context.Context, simpleJob) error { return nil })
		time.Sleep(time.Millisecond) // wait until the workers have started

		err = jq.Enqueue(ctx, payloadJob)
		assert.NoError(t, err)

		wg.Add(1)
		err = jq.RegisterJobFunc(func(ctx context.Context, j jobWithArgs) error {
			assert.Equal(t, payloadJob, j)
			assert.Equal(t, argName, j.Name)
			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		wg.Wait()
		_ = jq.Shutdown(ctx)
	})

	t.Run("queue starts only after poll interval time, after the last registered job func", func(t *testing.T) {
		t.Parallel()

		// To understand if the job queue has started its underlying worker pool already, this test relies on the
		// log output from RegisterJobFunc in case it is restarting.
		buf := &syncBuffer{}
		logger := alog.NewTest(buf)
		alog.Unwrap(logger).SetLevel(alog.LevelInfo)

		pollInterval := 100 * time.Millisecond
		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(logger, mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(pollInterval),
		)
		assert.NoError(t, err)

		// register first worker, no restart of the queue required anyway
		err = jq.RegisterJobFunc(func(context.Context, simpleJob) error { return nil })
		assert.NoError(t, err)
		assert.Empty(t, buf.String())

		time.Sleep(time.Millisecond) // wait a bit, but not longer than the pollInterval

		// register second worker, expect no restart of the queue, as it is not started yet
		err = jq.RegisterJobFunc(func(context.Context, jobWithArgs) error { return nil })
		assert.NoError(t, err)
		assert.Empty(t, buf.String())
		assert.NotContains(t, buf.String(), `msg="restart workers"`)

		time.Sleep(10 * pollInterval) // wait longer than the pollInterval, so the queue is started

		// register third worker, expect restart of the queue
		err = jq.RegisterJobFunc(func(context.Context, jobWithJobType) error { return nil })
		assert.NoError(t, err)
		assert.NotEmpty(t, buf.String())
		assert.Contains(t, buf.String(), `msg="restart workers"`)
	})

	t.Run("ensure worker pool is registered & unregistered", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		queries := models.New(pg)
		jq, err := jobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPoolName("pool_name"), jobs.WithPoolSize(7), jobs.WithPollInterval(time.Nanosecond),
		)
		assert.NoError(t, err)

		// pool only starts after a call to RegisterJobFunc
		_ = jq.RegisterJobFunc(func(context.Context, simpleJob) error { return nil })
		// wait for the startWorkers to register itself as online
		time.Sleep(100 * time.Millisecond)

		wp, err := queries.GetWorkerPools(ctx)
		assert.NoError(t, err)
		assert.Len(t, wp, 1)
		assert.Equal(t, "pool_name", wp[0].ID)
		assert.Equal(t, int16(7), wp[0].Workers)

		err = jq.Shutdown(ctx)
		assert.NoError(t, err)

		wp, err = queries.GetWorkerPools(ctx)
		assert.NoError(t, err)
		assert.Len(t, wp, 1)
		assert.Equal(t, int16(0), wp[0].Workers)
	})
}

func TestGueHandler_History(t *testing.T) {
	t.Parallel()

	t.Run("ensure successful jobs are recorded into gue_jobs_history table", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(time.Nanosecond),
		)
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(ctx context.Context, j jobWithArgs) error { return nil })
		assert.NoError(t, err)

		// job history table is empty before first job is enqueued
		ensureJobHistoryTableRows(t, pg, 0)

		err = jq.Enqueue(ctx, jobWithArgs{})
		assert.NoError(t, err)

		// wait until the worker & all it's hooks are processed. The use of a sync.WaitGroup in the JobFunc does not work,
		// because wg.Done() can only be called from the worker func and not the hooks (where it would need to be placed).
		time.Sleep(time.Millisecond * 400)

		_ = jq.Shutdown(ctx)

		// job history table contains finished job
		ensureJobHistoryTableRows(t, pg, 1)

		// ensure the job is finished successful
		var hJob gueJobHistory
		err = pgxscan.Get(ctx, pg, &hJob, `SELECT * FROM public.gue_jobs_history`)
		assert.NoError(t, err)

		assert.True(t, hJob.Success)
		assert.NotEmpty(t, hJob.FinishedAt)
		assert.Equal(t, 0, hJob.RunCount)
		assert.Empty(t, hJob.RunError)
		assert.NotEqual(t, hJob.CreatedAt, *hJob.FinishedAt)
	})

	t.Run("ensure failed jobs are recorded in gue_jobs_history table", func(t *testing.T) {
		t.Parallel()

		var (
			count int
			wg    sync.WaitGroup
		)

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(time.Nanosecond),
		)
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(ctx context.Context, j jobWithArgs) error {
			if count < 2 { // fail twice
				count++

				return errors.New("job returns with error") //nolint:goerr113
			}

			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		// job history table is empty before the first job is enqueued
		ensureJobHistoryTableRows(t, pg, 0)

		wg.Add(1)
		err = jq.Enqueue(ctx, jobWithArgs{Name: "argName"})
		assert.NoError(t, err)

		wg.Wait() // wait for the worker

		// wait until the all worker hooks are processed. The use of a sync.WaitGroup does not work,
		// because Done() can only be called from the worker func and not the hooks.
		time.Sleep(time.Millisecond * 100)

		_ = jq.Shutdown(ctx)

		ensureJobHistoryTableRows(t, pg, 3)

		// ensure the job is finished with fail conditions
		var hJobs []gueJobHistory
		err = pgxscan.Select(ctx, pg, &hJobs, `SELECT * FROM public.gue_jobs_history`)
		assert.NoError(t, err)
		assert.Len(t, hJobs, 3)

		assert.False(t, hJobs[0].Success)
		assert.NotEmpty(t, hJobs[0].FinishedAt)
		assert.Equal(t, 0, hJobs[0].RunCount)
		assert.NotEmpty(t, hJobs[0].RunError)
		assert.Contains(t, hJobs[0].RunError, "arrower: job failed: ")
		assert.Contains(t, string(hJobs[0].Args), "argName", "args are json formatted: {\"Name\":\"argName\"}")

		assert.False(t, hJobs[1].Success)
		assert.NotEmpty(t, hJobs[1].FinishedAt)
		assert.Equal(t, 1, hJobs[1].RunCount)
		assert.NotEmpty(t, hJobs[1].RunError)
		assert.Contains(t, hJobs[1].RunError, "arrower: job failed: ")
		assert.Contains(t, string(hJobs[0].Args), "argName", "args are json formatted: {\"Name\":\"argName\"}")

		assert.True(t, hJobs[2].Success)
		assert.NotEmpty(t, hJobs[2].FinishedAt)
		assert.Equal(t, 2, hJobs[2].RunCount)
		assert.Empty(t, hJobs[2].RunError)
		assert.Contains(t, string(hJobs[0].Args), "argName", "args are json formatted: {\"Name\":\"argName\"}")
	})

	t.Run("ensure panicked workers are recorded in the gue_jobs_history table", func(t *testing.T) {
		t.Parallel()

		t.Skip() // gue does not run WithPoolHooksJobDone functions, if a panic occurs

		var (
			count int
			wg    sync.WaitGroup
		)

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(time.Nanosecond),
		)
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(ctx context.Context, j jobWithArgs) error {
			if count == 0 {
				count++

				panic("job panics with error")
			}

			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		// job history table is empty before first job is enqueued
		ensureJobHistoryTableRows(t, pg, 0)

		wg.Add(1)
		err = jq.Enqueue(ctx, jobWithArgs{})
		assert.NoError(t, err)

		wg.Wait() // waits for the worker

		// wait until the all worker hooks are processed. The use of a sync.WaitGroup does not work,
		// because Done() can only be called from the worker func and not the hooks.
		time.Sleep(time.Millisecond * 50)

		_ = jq.Shutdown(ctx)

		ensureJobHistoryTableRows(t, pg, 2)

		// ensure the job is finished with fail conditions
		var hJobs []gueJobHistory
		err = pgxscan.Select(ctx, pg, &hJobs, `SELECT * FROM public.gue_jobs_history`)
		assert.NoError(t, err)
		assert.Len(t, hJobs, 2)

		assert.False(t, hJobs[0].Success)
		assert.NotEmpty(t, hJobs[0].FinishedAt)
		assert.Equal(t, 0, hJobs[0].RunCount)
		assert.NotEmpty(t, hJobs[0].RunError)
		assert.Contains(t, hJobs[0].RunError, "arrower: job failed: ")

		assert.True(t, hJobs[1].Success)
		assert.NotEmpty(t, hJobs[1].FinishedAt)
		assert.Equal(t, 1, hJobs[1].RunCount)
		assert.Empty(t, hJobs[1].RunError)
	})
}

func TestGueHandler_Shutdown(t *testing.T) {
	t.Parallel()

	t.Run("ensure shutdown can be called without workers started", func(t *testing.T) {
		t.Parallel()

		jq, err := jobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pgHandler.PGx(),
			jobs.WithPollInterval(time.Nanosecond),
		)
		assert.NoError(t, err)

		err = jq.Shutdown(ctx)
		assert.NoError(t, err)
	})

	t.Run("ensure long running jobs persist after shutdown of workers", func(t *testing.T) {
		t.Parallel()
		t.Skip() // currently, the Group that gue uses does not allow to return before all jobs are finished

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(time.Nanosecond),
		)
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(ctx context.Context, j simpleJob) error {
			t.Log("Started long running job")

			for {
				time.Sleep(time.Second)
				select {
				case <-ctx.Done():
					// return nil
				case <-time.After(500 * time.Millisecond):
				}
			}
		})
		assert.NoError(t, err)

		err = jq.Enqueue(context.Background(), simpleJob{})
		assert.NoError(t, err)

		time.Sleep(100 * time.Millisecond) // wait a bit for the job to start processing

		err = jq.Shutdown(context.Background())
		assert.NoError(t, err)
		ensureJobTableRows(t, pg, 1)
		ensureJobHistoryTableRows(t, pg, 0)
	})

	t.Run("ensure long running jobs persist after restart of workers", func(t *testing.T) {
		t.Parallel()
		t.Skip() // currently the Group that gue uses does not allow to return before all jobs are finished

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(time.Nanosecond),
		)
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(context.Context, simpleJob) error {
			t.Log("Started long running job")
			time.Sleep(time.Minute * 60 * 24) // simulate long-running job

			return nil
		})
		assert.NoError(t, err)

		err = jq.Enqueue(context.Background(), simpleJob{})
		assert.NoError(t, err)

		time.Sleep(time.Millisecond) // wait a bit for the job to start processing
		err = jq.RegisterJobFunc(func(context.Context, jobWithArgs) error { return nil })
		assert.NoError(t, err)
		ensureJobTableRows(t, pg, 1)
		ensureJobHistoryTableRows(t, pg, 0)
	})
}

func TestGueHandler_Tx(t *testing.T) {
	t.Parallel()

	t.Run("ensure tx from ctx is used and job is not enqueued on rollback", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(time.Nanosecond),
		)
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(ctx context.Context, j simpleJob) error { return nil })
		assert.NoError(t, err)

		// create new transaction and set it in the context
		newCtx := context.Background()
		txHandle, err := pg.Begin(newCtx)
		assert.NoError(t, err)
		newCtx = context.WithValue(newCtx, postgres.CtxTX, txHandle)

		err = jq.Enqueue(newCtx, simpleJob{})
		assert.NoError(t, err)

		rb := txHandle.Rollback(newCtx)
		assert.NoError(t, rb)

		// ensure job table has no rows, as rollback happened
		ensureJobTableRows(t, pg, 0)

		_ = jq.Shutdown(newCtx)
	})

	t.Run("ensure tx from ctx is used and job is enqueued on commit", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(time.Second), // prevent an immediate start of the worker, so the assertions below work
		)
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(ctx context.Context, j simpleJob) error { return nil })
		assert.NoError(t, err)

		// create new transaction and set it in the context, same as the middleware does
		txHandle, err := pg.Begin(ctx)
		assert.NoError(t, err)
		newCtx := context.WithValue(ctx, postgres.CtxTX, txHandle)

		err = jq.Enqueue(newCtx, simpleJob{})
		assert.NoError(t, err)

		err = txHandle.Commit(newCtx)
		assert.NoError(t, err)

		ensureJobTableRows(t, pg, 1)

		_ = jq.Shutdown(newCtx)
	})

	t.Run("ensure tx is present in ctx in the JobFunc", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup
		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(alog.NewNoopLogger(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(time.Nanosecond),
		)
		assert.NoError(t, err)

		wg.Add(1)
		err = jq.RegisterJobFunc(func(ctx context.Context, j simpleJob) error {
			tx, txOk := ctx.Value(postgres.CtxTX).(pgx.Tx)
			assert.True(t, txOk)
			assert.NotEmpty(t, tx)

			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		_ = jq.Enqueue(ctx, simpleJob{})

		wg.Wait()
		_ = jq.Shutdown(ctx)
	})
}

func ensureJobTableRows(t *testing.T, db *pgxpool.Pool, num int) {
	t.Helper()

	var c int
	_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM public.gue_jobs;`).Scan(&c)

	assert.Equal(t, num, c)
}

func ensureJobHistoryTableRows(t *testing.T, db *pgxpool.Pool, num int) {
	t.Helper()

	var c int
	_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM public.gue_jobs_history;`).Scan(&c)

	assert.Equal(t, num, c)
}
