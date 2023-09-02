package alog_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"

	"github.com/go-arrower/arrower/alog"
)

func TestAddAttr(t *testing.T) {
	t.Parallel()

	t.Run("add first attribute", func(t *testing.T) {
		t.Parallel()

		ctx := alog.AddAttr(context.Background(), slog.String("some", "attr"))

		attr, ok := alog.FromContext(ctx)
		assert.True(t, ok)
		assert.Len(t, attr, 1)
	})

	t.Run("add additional attribute", func(t *testing.T) {
		t.Parallel()

		ctx := alog.AddAttr(context.Background(), slog.String("initial", "attr"))

		ctx = alog.AddAttr(ctx, slog.String("some", "attr"))

		attr, ok := alog.FromContext(ctx)
		assert.True(t, ok)
		assert.Len(t, attr, 2)
	})
}

func TestAddAttrs(t *testing.T) {
	t.Parallel()

	t.Run("add first attributes", func(t *testing.T) {
		t.Parallel()

		ctx := alog.AddAttrs(context.Background(), slog.String("some", "attr"), slog.String("other", "attr"))

		attr, ok := alog.FromContext(ctx)
		assert.True(t, ok)
		assert.Len(t, attr, 2)
	})

	t.Run("add additional attributes ", func(t *testing.T) {
		t.Parallel()

		ctx := alog.AddAttr(context.Background(), slog.String("initial", "attr"))

		ctx = alog.AddAttrs(ctx, slog.String("some", "attr"), slog.String("other", "attr"))

		attr, ok := alog.FromContext(ctx)
		assert.True(t, ok)
		assert.Len(t, attr, 3)
	})
}

func TestResetAttrs(t *testing.T) {
	t.Parallel()

	ctx := alog.AddAttr(context.Background(), slog.String("some", "attr"))

	ctx = alog.ResetAttrs(ctx)

	attr, ok := alog.FromContext(ctx)
	assert.False(t, ok)
	assert.Empty(t, attr)
}

func TestFromContext(t *testing.T) {
	t.Parallel()

	t.Run("ensure empty ctx has no attr", func(t *testing.T) {
		t.Parallel()

		attr, ok := alog.FromContext(context.Background())
		assert.False(t, ok)
		assert.Empty(t, attr)
	})
}
