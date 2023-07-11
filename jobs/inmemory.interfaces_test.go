package jobs_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/jobs"
)

func TestNewInMemoryJobs(t *testing.T) {
	t.Parallel()

	t.Run("initialise a new InMemoryHandler", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewInMemoryJobs()
		assert.NotEmpty(t, jq)
	})
}

func TestInMemoryHandler_Enqueue(t *testing.T) {
	t.Parallel()

	t.Run("invalid job", func(t *testing.T) {
		jq := jobs.NewInMemoryJobs()

		err := jq.Enqueue(ctx, nil)
		assert.Error(t, err)
		assert.Equal(t, jobs.ErrInvalidJobType, err)
	})

	t.Run("simple job", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewInMemoryJobs()
		jassert := jq.Assert(t)
		jassert.Empty()

		_ = jq.Enqueue(ctx, simpleJob{})
		jassert.NotEmpty()
		jassert.Queued(simpleJob{}, 1)
	})

	t.Run("slice of same jobs", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewInMemoryJobs()
		jassert := jq.Assert(t)

		_ = jq.Enqueue(ctx, []jobWithArgs{{Name: argName}, {Name: argName}})
		jassert.Queued(jobWithArgs{}, 2)

		_ = jq.Enqueue(ctx, []simpleJob{{}, {}})
		jassert.Queued(jobWithSameNameAsSimpleJob{}, 2)
	})

	t.Run("slice of different job types", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewInMemoryJobs()
		jassert := jq.Assert(t)

		_ = jq.Enqueue(ctx, []any{jobWithArgs{Name: argName}, jobWithJobType{Name: argName}})
		jassert.Queued(jobWithArgs{}, 1)
		jassert.Queued(jobWithJobType{}, 1)
	})

	t.Run("enqueue safely concurrently", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup
		totalProducers := 1000

		jq := jobs.NewInMemoryJobs()

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

func TestInMemoryHandler_Reset(t *testing.T) {
	t.Parallel()

	t.Run("reset empty queue", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewInMemoryJobs()

		jq.Reset()
		jq.Assert(t).QueuedTotal(0)
	})

	t.Run("reset full queue", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewInMemoryJobs()
		jassert := jq.Assert(t)

		_ = jq.Enqueue(ctx, simpleJob{})
		_ = jq.Enqueue(ctx, simpleJob{})
		_ = jq.Enqueue(ctx, jobWithArgs{})
		jassert.QueuedTotal(3)

		jq.Reset()
		jassert.QueuedTotal(0)
	})
}

func TestInMemoryHandler_Assert(t *testing.T) {
	t.Parallel()

	jq := jobs.NewInMemoryJobs()

	jassert := jq.Assert(t)
	assert.NotEmpty(t, jassert)
}

func TestInMemoryAssertions_Empty(t *testing.T) {
	t.Parallel()

	t.Run("empty queue", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewInMemoryJobs()
		jassert := jq.Assert(new(testing.T))

		pass := jassert.Empty()
		assert.True(t, pass)
	})

	t.Run("not empty queue", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewInMemoryJobs()
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

		jq := jobs.NewInMemoryJobs()
		jassert := jq.Assert(new(testing.T))

		pass := jassert.NotEmpty()
		assert.False(t, pass)
	})

	t.Run("not empty queue", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewInMemoryJobs()
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

		jq := jobs.NewInMemoryJobs()
		jassert := jq.Assert(new(testing.T))

		pass := jassert.Queued(simpleJob{}, 0)
		assert.True(t, pass)
	})

	t.Run("job of type queued", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewInMemoryJobs()
		jassert := jq.Assert(new(testing.T))

		_ = jq.Enqueue(ctx, jobWithArgs{})
		_ = jq.Enqueue(ctx, simpleJob{})
		_ = jq.Enqueue(ctx, simpleJob{})
		pass := jassert.Queued(simpleJob{}, 2)
		assert.True(t, pass)
	})

	t.Run("different amount of jobs queued", func(t *testing.T) {
		t.Parallel()

		jq := jobs.NewInMemoryJobs()
		jassert := jq.Assert(new(testing.T))

		_ = jq.Enqueue(ctx, simpleJob{})
		pass := jassert.Queued(simpleJob{}, 1337)
		assert.False(t, pass)
	})
}

func TestInMemoryAssertions_QueuedTotal(t *testing.T) {
	t.Parallel()

	jq := jobs.NewInMemoryJobs()
	jassert := jq.Assert(new(testing.T))

	pass := jassert.QueuedTotal(0, "empty queue")
	assert.True(t, pass)

	_ = jq.Enqueue(ctx, simpleJob{})
	pass = jassert.QueuedTotal(0, "not empty queue - fails")
	assert.False(t, pass)
	pass = jassert.QueuedTotal(1, "not empty queue - succeeds")
	assert.True(t, pass)
}
