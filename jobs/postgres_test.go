//go:build integration

//nolint:wsl_v5
package jobs_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	mnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/trace"
	tnoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/contexts/auth"
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

func TestNewPostgresJobs(t *testing.T) {
	t.Parallel()

	t.Run("initialise a new PostgresJobsHandler", func(t *testing.T) {
		t.Parallel()

		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pgHandler.PGx())
		assert.NoError(t, err)
		assert.NotEmpty(t, jq)

		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)
	})

	t.Run("initialise a new PostgresJobsHandler with queue", func(t *testing.T) {
		t.Parallel()

		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pgHandler.PGx(),
			jobs.WithQueue("name"),
		)
		assert.NoError(t, err)
		assert.NotEmpty(t, jq)

		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)
	})

	t.Run("initialise a new PostgresJobsHandler with poll interval", func(t *testing.T) {
		t.Parallel()

		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pgHandler.PGx(),
			jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)
		assert.NotEmpty(t, jq)

		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)
	})

	t.Run("initialise a new PostgresJobsHandler with pool size", func(t *testing.T) {
		t.Parallel()

		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pgHandler.PGx(),
			jobs.WithPoolSize(1),
		)
		assert.NoError(t, err)
		assert.NotEmpty(t, jq)

		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)
	})

	t.Run("initialise a new PostgresJobsHandler with pool name", func(t *testing.T) {
		t.Parallel()

		logger := alog.Test(t)
		alog.Unwrap(logger).SetLevel(alog.LevelDebug)

		pg := pgHandler.NewTestDatabase()
		queries := models.New(pg)
		jq, err := jobs.NewPostgresJobs(logger, mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPoolName(argName), jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)
		assert.NotEmpty(t, jq)

		_ = jq.RegisterJobFunc(func(context.Context, simpleJob) error { return nil }) // register JobFunc to start workers
		_ = jq.Enqueue(t.Context(), simpleJob{})

		// wait for a worker to start and process the job
		assert.Eventually(t, func() bool {
			wp, er := queries.GetWorkerPools(t.Context())
			assert.NoError(t, er)

			return len(wp) == 1
		}, 15*time.Second, 100*time.Millisecond)

		wp, err := queries.GetWorkerPools(t.Context())
		assert.NoError(t, err)

		// expect one worker with the right name to be registered in the db
		assert.Len(t, wp, 1)
		assert.Equal(t, argName, wp[0].ID)

		logger.Contains("jobs.gue.client-id="+argName, "gue-logs should to contain the right name as client-id")

		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)
	})

	t.Run("queue options", func(t *testing.T) {
		t.Parallel()

		t.Run("with poll strategy", func(t *testing.T) {
			t.Parallel()

			var (
				wg    sync.WaitGroup
				mu    sync.Mutex
				order int
			)

			pg := pgHandler.NewTestDatabase()
			jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
				jobs.WithPollInterval(10*time.Millisecond),
				jobs.WithPoolSize(1), // prevent workers running in parallel
				jobs.WithPollStrategy(jobs.RunAtPollStrategy),
			)
			assert.NoError(t, err)

			// control the runAt time, so the queue considers the order as intended.
			now := time.Now().UTC()

			// The First Job has default priority and should be processed first
			err = jq.Enqueue(t.Context(), jobWithArgs{Name: "0"}, jobs.WithRunAt(now))
			assert.NoError(t, err)

			// The Second Job has higher priority but starts later
			err = jq.Enqueue(t.Context(), jobWithArgs{Name: "1"},
				jobs.WithRunAt(now.Add(time.Millisecond)), // db looses nanoseconds granularity => ms
				jobs.WithPriority(-1),
			)
			assert.NoError(t, err)

			wg.Add(2)
			err = jq.RegisterJobFunc(func(_ context.Context, job jobWithArgs) error {
				mu.Lock()
				defer mu.Unlock()

				if job.Name == "0" { // expect Job with Name 0 to be run first (^= 0) in order
					assert.Equal(t, 0, order)
				}

				if job.Name == "1" { // expect Job with Name 1 to be run afterwards ^= higher order
					assert.Equal(t, 1, order)
				}

				order++
				wg.Done()

				return nil
			})
			assert.NoError(t, err)

			wg.Wait()
			time.Sleep(200 * time.Millisecond)

			err = jq.Shutdown(t.Context())
			assert.NoError(t, err)
		})
	})
}

