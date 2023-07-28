package mw_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/mw"
)

func TestLogged(t *testing.T) {
	t.Parallel()

	t.Run("successful command", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := alog.NewTest(buf)

		cmd := mw.Logged(logger, func(context.Context, exampleCommand) (string, error) {
			return "", nil
		})

		_, _ = cmd(context.Background(), exampleCommand{})

		assert.Contains(t, buf.String(), `msg="executing command"`)
		assert.Contains(t, buf.String(), `command=mw_test.exampleCommand`)
		assert.Contains(t, buf.String(), `msg="command executed successfully"`)
	})

	t.Run("failed command", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := alog.NewTest(buf)

		cmd := mw.Logged(logger, func(context.Context, exampleCommand) (string, error) {
			return "", errUseCaseFails
		})

		_, _ = cmd(context.Background(), exampleCommand{})

		assert.Contains(t, buf.String(), `msg="executing command"`)
		assert.Contains(t, buf.String(), `command=mw_test.exampleCommand`)
		assert.Contains(t, buf.String(), `msg="failed to execute command"`)
		assert.Contains(t, buf.String(), `error=some-error`)
	})
}

func TestLoggedU(t *testing.T) {
	t.Parallel()

	t.Run("successful command", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := alog.NewTest(buf)

		cmd := mw.LoggedU(logger, func(context.Context, exampleCommand) error {
			return nil
		})

		_ = cmd(context.Background(), exampleCommand{})

		assert.Contains(t, buf.String(), `msg="executing command"`)
		assert.Contains(t, buf.String(), `command=mw_test.exampleCommand`)
		assert.Contains(t, buf.String(), `msg="command executed successfully"`)
	})

	t.Run("failed command", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := alog.NewTest(buf)

		cmd := mw.LoggedU(logger, func(context.Context, exampleCommand) error {
			return errUseCaseFails
		})

		_ = cmd(context.Background(), exampleCommand{})

		assert.Contains(t, buf.String(), `msg="executing command"`)
		assert.Contains(t, buf.String(), `command=mw_test.exampleCommand`)
		assert.Contains(t, buf.String(), `msg="failed to execute command"`)
		assert.Contains(t, buf.String(), `error=some-error`)
	})
}
