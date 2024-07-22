package app_test

import (
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

		logger := alog.Test(t)
		handler := app.NewLoggedRequest(logger, app.TestSuccessRequestHandler[request, response]())

		_, err := handler.H(context.Background(), request{})
		assert.NoError(t, err)

		logger.Contains(`msg="executing request"`)
		logger.Contains(`command=app_test.request`)
		logger.Contains(`msg="request executed successfully"`)
	})

	t.Run("failed request", func(t *testing.T) {
		t.Parallel()

		logger := alog.Test(t)
		handler := app.NewLoggedRequest(logger, app.TestFailureRequestHandler[request, response]())

		_, err := handler.H(context.Background(), request{})
		assert.Error(t, err)

		logger.Contains(`msg="executing request"`)
		logger.Contains(`command=app_test.request`)
		logger.Contains(`msg="failed to execute request"`)
		logger.Contains(`error="usecase failed"`)
	})
}

func TestCommandLoggingDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful command", func(t *testing.T) {
		t.Parallel()

		logger := alog.Test(t)
		handler := app.NewLoggedCommand(logger, app.TestSuccessCommandHandler[command]())

		err := handler.H(context.Background(), command{})
		assert.NoError(t, err)

		logger.Contains(`msg="executing command"`)
		logger.Contains(`command=app_test.command`)
		logger.Contains(`msg="command executed successfully"`)
	})

	t.Run("failed command", func(t *testing.T) {
		t.Parallel()

		logger := alog.Test(t)
		handler := app.NewLoggedCommand(logger, app.TestFailureCommandHandler[command]())

		err := handler.H(context.Background(), command{})
		assert.Error(t, err)

		logger.Contains(`msg="executing command"`)
		logger.Contains(`command=app_test.command`)
		logger.Contains(`msg="failed to execute command"`)
		logger.Contains(`error="usecase failed"`)
	})
}

//nolint:dupl // decorators are basically identical but need a different signature and log output
func TestQueryLoggingDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful query", func(t *testing.T) {
		t.Parallel()

		logger := alog.Test(t)
		handler := app.NewLoggedQuery(logger, app.TestSuccessQueryHandler[query, response]())

		_, err := handler.H(context.Background(), query{})
		assert.NoError(t, err)

		logger.Contains(`msg="executing query"`)
		logger.Contains(`command=app_test.query`)
		logger.Contains(`msg="query executed successfully"`)
	})

	t.Run("failed query", func(t *testing.T) {
		t.Parallel()

		logger := alog.Test(t)
		handler := app.NewLoggedQuery(logger, app.TestFailureQueryHandler[query, response]())

		_, err := handler.H(context.Background(), query{})
		assert.Error(t, err)

		logger.Contains(`msg="executing query"`)
		logger.Contains(`command=app_test.query`)
		logger.Contains(`msg="failed to execute query"`)
		logger.Contains(`error="usecase failed"`)
	})
}

func TestJobLoggingDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful job", func(t *testing.T) {
		t.Parallel()

		logger := alog.Test(t)
		handler := app.NewLoggedJob(logger, app.TestSuccessJobHandler[job]())

		err := handler.H(context.Background(), job{})
		assert.NoError(t, err)

		logger.Contains(`msg="executing job"`)
		logger.Contains(`command=app_test.job`)
		logger.Contains(`msg="job executed successfully"`)
	})

	t.Run("failed job", func(t *testing.T) {
		t.Parallel()

		logger := alog.Test(t)
		handler := app.NewLoggedJob(logger, app.TestFailureJobHandler[job]())

		err := handler.H(context.Background(), job{})
		assert.Error(t, err)

		logger.Contains(`msg="executing job"`)
		logger.Contains(`command=app_test.job`)
		logger.Contains(`msg="failed to execute job"`)
		logger.Contains(`error="usecase failed"`)
	})
}

type (
	request  struct{}
	response struct{}
	command  struct{}
	query    struct{}
	job      struct{}
)