func TestPostgresJobs_RegisterJobFunc(t *testing.T) {
	t.Parallel()

	t.Run("ensure JobFunc is of type func and signature has two arguments with first ctx and returns error", func(t *testing.T) {
		t.Parallel()

		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pgHandler.PGx())
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(nil)
		assert.ErrorIs(t, err, jobs.ErrInvalidJobFunc)

		err = jq.RegisterJobFunc("")
		assert.ErrorIs(t, err, jobs.ErrInvalidJobFunc)

		err = jq.RegisterJobFunc(func() {})
		assert.ErrorIs(t, err, jobs.ErrInvalidJobFunc)

		err = jq.RegisterJobFunc(func(_ int, _ []byte) error { return nil })
		assert.ErrorIs(t, err, jobs.ErrInvalidJobFunc)

		// primitive types do not implement a JobType interface and have no struct name.
		err = jq.RegisterJobFunc(func(_ context.Context, _ []byte) error { return nil })
		assert.ErrorIs(t, err, jobs.ErrInvalidJobFunc)

		err = jq.RegisterJobFunc(func(_ context.Context, _ simpleJob) {})
		assert.ErrorIs(t, err, jobs.ErrInvalidJobFunc)

		err = jq.RegisterJobFunc(func(_ context.Context, _ simpleJob) int { return 0 })
		assert.ErrorIs(t, err, jobs.ErrInvalidJobFunc)

		err = jq.RegisterJobFunc(func(_ context.Context, _ simpleJob) error { return nil })
		assert.NoError(t, err)

		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)
	})

	t.Run("ensure only one worker can register for a JobType name", func(t *testing.T) {
		t.Parallel()

		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pgHandler.PGx())
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(_ context.Context, _ simpleJob) error { return nil })
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(_ context.Context, _ simpleJob) error { return nil })
		assert.ErrorIs(t, err, jobs.ErrInvalidJobFunc)

		err = jq.RegisterJobFunc(func(_ context.Context, _ jobWithSameNameAsSimpleJob) error { return nil })
		assert.ErrorIs(t, err, jobs.ErrInvalidJobFunc)

		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)
	})

	t.Run("ensure register takes a jobType from the interface implementation", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)

		wg.Add(1)
		err = jq.RegisterJobFunc(func(_ context.Context, job jobWithJobType) error {
			assert.Equal(t, jobWithJobType{Name: argName}, job)
			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		err = jq.Enqueue(t.Context(), jobWithJobType{Name: argName})
		assert.NoError(t, err)

		wg.Wait()
		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)
	})

	t.Run("ensure retries are done on an error", func(t *testing.T) {
		t.Parallel()

		var (
			wg         sync.WaitGroup
			mu         sync.Mutex
			retryCount int
		)

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)

		wg.Add(1)
		err = jq.RegisterJobFunc(func(_ context.Context, job jobWithJobType) error {
			mu.Lock()
			defer mu.Unlock()

			if retryCount < 1 {
				retryCount++

				return fmt.Errorf("some failure inside a job worker: %d", retryCount) //nolint:err113,err113
			}

			assert.Equal(t, jobWithJobType{Name: argName}, job)
			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		err = jq.Enqueue(t.Context(), jobWithJobType{Name: argName})
		assert.NoError(t, err)

		wg.Wait()
		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)
	})
}

