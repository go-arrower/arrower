package arrower_test

import (
	"bytes"
	"testing"

	"github.com/go-arrower/arrower"

	"github.com/stretchr/testify/assert"
)

func TestNewDevelopment(t *testing.T) {
	t.Parallel()

	t.Run("log hello", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := arrower.NewDevelopment(buf)
		logger.Info("hello logger")

		assert.Contains(t, buf.String(), "hello logger")
	})

	t.Run("level debug by default", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := arrower.NewDevelopment(buf)

		logger.Log(nil, arrower.LevelDebug, "arrower debug")
		logger.Log(nil, arrower.LevelTrace, "arrower trance")
		assert.Empty(t, buf.String())

		logger.Debug("application debug msg")
		assert.Contains(t, buf.String(), `msg="application debug msg"`)
	})
}

func TestSetLevel(t *testing.T) {
	t.Parallel()

	t.Run("set level arrower:debug", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := arrower.NewDevelopment(buf)

		arrower.SetLogLevel(arrower.LevelDebug)
		logger.Log(nil, arrower.LevelDebug, "arrower debug")
		assert.Contains(t, buf.String(), `msg="arrower debug"`)
		assert.Contains(t, buf.String(), `level=ARROWER:DEBUG`)
	})

	t.Run("set level arrower:trance", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := arrower.NewDevelopment(buf)

		arrower.SetLogLevel(arrower.LevelTrace)
		logger.Log(nil, arrower.LevelTrace, "arrower trace")
		assert.Contains(t, buf.String(), `msg="arrower trace"`)
		assert.Contains(t, buf.String(), `level=ARROWER:TRACE`)
	})
}
