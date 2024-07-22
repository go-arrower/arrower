package jobs_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/jobs"
)

func TestNewTestingJobs(t *testing.T) {
	t.Parallel()

	t.Run("initialise a new job queue for testing", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewTestingJobs()
		assert.NotEmpty(t, jq)
	})
}

func TestNewInMemoryJobs(t *testing.T) {
	t.Parallel()

	var wg sync.WaitGroup

	jq := jobs.NewInMemoryJobs()
	err := jq.RegisterJobFunc(func(_ context.Context, job jobWithArgs) error {
		assert.Equal(t, argName, job.Name)
		wg.Done()

		return nil
	})
	assert.NoError(t, err)

	wg.Add(2)

	err = jq.Enqueue(ctx, []jobWithArgs{{Name: argName}, {Name: argName}})
	assert.NoError(t, err)

	wg.Wait()

	jq.Assert(t).Empty()

	_ = jq.Shutdown(ctx)
}

func TestInMemoryHandler_Enqueue(t *testing.T) {
	t.Parallel()

	t.Run("invalid job", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewTestingJobs()

		err := jq.Enqueue(ctx, nil)
		assert.Error(t, err)
		assert.Equal(t, jobs.ErrInvalidJobType, err)
	})

	t.Run("simple job", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewTestingJobs()
		jassert := jq.Assert(t)
		jassert.Empty()

		_ = jq.Enqueue(ctx, simpleJob{})

		jassert.NotEmpty()
		jassert.Queued(simpleJob{}, 1)
	})

	t.Run("slice of same jobs", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewTestingJobs()
		jassert := jq.Assert(t)

		_ = jq.Enqueue(ctx, []jobWithArgs{{Name: argName}, {Name: argName}})
		jassert.Queued(jobWithArgs{}, 2)

		_ = jq.Enqueue(ctx, []simpleJob{{}, {}})
		jassert.Queued(jobWithSameNameAsSimpleJob{}, 2)
	})

	t.Run("slice of different job types", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewTestingJobs()
		jassert := jq.Assert(t)

		_ = jq.Enqueue(ctx, []any{jobWithArgs{Name: argName}, jobWithJobType{Name: argName}})
		jassert.Queued(jobWithArgs{}, 1)
		jassert.Queued(jobWithJobType{}, 1)
	})

	t.Run("enqueue safely concurrently", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup
		totalProducers := 1000

		jq := jobs.NewTestingJobs()

		wg.Add(totalProducers)
		for i := 0; i < totalProducers; i++ {
			go func() {
				_ = jq.Enqueue(ctx, simpleJob{})
				wg.Done()
			}()
		}

		wg.Wait()
		jq.Assert(t).QueuedTotal(totalProducers)
	})
}

func TestInMemoryQueue_Schedule(t *testing.T) {
	t.Parallel()

	t.Run("invalid schedule", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewTestingJobs()

		err := jq.Schedule("", simpleJob{})
		assert.Error(t, err)
		assert.ErrorIs(t, err, jobs.ErrScheduleFailed)
	})

	t.Run("invalid job", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewTestingJobs()

		err := jq.Schedule("@every 1ms", nil)
		assert.Error(t, err)
	})

	t.Run("run cron", func(t *testing.T) {
		t.Parallel()

		var (
			mu      sync.Mutex
			counter int
		)

		jq := jobs.NewTestingJobs()
		err := jq.RegisterJobFunc(func(_ context.Context, job jobWithArgs) error {
			mu.Lock()
			defer mu.Unlock()

			counter++
			assert.NotEmpty(t, job.Name)

			return nil
		})
		assert.NoError(t, err)

		err = jq.Schedule("@every 1ms", jobWithArgs{Name: gofakeit.Name()})
		assert.NoError(t, err)
		err = jq.Schedule("@every 1ms", jobWithArgs{Name: gofakeit.Name()})
		assert.NoError(t, err)

		jq.Start(ctx)

		// it looks like the cron library ignores 1ms values and always ticks at a second mark
		time.Sleep(2100 * time.Millisecond)

		assert.GreaterOrEqual(t, counter, 4)
	})
}

func TestInMemoryHandler_Reset(t *testing.T) {
	t.Parallel()

	t.Run("reset empty queue", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewTestingJobs()

		jq.Reset()
		jq.Assert(t).QueuedTotal(0)
	})

	t.Run("reset full queue", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewTestingJobs()
		jassert := jq.Assert(t)

		_ = jq.Enqueue(ctx, simpleJob{})
		_ = jq.Enqueue(ctx, simpleJob{})
		_ = jq.Enqueue(ctx, jobWithArgs{})
		jassert.QueuedTotal(3)

		jq.Reset()
		jassert.QueuedTotal(0)
	})
}

func TestInMemoryHandler_GetFirst(t *testing.T) {
	t.Parallel()

	t.Run("no job as queue is empty", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewTestingJobs()

		assert.Nil(t, jq.GetFirst())
	})

	t.Run("get first job", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewTestingJobs()
		_ = jq.Enqueue(ctx, []jobs.Job{jobWithArgs{Name: argName}, jobWithArgs{Name: "otherName"}})

		j := jq.GetFirst().(jobWithArgs)
		assert.Equal(t, argName, j.Name)
	})
}