func TestPostgresJobs_Enqueue(t *testing.T) {
	t.Parallel()

	t.Run("invalid job", func(t *testing.T) {
		t.Parallel()

		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pgHandler.PGx(),
			jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)

		err = jq.Enqueue(t.Context(), "")
		assert.ErrorIs(t, err, jobs.ErrInvalidJobType)

		err = jq.Enqueue(t.Context(), []byte(""))
		assert.ErrorIs(t, err, jobs.ErrInvalidJobType)

		err = jq.Enqueue(t.Context(), 0)
		assert.ErrorIs(t, err, jobs.ErrInvalidJobType)

		err = jq.Enqueue(t.Context(), nil)
		assert.ErrorIs(t, err, jobs.ErrInvalidJobType)

		err = jq.Enqueue(t.Context(), func() {})
		assert.ErrorIs(t, err, jobs.ErrInvalidJobType)

		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)
	})

	t.Run("single job", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup

		pg := pgHandler.NewTestDatabase()

		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), trace.NewTracerProvider(), pg,
			jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)

		wg.Add(1)
		err = jq.RegisterJobFunc(func(_ context.Context, job jobWithArgs) error {
			assert.Equal(t, payloadJob, job)
			assert.Equal(t, argName, job.Name)
			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		err = jq.Enqueue(t.Context(), payloadJob)
		assert.NoError(t, err)

		wg.Wait()
		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)
	})

	t.Run("slice of same job type, struct name and custom", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(_ context.Context, job jobWithArgs) error {
			assert.Equal(t, argName, job.Name)
			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(_ context.Context, job jobWithJobType) error {
			assert.Equal(t, argName, job.Name)
			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		wg.Add(2)
		err = jq.Enqueue(t.Context(), []jobWithArgs{{Name: argName}, {Name: argName}})
		assert.NoError(t, err)

		wg.Add(2)
		err = jq.Enqueue(t.Context(), []jobWithJobType{{Name: argName}, {Name: argName}})
		assert.NoError(t, err)

		wg.Wait()                          // all workers are done, and now:
		time.Sleep(800 * time.Millisecond) // wait until gue finishes with the underlying transaction

		ensureJobTableRows(t, pg, 0+1)      // all Jobs are processed and cron jobs are left
		ensureJobHistoryTableRows(t, pg, 4) // history has all Jobs

		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)
	})

	t.Run("slice of different job types", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(_ context.Context, job jobWithArgs) error {
			assert.Equal(t, argName, job.Name)
			wg.Done()

			return nil
		})
		assert.NoError(t, err)
		err = jq.RegisterJobFunc(func(_ context.Context, job jobWithJobType) error {
			assert.Equal(t, argName, job.Name)
			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		wg.Add(2)
		err = jq.Enqueue(t.Context(), []any{jobWithArgs{Name: argName}, jobWithJobType{Name: argName}})
		assert.NoError(t, err)

		wg.Wait()                          // all workers are done, and now:
		time.Sleep(200 * time.Millisecond) // wait until gue finishes with the underlying transaction

		ensureJobTableRows(t, pg, 0+1)      // all Jobs are processed and cron jobs are left
		ensureJobHistoryTableRows(t, pg, 2) // history has all Jobs

		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)
	})

	t.Run("ensure priority is set", func(t *testing.T) {
		t.Parallel()

		var (
			wg    sync.WaitGroup
			mu    sync.Mutex
			order int
		)

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(10*time.Millisecond), jobs.WithPoolSize(1),
		)
		assert.NoError(t, err)

		// enforce the same runAt time, so the queue considers the priority as the second argument to order with.
		runAt := jobs.WithRunAt(time.Now().UTC())

		// The First Job has default priority
		err = jq.Enqueue(t.Context(), jobWithArgs{Name: "1"}, runAt)
		assert.NoError(t, err)

		// The Second Job has higher priority and should be processed first
		err = jq.Enqueue(t.Context(), jobWithArgs{Name: "0"}, runAt, jobs.WithPriority(-1))
		assert.NoError(t, err)

		wg.Add(2)

		err = jq.RegisterJobFunc(func(_ context.Context, job jobWithArgs) error {
			mu.Lock()
			defer mu.Unlock()

			if job.Name == "0" { // expect Job with Name 0 to be run first (^= 0) in order
				assert.Equal(t, 0, order)
			}

			if job.Name == "1" { // expect Job with Name 1 to be run afterwards ^= higher order
				assert.Equal(t, 1, order)
			}

			order++
			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		wg.Wait()
		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)
	})
}

