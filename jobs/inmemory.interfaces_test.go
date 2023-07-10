package jobs_test

import (
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

	jq := jobs.NewInMemoryJobs()
	jassert := jq.Assert(t)
	jassert.Empty()

	_ = jq.Enqueue(ctx, simpleJob{})
	jassert.NotEmpty()
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