func TestInMemoryHandler_Get(t *testing.T) {
	t.Parallel()

	t.Run("no job as queue is empty", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewTestingJobs()

		assert.Nil(t, jq.Get(0))
		assert.Nil(t, jq.Get(1))
	})

	t.Run("get second job", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewTestingJobs()
		_ = jq.Enqueue(ctx, []jobs.Job{jobWithArgs{Name: argName}, jobWithArgs{Name: "otherName"}})

		j := jq.Get(2).(jobWithArgs)
		assert.Equal(t, "otherName", j.Name)
	})
}

func TestInMemoryHandler_GetFirstOf(t *testing.T) {
	t.Parallel()

	t.Run("no job as queue is empty", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewTestingJobs()

		assert.Nil(t, jq.GetFirstOf(nil))
		assert.Nil(t, jq.GetFirstOf(simpleJob{}))
	})

	t.Run("get first job", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewTestingJobs()
		_ = jq.Enqueue(ctx, []jobs.Job{simpleJob{}, jobWithArgs{Name: argName}})

		j := jq.GetFirstOf(jobWithArgs{}).(jobWithArgs)
		assert.Equal(t, argName, j.Name)
	})
}

func TestInMemoryHandler_GetOf(t *testing.T) {
	t.Parallel()

	t.Run("no job as queue is empty", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewTestingJobs()

		assert.Nil(t, jq.GetOf(simpleJob{}, 0))
		assert.Nil(t, jq.GetOf(simpleJob{}, 1))
	})

	t.Run("get second job of type", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewTestingJobs()
		_ = jq.Enqueue(ctx, []jobs.Job{
			simpleJob{},
			simpleJob{},
			jobWithArgs{Name: "someArg"},
			jobWithArgs{Name: argName},
			jobWithArgs{Name: "someOther"},
		})

		j := jq.GetOf(jobWithArgs{}, 2).(jobWithArgs)
		assert.Equal(t, argName, j.Name)
	})
}

func TestInMemoryHandler_Assert(t *testing.T) {
	t.Parallel()

	jq := jobs.NewTestingJobs()

	jassert := jq.Assert(t)
	assert.NotEmpty(t, jassert)
}

func TestInMemoryAssertions_Empty(t *testing.T) {
	t.Parallel()

	t.Run("empty queue", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewTestingJobs()
		jassert := jq.Assert(new(testing.T))

		pass := jassert.Empty()
		assert.True(t, pass)
	})

	t.Run("not empty queue", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewTestingJobs()
		jassert := jq.Assert(new(testing.T))

		_ = jq.Enqueue(ctx, simpleJob{})

		pass := jassert.Empty()
		assert.False(t, pass)
	})
}

func TestInMemoryAssertions_NotEmpty(t *testing.T) {
	t.Parallel()

	t.Run("empty queue", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewTestingJobs()
		jassert := jq.Assert(new(testing.T))

		pass := jassert.NotEmpty()
		assert.False(t, pass)
	})

	t.Run("not empty queue", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewTestingJobs()
		jassert := jq.Assert(new(testing.T))

		_ = jq.Enqueue(ctx, simpleJob{})

		pass := jassert.NotEmpty()
		assert.True(t, pass)
	})
}

func TestInMemoryAssertions_Queued(t *testing.T) {
	t.Parallel()

	t.Run("no job of type queued", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewTestingJobs()
		jassert := jq.Assert(new(testing.T))

		pass := jassert.Queued(simpleJob{}, 0)
		assert.True(t, pass)
	})

	t.Run("job of type queued", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewTestingJobs()
		jassert := jq.Assert(new(testing.T))

		_ = jq.Enqueue(ctx, jobWithArgs{})
		_ = jq.Enqueue(ctx, simpleJob{})
		_ = jq.Enqueue(ctx, simpleJob{})

		pass := jassert.Queued(simpleJob{}, 2)
		assert.True(t, pass)
	})

	t.Run("different amount of jobs queued", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewTestingJobs()
		jassert := jq.Assert(new(testing.T))

		_ = jq.Enqueue(ctx, simpleJob{})

		pass := jassert.Queued(simpleJob{}, 1337)
		assert.False(t, pass)
	})
}

func TestInMemoryAssertions_QueuedTotal(t *testing.T) {
	t.Parallel()

	jq := jobs.NewTestingJobs()
	jassert := jq.Assert(new(testing.T))

	pass := jassert.QueuedTotal(0, "empty queue")
	assert.True(t, pass)

	_ = jq.Enqueue(ctx, simpleJob{})

	pass = jassert.QueuedTotal(0, "not empty queue - fails")
	assert.False(t, pass)
	pass = jassert.QueuedTotal(1, "not empty queue - succeeds")
	assert.True(t, pass)
}