func TestPostgresJobsHandler_Schedule(t *testing.T) {
	t.Parallel()

	t.Run("schedule a task", func(t *testing.T) {
		t.Parallel()

		var (
			wg      sync.WaitGroup
			mu      sync.Mutex
			counter int
		)

		logger := alog.Test(t)
		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(logger, mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)

		const numScheduledJobsToMonitor = 2
		err = jq.RegisterJobFunc(func(_ context.Context, _ simpleJob) error {
			mu.Lock()
			defer mu.Unlock()

			// prevent calling Done() on a finished wg, panic: WaitGroup is reused before the previous Wait has returned
			if counter < numScheduledJobsToMonitor {
				wg.Done()
			}

			counter++

			return nil
		})
		assert.NoError(t, err)

		err = jq.Schedule("@every 1ms", simpleJob{})
		assert.NoError(t, err)

		wg.Add(numScheduledJobsToMonitor)
		wg.Wait()                          // all workers are done, and now:
		time.Sleep(200 * time.Millisecond) // wait until gue finishes with the underlying transaction
		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)

		logger.NotContains("recording job worker's git hash failed")

		assert.Eventually(t, func() bool {
			var c int
			err = pgxscan.Get(t.Context(), pg, &c, `SELECT COUNT(*) FROM arrower.gue_jobs_history`)
			assert.NoError(t, err)

			return c >= numScheduledJobsToMonitor
		}, 15*time.Second, time.Second)

		var hJobs []gueJobHistory
		err = pgxscan.Select(t.Context(), pg, &hJobs, `SELECT * FROM arrower.gue_jobs_history`)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(hJobs), numScheduledJobsToMonitor)
		assert.Contains(t, hJobs[0].JobType, "simpleJob")
		assert.Contains(t, hJobs[1].JobType, "simpleJob")

		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)
	})

	t.Run("gueron job works with arrower custom payload", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		logger := alog.Test(t)
		alog.Unwrap(logger).SetLevel(alog.LevelInfo)
		jq, err := jobs.NewPostgresJobs(logger, mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(_ context.Context, _ simpleJob) error { return nil })
		assert.NoError(t, err)

		err = jq.Schedule("@every 1ms", simpleJob{})
		assert.NoError(t, err)

		// run the cron now
		assert.Eventually(t, func() bool {
			c, er := pg.Exec(t.Context(), `UPDATE arrower.gue_jobs SET run_at = $1 WHERE job_type = $2;`, "2023-06-20 19:35:27-01", "gueron-refresh-schedule")
			assert.NoError(t, er)

			return int64(1) == c.RowsAffected()
		}, 15*time.Second, time.Second)

		assert.Eventually(t, func() bool {
			var c int
			err = pgxscan.Get(t.Context(), pg, &c, `SELECT COUNT(*) FROM arrower.gue_jobs_history WHERE job_type='gueron-refresh-schedule'`)
			assert.NoError(t, err)

			return c == 1
		}, 15*time.Second, time.Second)

		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)

		//
		// ensure the scheduler run and the payload is augmented by arrower and it not pure gueron payload
		logger.NotContains("git hash failed", "gueron should deserialise into the arrower job payload without issue")

		var hJobs []gueJobHistory
		err = pgxscan.Select(t.Context(), pg, &hJobs, `SELECT * FROM arrower.gue_jobs_history WHERE job_type='gueron-refresh-schedule'`)
		assert.NoError(t, err)
		assert.Empty(t, hJobs[0].RunError)
		assert.Contains(t, string(hJobs[0].Args), "gitHashProcessed", "gueron does not have the arrower payload, but should at least record what is possible")
		assert.NotContains(t, string(hJobs[0].Args), `"jobData":null`)
		assert.Contains(t, string(hJobs[0].Args), `"jobData":{}`)
	})

	t.Run("add schedule after queue already started", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		logger := alog.Test(t)
		alog.Unwrap(logger).SetLevel(alog.LevelInfo)
		jq, err := jobs.NewPostgresJobs(logger, mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(_ context.Context, _ simpleJob) error { return nil })
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(_ context.Context, _ jobWithArgs) error { return nil })
		assert.NoError(t, err)

		err = jq.Schedule("@every 1ms", simpleJob{})
		assert.NoError(t, err)

		//
		// wait for queue to start processing
		assert.Eventually(t, func() bool {
			var c int
			_ = pg.QueryRow(t.Context(), `SELECT COUNT(*) FROM arrower.gue_jobs_history WHERE job_type='arrower/jobs_test.simpleJob';`).Scan(&c)
			return c > 1 //nolint:nlreturn
		}, 15*time.Second, time.Second)

		//
		// add a schedule on started queue
		err = jq.Schedule("@every 1ms", jobWithArgs{})
		assert.NoError(t, err)

		assert.Eventually(t, func() bool {
			var c int
			_ = pg.QueryRow(t.Context(), `SELECT COUNT(*) FROM arrower.gue_jobs_history WHERE job_type='arrower/jobs_test.jobWithArgs';`).Scan(&c)
			return c > 1 //nolint:nlreturn
		}, 15*time.Second, time.Second)

		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)
	})
}

