package arrower_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"

	"github.com/go-arrower/arrower"
)

const (
	userKey       = "user_id"
	userName      = "1337"
	userFormatted = "user_id=1337"
)

func TestNewDevelopment(t *testing.T) {
	t.Parallel()

	t.Run("log hello", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		h := arrower.NewFilteredLogger(buf)
		h.Logger.Info("hello logger")

		assert.Contains(t, buf.String(), "hello logger")
	})

	t.Run("level debug by default", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		h := arrower.NewFilteredLogger(buf)

		h.Logger.Log(context.Background(), arrower.LevelDebug, "arrower debug")
		h.Logger.Log(context.Background(), arrower.LevelTrace, "arrower trance")
		assert.Empty(t, buf.String())

		h.Logger.Debug("application debug msg")
		assert.Contains(t, buf.String(), `msg="application debug msg"`)
	})
}

func TestSetLevel(t *testing.T) {
	t.Parallel()

	t.Run("set level arrower:debug", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		h := arrower.NewFilteredLogger(buf)

		h.SetLogLevel(arrower.LevelDebug)
		h.Logger.Log(context.Background(), arrower.LevelDebug, "arrower debug")
		assert.Contains(t, buf.String(), `msg="arrower debug"`)
		assert.Contains(t, buf.String(), `level=ARROWER:DEBUG`)
	})

	t.Run("set level arrower:trance", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		h := arrower.NewFilteredLogger(buf)

		h.SetLogLevel(arrower.LevelTrace)
		h.Logger.Log(context.Background(), arrower.LevelTrace, "arrower trace")
		assert.Contains(t, buf.String(), `msg="arrower trace"`)
		assert.Contains(t, buf.String(), `level=ARROWER:TRACE`)
	})
}

func TestFilteredLogger_Handle(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	handler := arrower.NewFilteredLogger(buf)
	logger := handler.Logger

	buf.Reset()
	logger.Debug("message")
	assert.Contains(t, buf.String(), "message")

	buf.Reset()
	logger.Debug("", userKey, "default")
	assert.Contains(t, buf.String(), "default", "not observed user")

	handler.SetLogLevel(slog.LevelError)

	buf.Reset()
	logger.Debug("", userKey, "default")
	assert.NotContains(t, buf.String(), "default", "not observed user")

	buf.Reset()
	logger.Debug("", userKey, userName)
	assert.NotContains(t, buf.String(), userFormatted, "not observed yet")

	buf.Reset()
	handler.SetUserFilter("1337")
	logger.Debug("", userKey, userName)
	assert.Contains(t, buf.String(), userFormatted, "observed user")

	buf.Reset()
	handler.RemoveUserFilter(userName)
	logger.Debug("", userKey, userName)
	assert.NotContains(t, buf.String(), userFormatted, "removed user")

	buf.Reset()
	handler.SetLogLevel(slog.LevelDebug)

	logger = logger.WithGroup("arrower")

	handler.SetUserFilter(userName)
	logger.Debug("", userKey, userName)
	assert.Contains(t, buf.String(), userFormatted, "observed user with group")

	buf.Reset()
	handler.SetLogLevel(slog.LevelDebug)

	logger = logger.With("app", "logger")
	logger.Debug("", userKey, userName)
	assert.Contains(t, buf.String(), "app=logger")
}

func TestFilteredLogger_Enabled(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	handler := arrower.NewFilteredLogger(buf)
	logger := handler.Logger

	handler.SetLogLevel(slog.LevelDebug)
	logger.Debug("message", userKey, userName)
	assert.Contains(t, buf.String(), userFormatted)

	buf.Reset()
	handler.SetLogLevel(slog.LevelError)
	logger.Debug("message", userKey, userName)
	assert.NotContains(t, buf.String(), userFormatted)

	handler.SetUserFilter("1337")
	logger.Debug("message", userKey, userName)
	assert.Contains(t, buf.String(), userFormatted)
}
