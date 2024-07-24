package alog_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/alog"
)

func TestAddAttr(t *testing.T) {
	t.Parallel()

	t.Run("add first attribute", func(t *testing.T) {
		t.Parallel()

		ctx := alog.AddAttr(ctx, slog.String("some", "attr")) //nolint:govet // shadow ctx to not overwrite it for other tests

		attr, ok := alog.FromContext(ctx)
		assert.True(t, ok)
		assert.Len(t, attr, 1)
	})

	t.Run("add additional attribute", func(t *testing.T) {
		t.Parallel()

		ctx := alog.AddAttr(ctx, slog.String("initial", "attr")) //nolint:govet // shadow ctx to not overwrite it for other tests

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

		ctx := alog.AddAttrs(ctx, slog.String("some", "attr"), slog.String("other", "attr")) //nolint:govet // shadow ctx to not overwrite it for other tests

		attr, ok := alog.FromContext(ctx)
		assert.True(t, ok)
		assert.Len(t, attr, 2)
	})

	t.Run("add additional attributes ", func(t *testing.T) {
		t.Parallel()

		ctx := alog.AddAttr(ctx, slog.String("initial", "attr")) //nolint:govet // shadow ctx to not overwrite it for other tests

		ctx = alog.AddAttrs(ctx, slog.String("some", "attr"), slog.String("other", "attr"))

		attr, ok := alog.FromContext(ctx)
		assert.True(t, ok)
		assert.Len(t, attr, 3)
	})
}

func TestResetAttrs(t *testing.T) {
	t.Parallel()

	ctx := alog.AddAttr(ctx, slog.String("some", "attr")) //nolint:govet // shadow ctx to not overwrite it for other tests

	ctx = alog.ResetAttrs(ctx)

	attr, ok := alog.FromContext(ctx)
	assert.False(t, ok)
	assert.Empty(t, attr)
}

func TestFromContext(t *testing.T) {
	t.Parallel()

	t.Run("ensure empty ctx has no attr", func(t *testing.T) {
		t.Parallel()

		attr, ok := alog.FromContext(ctx)
		assert.False(t, ok)
		assert.Empty(t, attr)
	})
}