func TestPostgresJobs_StartWorkers(t *testing.T) {
	t.Parallel()

	t.Run("restart queue if RegisterJobFunc is called after start", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)

		_ = jq.RegisterJobFunc(func(context.Context, simpleJob) error { return nil })
		time.Sleep(time.Millisecond) // wait until the workers have started

		err = jq.Enqueue(t.Context(), payloadJob)
		assert.NoError(t, err)

		wg.Add(1)
		err = jq.RegisterJobFunc(func(_ context.Context, job jobWithArgs) error {
			assert.Equal(t, payloadJob, job)
			assert.Equal(t, argName, job.Name)
			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		wg.Wait()
		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)
	})

	t.Run("queue starts only after poll interval time, after the last registered job func", func(t *testing.T) {
		t.Parallel()

		// To understand if the job queue has started its underlying worker pool already, this test relies on the
		// log output from RegisterJobFunc in case it is restarting.
		logger := alog.Test(t)
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
		logger.Empty()

		time.Sleep(time.Millisecond) // wait a bit, but not longer than the pollInterval

		// register second worker, expect no restart of the queue, as it is not started yet
		err = jq.RegisterJobFunc(func(context.Context, jobWithArgs) error { return nil })
		assert.NoError(t, err)
		logger.Empty()
		logger.NotContains(`msg="restart workers"`)

		time.Sleep(10 * pollInterval) // wait longer than the pollInterval, so the queue is started

		// register third worker, expect restart of the queue
		err = jq.RegisterJobFunc(func(context.Context, jobWithJobType) error { return nil })
		assert.NoError(t, err)
		logger.NotEmpty()
		logger.Contains(`msg="restart workers"`)

		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)
	})

	t.Run("ensure worker pool is registered & unregistered", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		queries := models.New(pg)
		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPoolName("pool_name"), jobs.WithPoolSize(7), jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)

		// pool only starts after a call to RegisterJobFunc
		err = jq.RegisterJobFunc(func(context.Context, simpleJob) error { return nil })
		assert.NoError(t, err)
		// wait for the startWorkers to register itself as online
		time.Sleep(200 * time.Millisecond)

		wp, err := queries.GetWorkerPools(t.Context())
		assert.NoError(t, err)
		assert.Len(t, wp, 1)
		assert.Equal(t, "pool_name", wp[0].ID)
		assert.Equal(t, int16(7), wp[0].Workers)

		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)

		wp, err = queries.GetWorkerPools(t.Context())
		assert.NoError(t, err)
		assert.Len(t, wp, 1)
		assert.Equal(t, int16(0), wp[0].Workers)
	})
}

