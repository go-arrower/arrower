package jobs_test

import (
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/jobs"
)

func TestTest(t *testing.T) {
	t.Parallel()

	t.Run("test queue", func(t *testing.T) {
		t.Parallel()

		jq := jobs.Test(t)
		assert.NotNil(t, jq)
	})

	t.Run("nil does panic", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			jobs.Test(nil)
		})
	})
}

func TestTestQueue_Jobs(t *testing.T) {
	t.Parallel()

	jq := jobs.Test(t)
	_ = jq.Enqueue(ctx, simpleJob{})
	_ = jq.Enqueue(ctx, simpleJob{})

	assert.Len(t, jq.Jobs(), 2)
}

func TestTestQueue_GetFirst(t *testing.T) {
	t.Parallel()

	t.Run("no job as queue is empty", func(t *testing.T) {
		t.Parallel()

		jq := jobs.Test(t)

		assert.Nil(t, jq.GetFirst())
	})

	t.Run("get first job", func(t *testing.T) {
		t.Parallel()

		jq := jobs.Test(t)
		_ = jq.Enqueue(ctx, []jobs.Job{jobWithArgs{Name: argName}, jobWithArgs{Name: gofakeit.Name()}})

		j := jq.GetFirst().(jobWithArgs)
		assert.Equal(t, argName, j.Name)
	})
}

func TestTestQueue_Get(t *testing.T) {
	t.Parallel()

	t.Run("no job as queue is empty", func(t *testing.T) {
		t.Parallel()

		jq := jobs.Test(t)

		assert.Nil(t, jq.Get(0))
		assert.Nil(t, jq.Get(1))
	})

	t.Run("get second job", func(t *testing.T) {
		t.Parallel()

		jq := jobs.Test(t)
		_ = jq.Enqueue(ctx, []jobs.Job{jobWithArgs{Name: argName}, jobWithArgs{Name: "otherName"}})

		j := jq.Get(2).(jobWithArgs)
		assert.Equal(t, "otherName", j.Name)
	})
}

func TestTestQueue_GetFirstOf(t *testing.T) {
	t.Parallel()

	t.Run("no job as queue is empty", func(t *testing.T) {
		t.Parallel()

		jq := jobs.Test(t)

		assert.Nil(t, jq.GetFirstOf(nil))
		assert.Nil(t, jq.GetFirstOf(simpleJob{}))
	})

	t.Run("get first job", func(t *testing.T) {
		t.Parallel()

		jq := jobs.Test(t)
		_ = jq.Enqueue(ctx, []jobs.Job{simpleJob{}, jobWithArgs{Name: argName}})

		j := jq.GetFirstOf(jobWithArgs{}).(jobWithArgs)
		assert.Equal(t, argName, j.Name)
	})
}

func TestTestQueue_GetOf(t *testing.T) {
	t.Parallel()

	t.Run("no job as queue is empty", func(t *testing.T) {
		t.Parallel()

		jq := jobs.Test(t)

		assert.Nil(t, jq.GetOf(simpleJob{}, 0))
		assert.Nil(t, jq.GetOf(simpleJob{}, 1))
	})

	t.Run("get second job of type", func(t *testing.T) {
		t.Parallel()

		jq := jobs.Test(t)
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

func TestTestAssertions_Empty(t *testing.T) {
	t.Parallel()

	t.Run("empty queue", func(t *testing.T) {
		t.Parallel()

		jq := jobs.Test(new(testing.T))

		pass := jq.Empty()
		assert.True(t, pass)
	})

	t.Run("not empty queue", func(t *testing.T) {
		t.Parallel()

		jq := jobs.Test(new(testing.T))

		_ = jq.Enqueue(ctx, simpleJob{})

		pass := jq.Empty()
		assert.False(t, pass)
	})
}

func TestTestAssertions_NotEmpty(t *testing.T) {
	t.Parallel()

	t.Run("empty queue", func(t *testing.T) {
		t.Parallel()

		jq := jobs.Test(new(testing.T))

		pass := jq.NotEmpty()
		assert.False(t, pass)
	})

	t.Run("not empty queue", func(t *testing.T) {
		t.Parallel()

		jq := jobs.Test(new(testing.T))

		_ = jq.Enqueue(ctx, simpleJob{})

		pass := jq.NotEmpty()
		assert.True(t, pass)
	})
}

func TestTestAssertions_Total(t *testing.T) {
	t.Parallel()

	t.Run("empty queue", func(t *testing.T) {
		t.Parallel()

		jq := jobs.Test(new(testing.T))

		pass := jq.Total(0, "empty queue -> passes")
		assert.True(t, pass)

		pass = jq.Total(1, "empty queue -> fails")
		assert.False(t, pass)
	})

	t.Run("not empty queue", func(t *testing.T) {
		t.Parallel()

		jq := jobs.Test(new(testing.T))

		_ = jq.Enqueue(ctx, simpleJob{})

		pass := jq.Total(1, "not empty queue -> passes")
		assert.True(t, pass)

		pass = jq.Total(0, "not empty queue -> fails")
		assert.False(t, pass)
	})
}
