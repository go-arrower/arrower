package alog_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/alog"
)

func TestTest(t *testing.T) {
	t.Parallel()

	t.Run("test logger", func(t *testing.T) {
		t.Parallel()

		logger := alog.Test(t)
		assert.NotNil(t, logger)
	})

	t.Run("nil does panic", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			alog.Test(nil)
		})
	})

	t.Run("default level is debug", func(t *testing.T) {
		t.Parallel()

		logger := alog.Test(t)

		logger.Debug("debug msg")

		assert.Contains(t, logger.String(), "debug msg")
	})

	t.Run("with group", func(t *testing.T) {
		t.Parallel()

		logger := alog.Test(t)

		logger.DebugContext(context.Background(), "msg 0")

		simulateComponentWorkingWithLogger(logger)

		logger.Contains("msg 0")
		logger.Contains("GROUP.some=key")
	})
}

func simulateComponentWorkingWithLogger(logger alog.Logger) {
	logger = logger.WithGroup("GROUP")
	logger.DebugContext(context.Background(), "msg group", "some", "key")
}

func TestTestLogger_Lines(t *testing.T) {
	t.Parallel()

	logger := alog.Test(t)
	logger.DebugContext(ctx, "line 0")
	logger.DebugContext(ctx, "line 1")

	assert.Len(t, logger.Lines(), 2)
	assert.Contains(t, logger.Lines()[0], `level=DEBUG msg="line 0"`)
	assert.Contains(t, logger.Lines()[1], `level=DEBUG msg="line 1"`)
}

func TestTestLogger_Empty(t *testing.T) {
	t.Parallel()

	t.Run("empty logger", func(t *testing.T) {
		t.Parallel()

		logger := alog.Test(new(testing.T))

		pass := logger.Empty()
		assert.True(t, pass)
	})

	t.Run("not empty logger", func(t *testing.T) {
		t.Parallel()

		logger := alog.Test(new(testing.T))

		logger.Debug("debug msg")

		pass := logger.Empty()
		assert.False(t, pass)
	})
}

func TestTestLogger_NotEmpty(t *testing.T) {
	t.Parallel()

	t.Run("empty logger", func(t *testing.T) {
		t.Parallel()

		logger := alog.Test(new(testing.T))

		pass := logger.NotEmpty()
		assert.False(t, pass)
	})

	t.Run("not empty logger", func(t *testing.T) {
		t.Parallel()

		logger := alog.Test(new(testing.T))

		logger.Debug("debug msg")

		pass := logger.NotEmpty()
		assert.True(t, pass)
	})
}

func TestTestLogger_Contains(t *testing.T) {
	t.Parallel()

	t.Run("contains", func(t *testing.T) {
		t.Parallel()

		logger := alog.Test(new(testing.T))

		logger.Debug("debug msg")

		pass := logger.Contains("debug msg")
		assert.True(t, pass)
	})

	t.Run("not contains", func(t *testing.T) {
		t.Parallel()

		logger := alog.Test(new(testing.T))

		logger.Debug("debug msg")

		pass := logger.Contains("other msg")
		assert.False(t, pass)
	})
}

func TestTestLogger_NotContains(t *testing.T) {
	t.Parallel()

	t.Run("not contains", func(t *testing.T) {
		t.Parallel()

		logger := alog.Test(new(testing.T))

		logger.Debug("debug msg")

		pass := logger.NotContains("other msg")
		assert.True(t, pass)
	})

	t.Run("contains", func(t *testing.T) {
		t.Parallel()

		logger := alog.Test(new(testing.T))

		logger.Debug("debug msg")

		pass := logger.NotContains("debug msg")
		assert.False(t, pass)
	})
}

func TestTestLogger_Total(t *testing.T) {
	t.Parallel()

	t.Run("empty logger", func(t *testing.T) {
		t.Parallel()

		logger := alog.Test(new(testing.T))

		pass := logger.Total(0, "empty queue -> passes")
		assert.True(t, pass)

		pass = logger.Total(1, "empty queue -> fails")
		assert.False(t, pass)
	})

	t.Run("not empty logger", func(t *testing.T) {
		t.Parallel()

		logger := alog.Test(new(testing.T))

		logger.Debug("debug msg")

		pass := logger.Total(1, "not empty queue -> passes")
		assert.True(t, pass)

		pass = logger.Total(0, "not empty queue -> fails")
		assert.False(t, pass)
	})
}

func TestTestLogger_Concurrent(t *testing.T) {
	t.Parallel()

	logger := alog.Test(t)

	t.Run("concurrent", func(t *testing.T) {
		t.Parallel()

		logger.DebugContext(ctx, "debug msg")
	})

	t.Run("concurrent", func(t *testing.T) {
		t.Parallel()

		logger.NotContains("info")
	})
}