func TestPostgresJobs_History(t *testing.T) {
	t.Parallel()

	t.Run("ensure successful jobs are recorded into gue_jobs_history table", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(_ context.Context, _ jobWithArgs) error { return nil })
		assert.NoError(t, err)

		// job history table is empty before the first Job is enqueued
		ensureJobHistoryTableRows(t, pg, 0)

		err = jq.Enqueue(t.Context(), jobWithArgs{})
		assert.NoError(t, err)

		// Wait until the worker & all it's hooks are processed. The use of a sync.WaitGroup in the JobFunc does not work,
		// because wg.Done() can only be called from the worker func and not the hooks (where it would need to be placed).
		time.Sleep(time.Millisecond * 400)

		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)

		// job history table contains the finished Job
		ensureJobHistoryTableRows(t, pg, 1)

		// ensure the Job is finished successful
		var hJob gueJobHistory
		err = pgxscan.Get(t.Context(), pg, &hJob, `SELECT * FROM arrower.gue_jobs_history`)
		assert.NoError(t, err)

		assert.True(t, hJob.Success)
		assert.NotEmpty(t, hJob.FinishedAt)
		assert.Equal(t, 0, hJob.RunCount)
		assert.Empty(t, hJob.RunError)
		assert.NotEqual(t, hJob.CreatedAt, *hJob.FinishedAt)
		assert.Empty(t, hJob.PrunedAt)
	})

	t.Run("ensure failed jobs are recorded in gue_jobs_history table", func(t *testing.T) {
		t.Parallel()

		var (
			count int
			wg    sync.WaitGroup
		)

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(_ context.Context, _ jobWithArgs) error {
			if count < 2 { // fail twice
				count++

				return errors.New("job returns with error") //nolint:err113,err113
			}

			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		// job history table is empty before the first Job is enqueued
		ensureJobHistoryTableRows(t, pg, 0)

		wg.Add(1)
		err = jq.Enqueue(t.Context(), jobWithArgs{Name: "argName"})
		assert.NoError(t, err)

		wg.Wait() // wait for the worker

		// wait until the all worker hooks are processed. The use of a sync.WaitGroup does not work,
		// because Done() can only be called from the worker func and not the hooks.
		time.Sleep(time.Millisecond * 100)

		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)

		ensureJobHistoryTableRows(t, pg, 3)

		// ensure the Job is finished with fail conditions
		var hJobs []gueJobHistory
		err = pgxscan.Select(t.Context(), pg, &hJobs, `SELECT * FROM arrower.gue_jobs_history`)
		assert.NoError(t, err)
		assert.Len(t, hJobs, 3)

		assert.False(t, hJobs[0].Success)
		assert.NotEmpty(t, hJobs[0].FinishedAt)
		assert.Equal(t, 0, hJobs[0].RunCount)
		assert.NotEmpty(t, hJobs[0].RunError)
		assert.Contains(t, hJobs[0].RunError, "arrower: job failed: ")
		assert.Equal(t, "argName", jobWithArgsFromDBSerialisation(t, hJobs[0].Args).Name)

		assert.False(t, hJobs[1].Success)
		assert.NotEmpty(t, hJobs[1].FinishedAt)
		assert.Equal(t, 1, hJobs[1].RunCount)
		assert.NotEmpty(t, hJobs[1].RunError)
		assert.Contains(t, hJobs[1].RunError, "arrower: job failed: ")
		assert.Equal(t, "argName", jobWithArgsFromDBSerialisation(t, hJobs[1].Args).Name)

		assert.True(t, hJobs[2].Success)
		assert.NotEmpty(t, hJobs[2].FinishedAt)
		assert.Equal(t, 2, hJobs[2].RunCount)
		assert.Empty(t, hJobs[2].RunError)
		assert.Equal(t, "argName", jobWithArgsFromDBSerialisation(t, hJobs[2].Args).Name)
	})

	t.Run("ensure panicked workers are recorded in the gue_jobs_history table", func(t *testing.T) {
		t.Parallel()

		var (
			count int
			wg    sync.WaitGroup
		)

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(_ context.Context, _ jobWithArgs) error {
			if count == 0 {
				count++

				panic("job panics with error")
			}

			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		// job history table is empty before the first Job is enqueued
		ensureJobHistoryTableRows(t, pg, 0)

		wg.Add(1)
		err = jq.Enqueue(t.Context(), jobWithArgs{})
		assert.NoError(t, err)

		wg.Wait() // waits for the worker

		// Wait until the all worker hooks are processed. The use of a sync.WaitGroup does not work,
		// because Done() can only be called from the worker func and not the hooks.
		time.Sleep(time.Millisecond * 200)

		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)

		ensureJobHistoryTableRows(t, pg, 2)

		// ensure the Job is finished with fail conditions
		var hJobs []gueJobHistory
		err = pgxscan.Select(t.Context(), pg, &hJobs, `SELECT * FROM arrower.gue_jobs_history`)
		assert.NoError(t, err)
		assert.Len(t, hJobs, 2)

		assert.False(t, hJobs[0].Success)
		assert.NotEmpty(t, hJobs[0].FinishedAt)
		assert.Equal(t, 0, hJobs[0].RunCount)
		assert.NotEmpty(t, hJobs[0].RunError)
		assert.Contains(t, hJobs[0].RunError, "job panicked:")

		assert.True(t, hJobs[1].Success)
		assert.NotEmpty(t, hJobs[1].FinishedAt)
		assert.Equal(t, 1, hJobs[1].RunCount)
		assert.Empty(t, hJobs[1].RunError)
	})
}

func TestPostgresJobs_Shutdown(t *testing.T) {
	t.Parallel()

	t.Run("ensure shutdown can be called without workers started", func(t *testing.T) {
		t.Parallel()

		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pgHandler.PGx(),
			jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)

		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)
	})

	t.Run("ensure long running jobs persist after shutdown of workers", func(t *testing.T) {
		t.Parallel()
		t.Skip() // currently, the Group that gue uses does not allow returning before all Jobs are finished

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(ctx context.Context, _ simpleJob) error {
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

		err = jq.Enqueue(t.Context(), simpleJob{})
		assert.NoError(t, err)

		time.Sleep(100 * time.Millisecond) // wait a bit for the Job to start processing

		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)
		ensureJobTableRows(t, pg, 1)
		ensureJobHistoryTableRows(t, pg, 0)
	})

	t.Run("ensure long running jobs persist after restart of workers", func(t *testing.T) {
		t.Parallel()
		t.Skip() // currently, the Group that gue uses does not allow returning before all Jobs are finished

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(context.Context, simpleJob) error {
			t.Log("Started long running job")
			time.Sleep(time.Minute * 60 * 24) // simulate a long-running Job

			return nil
		})
		assert.NoError(t, err)

		err = jq.Enqueue(t.Context(), simpleJob{})
		assert.NoError(t, err)

		time.Sleep(time.Millisecond) // wait a bit for the Job to start processing
		err = jq.RegisterJobFunc(func(context.Context, jobWithArgs) error { return nil })
		assert.NoError(t, err)
		ensureJobTableRows(t, pg, 1)
		ensureJobHistoryTableRows(t, pg, 0)

		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)
	})
}

