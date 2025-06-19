package alog_test

import (
	"errors"
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

		attr := alog.FromContext(ctx)
		assert.Len(t, attr, 1)
	})

	t.Run("add additional attribute", func(t *testing.T) {
		t.Parallel()

		ctx := alog.AddAttr(ctx, slog.String("initial", "attr")) //nolint:govet // shadow ctx to not overwrite it for other tests

		ctx = alog.AddAttr(ctx, slog.String("some", "attr"))

		attr := alog.FromContext(ctx)
		assert.Len(t, attr, 2)
	})
}

func TestAddAttrs(t *testing.T) {
	t.Parallel()

	t.Run("add first attributes", func(t *testing.T) {
		t.Parallel()

		ctx := alog.AddAttrs(ctx, slog.String("some", "attr"), slog.String("other", "attr")) //nolint:govet // shadow ctx to not overwrite it for other tests

		attr := alog.FromContext(ctx)
		assert.Len(t, attr, 2)
	})

	t.Run("add additional attributes ", func(t *testing.T) {
		t.Parallel()

		ctx := alog.AddAttr(ctx, slog.String("initial", "attr")) //nolint:govet // shadow ctx to not overwrite it for other tests

		ctx = alog.AddAttrs(ctx, slog.String("some", "attr"), slog.String("other", "attr"))

		attr := alog.FromContext(ctx)
		assert.Len(t, attr, 3)
	})
}

func TestClearAttrs(t *testing.T) {
	t.Parallel()

	ctx := alog.AddAttr(ctx, slog.String("some", "attr")) //nolint:govet // shadow ctx to not overwrite it for other tests

	ctx = alog.ClearAttrs(ctx)

	attr := alog.FromContext(ctx)
	assert.Empty(t, attr)
}

func TestFromContext(t *testing.T) {
	t.Parallel()

	t.Run("ensure empty ctx has no attr", func(t *testing.T) {
		t.Parallel()

		attrs := alog.FromContext(ctx)
		assert.NotNil(t, attrs)
		assert.Empty(t, attrs)
	})
}

func TestError(t *testing.T) {
	t.Parallel()

	got := alog.Error(errors.New("my-error")) //nolint:err113
	assert.Equal(t, "err", got.Key)
	assert.Equal(t, "my-error", got.Value.String())
	assert.Equal(t, "err=my-error", got.String())
}
