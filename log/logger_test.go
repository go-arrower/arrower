package log_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/log"
)

func TestNewDevelopment(t *testing.T) {
	t.Parallel()

	t.Run("log hello", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := log.NewDevelopment(buf)
		logger.Info("hello logger")

		assert.Contains(t, buf.String(), "hello logger")
	})

	t.Run("level debug by default", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		buf := &bytes.Buffer{}
		logger := log.NewDevelopment(buf)

		logger.Log(ctx, log.ArrowerLevelDebug, "arrower debug")
		logger.Log(ctx, log.ArrowerLevelTrace, "arrower trance")
		assert.Empty(t, buf.String())

		logger.Debug("application debug msg")
		assert.Contains(t, buf.String(), `msg="application debug msg"`)
	})
}

func TestSetLevel(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("set level arrower:debug", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := log.NewDevelopment(buf)

		log.SetLevel(log.ArrowerLevelDebug)
		logger.Log(ctx, log.ArrowerLevelDebug, "arrower debug")
		assert.Contains(t, buf.String(), `msg="arrower debug"`)
		assert.Contains(t, buf.String(), `level=Arrower:DEBUG`)
	})

	t.Run("set level arrower:trance", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := log.NewDevelopment(buf)

		log.SetLevel(log.ArrowerLevelTrace)
		logger.Log(ctx, log.ArrowerLevelTrace, "arrower trace")
		assert.Contains(t, buf.String(), `msg="arrower trace"`)
		assert.Contains(t, buf.String(), `level=Arrower:TRACE`)
	})
}