func TestPostgresJobs_Tx(t *testing.T) {
	t.Parallel()

	t.Run("ensure tx from ctx is used and job is not enqueued on rollback", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(_ context.Context, _ simpleJob) error { return nil })
		assert.NoError(t, err)

		// create a new transaction and set it in the context
		newCtx := t.Context()
		txHandle, err := pg.Begin(newCtx)
		assert.NoError(t, err)
		newCtx = context.WithValue(newCtx, postgres.CtxTX, txHandle)

		err = jq.Enqueue(newCtx, simpleJob{})
		assert.NoError(t, err)

		rb := txHandle.Rollback(newCtx)
		assert.NoError(t, rb)

		// ensure Job table has no rows, as rollback happened
		ensureJobTableRows(t, pg, 0)

		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)
	})

	t.Run("ensure tx from ctx is used and job is enqueued on commit", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(time.Second), // prevent an immediate start of the worker, so the assertions below work
		)
		assert.NoError(t, err)

		err = jq.RegisterJobFunc(func(_ context.Context, _ simpleJob) error { return nil })
		assert.NoError(t, err)

		// create new transaction and set it in the context, same as the middleware does
		txHandle, err := pg.Begin(t.Context())
		assert.NoError(t, err)
		newCtx := context.WithValue(t.Context(), postgres.CtxTX, txHandle)

		err = jq.Enqueue(newCtx, simpleJob{})
		assert.NoError(t, err)

		err = txHandle.Commit(newCtx)
		assert.NoError(t, err)

		ensureJobTableRows(t, pg, 1)

		err = jq.Shutdown(newCtx)
		assert.NoError(t, err)
	})

	t.Run("ensure tx is present in ctx in the JobFunc", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup
		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)
		_, err = pg.Exec(t.Context(), `CREATE TABLE IF NOT EXISTS some_table (id SERIAL PRIMARY KEY);`)
		assert.NoError(t, err)

		wg.Add(1)
		err = jq.RegisterJobFunc(func(ctx context.Context, _ simpleJob) error {
			tx, txOk := ctx.Value(postgres.CtxTX).(pgx.Tx)
			assert.True(t, txOk)
			assert.NotEmpty(t, tx)

			_, funcErr := tx.Exec(ctx, `INSERT INTO some_table(id) VALUES (DEFAULT);`)
			assert.NoError(t, funcErr)

			var ids []int
			funcErr = pgxscan.Select(ctx, tx, &ids, `SELECT * FROM some_table;`)
			assert.NoError(t, funcErr)
			assert.Len(t, ids, 1)

			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		err = jq.Enqueue(t.Context(), simpleJob{})
		assert.NoError(t, err)

		wg.Wait()
		time.Sleep(400 * time.Millisecond)

		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)

		var ids []int
		err = pgxscan.Select(t.Context(), pg, &ids, `SELECT * FROM some_table;`)
		assert.NoError(t, err)
		assert.Len(t, ids, 1, "job inserts into db in transaction")

		ensureJobTableRows(t, pg, 1) // gueron
		ensureJobHistoryTableRows(t, pg, 1)
	})

	t.Run("ensure tx is rolled back", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup
		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)
		_, err = pg.Exec(t.Context(), `CREATE TABLE IF NOT EXISTS some_table (id SERIAL PRIMARY KEY);`)
		assert.NoError(t, err)

		wg.Add(1)
		err = jq.RegisterJobFunc(func(ctx context.Context, _ simpleJob) error {
			tx, _ := ctx.Value(postgres.CtxTX).(pgx.Tx)
			assert.NotEmpty(t, tx)

			_, funcErr := tx.Exec(ctx, `INSERT INTO some_table(id) VALUES (DEFAULT);`)
			assert.NoError(t, funcErr)

			var ids []int
			funcErr = pgxscan.Select(ctx, tx, &ids, `SELECT * FROM some_table;`)
			assert.NoError(t, funcErr)
			assert.Len(t, ids, 1)

			wg.Done()

			return errors.New("some-error") //nolint:err113,err113
		})
		assert.NoError(t, err)

		err = jq.Enqueue(t.Context(), simpleJob{})
		assert.NoError(t, err)

		wg.Wait()
		time.Sleep(400 * time.Millisecond)

		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)

		var ids []int
		err = pgxscan.Select(t.Context(), pg, &ids, `SELECT * FROM some_table;`)
		assert.NoError(t, err)
		assert.Empty(t, ids, "tx got rolled back")

		ensureJobTableRows(t, pg, 1+1) // one job + gueron
		ensureJobHistoryTableRows(t, pg, 1)
	})
}

