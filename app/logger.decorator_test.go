package app_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/app"
)

//nolint:dupl // decorators are basically identical but need a different signature and log output
func TestRequestLoggingDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful request", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := alog.NewTest(buf)
		handler := app.NewLoggedRequest(logger, app.TestSuccessRequestHandler[request, response]())

		_, err := handler.H(context.Background(), request{})
		assert.NoError(t, err)

		assert.Contains(t, buf.String(), `msg="executing request"`)
		assert.Contains(t, buf.String(), `command=app_test.request`)
		assert.Contains(t, buf.String(), `msg="request executed successfully"`)
	})

	t.Run("failed request", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := alog.NewTest(buf)
		handler := app.NewLoggedRequest(logger, app.TestFailureRequestHandler[request, response]())

		_, err := handler.H(context.Background(), request{})
		assert.Error(t, err)

		assert.Contains(t, buf.String(), `msg="executing request"`)
		assert.Contains(t, buf.String(), `command=app_test.request`)
		assert.Contains(t, buf.String(), `msg="failed to execute request"`)
		assert.Contains(t, buf.String(), `error="usecase failed"`)
	})
}

func TestCommandLoggingDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful command", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := alog.NewTest(buf)
		handler := app.NewLoggedCommand(logger, app.TestSuccessCommandHandler[command]())

		err := handler.H(context.Background(), command{})
		assert.NoError(t, err)

		assert.Contains(t, buf.String(), `msg="executing command"`)
		assert.Contains(t, buf.String(), `command=app_test.command`)
		assert.Contains(t, buf.String(), `msg="command executed successfully"`)
	})

	t.Run("failed command", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := alog.NewTest(buf)
		handler := app.NewLoggedCommand(logger, app.TestFailureCommandHandler[command]())

		err := handler.H(context.Background(), command{})
		assert.Error(t, err)

		assert.Contains(t, buf.String(), `msg="executing command"`)
		assert.Contains(t, buf.String(), `command=app_test.command`)
		assert.Contains(t, buf.String(), `msg="failed to execute command"`)
		assert.Contains(t, buf.String(), `error="usecase failed"`)
	})
}

//nolint:dupl // decorators are basically identical but need a different signature and log output
func TestQueryLoggingDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful query", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := alog.NewTest(buf)
		handler := app.NewLoggedQuery(logger, app.TestSuccessQueryHandler[query, response]())

		_, err := handler.H(context.Background(), query{})
		assert.NoError(t, err)

		assert.Contains(t, buf.String(), `msg="executing query"`)
		assert.Contains(t, buf.String(), `command=app_test.query`)
		assert.Contains(t, buf.String(), `msg="query executed successfully"`)
	})

	t.Run("failed query", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := alog.NewTest(buf)
		handler := app.NewLoggedQuery(logger, app.TestFailureQueryHandler[query, response]())

		_, err := handler.H(context.Background(), query{})
		assert.Error(t, err)

		assert.Contains(t, buf.String(), `msg="executing query"`)
		assert.Contains(t, buf.String(), `command=app_test.query`)
		assert.Contains(t, buf.String(), `msg="failed to execute query"`)
		assert.Contains(t, buf.String(), `error="usecase failed"`)
	})
}

func TestJobLoggingDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful job", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := alog.NewTest(buf)
		handler := app.NewLoggedJob(logger, app.TestSuccessJobHandler[job]())

		err := handler.H(context.Background(), job{})
		assert.NoError(t, err)

		assert.Contains(t, buf.String(), `msg="executing job"`)
		assert.Contains(t, buf.String(), `command=app_test.job`)
		assert.Contains(t, buf.String(), `msg="job executed successfully"`)
	})

	t.Run("failed job", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := alog.NewTest(buf)
		handler := app.NewLoggedJob(logger, app.TestFailureJobHandler[job]())

		err := handler.H(context.Background(), job{})
		assert.Error(t, err)

		assert.Contains(t, buf.String(), `msg="executing job"`)
		assert.Contains(t, buf.String(), `command=app_test.job`)
		assert.Contains(t, buf.String(), `msg="failed to execute job"`)
		assert.Contains(t, buf.String(), `error="usecase failed"`)
	})
}

type (
	request  struct{}
	response struct{}
	command  struct{}
	query    struct{}
	job      struct{}
)