func TestPostgresJobs_Instrumentation(t *testing.T) {
	t.Parallel()

	t.Run("ensure worker has ctx data set after persistence", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)

		wg.Add(1)
		err = jq.RegisterJobFunc(func(ctx context.Context, _ simpleJob) error {
			attr := alog.FromContext(ctx)
			assert.NotEmpty(t, attr)
			assert.Equal(t, "job", attr[0].Key)
			assert.Equal(t, "id", attr[0].Value.Group()[0].Key)
			assert.NotEmpty(t, attr[0].Value, "ctx needs a job_id as slog attribute")

			jobID, exists := jobs.FromContext(ctx)
			assert.True(t, exists)
			assert.NotEmpty(t, jobID, "ctx needs a jobID")

			userID, ok := ctx.Value(auth.CtxUserID).(string)
			assert.True(t, ok)
			assert.Equal(t, "user-id", userID)

			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		err = jq.Enqueue(context.WithValue(t.Context(), auth.CtxUserID, "user-id"), simpleJob{})
		assert.NoError(t, err)

		wg.Wait()
		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)
	})

	t.Run("ensure ctx without userID had none set", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)

		wg.Add(1)
		err = jq.RegisterJobFunc(func(ctx context.Context, _ simpleJob) error {
			userID := ctx.Value(auth.CtxUserID)
			assert.Nil(t, userID, "if userID is not set, prevent invalid value to make uuid.MustParse work")

			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		err = jq.Enqueue(t.Context(), simpleJob{})
		assert.NoError(t, err)

		wg.Wait()
		err = jq.Shutdown(t.Context())
		assert.NoError(t, err)
	})

	t.Run("args contains version information", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup

		pg := pgHandler.NewTestDatabase()
		jq, err := jobs.NewPostgresJobs(slog.New(slog.DiscardHandler), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), pg,
			jobs.WithPollInterval(10*time.Millisecond),
		)
		assert.NoError(t, err)

		wg.Add(1)
		err = jq.RegisterJobFunc(func(_ context.Context, _ simpleJob) error {
			wg.Done()

			return nil
		})
		assert.NoError(t, err)

		err = jq.Enqueue(t.Context(), simpleJob{})
		assert.NoError(t, err)

		wg.Wait() // waits for the worker

		// Wait until the all worker hooks are processed. The use of a sync.WaitGroup does not work,
		// because Done() can only be called from the worker func and not the hooks.
		time.Sleep(time.Millisecond * 200)

		var hJobs []gueJobHistory
		err = pgxscan.Select(t.Context(), pg, &hJobs, `SELECT * FROM arrower.gue_jobs_history`)
		assert.NoError(t, err)
		assert.Len(t, hJobs, 1)

		payload := jobs.PersistencePayload{}
		err = json.Unmarshal(hJobs[0].Args, &payload)
		assert.NoError(t, err)
		assert.NotEmpty(t, payload.GitHashEnqueued)
		assert.Equal(t, "unknown", payload.GitHashEnqueued, "while run with `go test` the version is not in the binary")
		assert.NotEmpty(t, payload.GitHashProcessed)
		assert.Equal(t, "unknown", payload.GitHashProcessed, "while run with `go test` the version is not in the binary")
	})
}

func ensureJobTableRows(t *testing.T, db *pgxpool.Pool, num int) {
	t.Helper()

	var c int
	assert.Eventually(t, func() bool {
		_ = db.QueryRow(t.Context(), `SELECT COUNT(*) FROM arrower.gue_jobs;`).Scan(&c)
		return c == num
	}, 15*time.Second, time.Second)
}

func ensureJobHistoryTableRows(t *testing.T, db *pgxpool.Pool, num int) {
	t.Helper()

	var c int
	assert.Eventually(t, func() bool {
		_ = db.QueryRow(t.Context(), `SELECT COUNT(*) FROM arrower.gue_jobs_history;`).Scan(&c)
		return num == c
	}, 15*time.Second, time.Second)
}

func jobWithArgsFromDBSerialisation(t *testing.T, rawPayload []byte) jobWithArgs {
	t.Helper()

	payload := jobs.PersistencePayload{}
	argsP := jobWithArgs{}

	err := json.Unmarshal(rawPayload, &payload)
	assert.NoError(t, err)

	b, err := json.Marshal(payload.JobData)
	assert.NoError(t, err)
	err = json.Unmarshal(b, &argsP)
	assert.NoError(t, err)

	return argsP
}
