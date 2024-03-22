package app_test

import (
	"bytes"
	"context"
	"errors"
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
		handler := app.NewLoggedRequest[request, response](logger, &requestSuccessHandler{})

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
		handler := app.NewLoggedRequest[request, response](logger, &requestFailureHandler{})

		_, err := handler.H(context.Background(), request{})
		assert.Error(t, err)

		assert.Contains(t, buf.String(), `msg="executing request"`)
		assert.Contains(t, buf.String(), `command=app_test.request`)
		assert.Contains(t, buf.String(), `msg="failed to execute request"`)
		assert.Contains(t, buf.String(), `error=some-error`)
	})
}

func TestCommandLoggingDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful command", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := alog.NewTest(buf)
		handler := app.NewLoggedCommand[request](logger, &commandSuccessHandler{})

		err := handler.H(context.Background(), request{})
		assert.NoError(t, err)

		assert.Contains(t, buf.String(), `msg="executing command"`)
		assert.Contains(t, buf.String(), `command=app_test.request`)
		assert.Contains(t, buf.String(), `msg="command executed successfully"`)
	})

	t.Run("failed command", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := alog.NewTest(buf)
		handler := app.NewLoggedCommand[request](logger, &commandFailureHandler{})

		err := handler.H(context.Background(), request{})
		assert.Error(t, err)

		assert.Contains(t, buf.String(), `msg="executing command"`)
		assert.Contains(t, buf.String(), `command=app_test.request`)
		assert.Contains(t, buf.String(), `msg="failed to execute command"`)
		assert.Contains(t, buf.String(), `error=some-error`)
	})
}

//nolint:dupl // decorators are basically identical but need a different signature and log output
func TestQueryLoggingDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful query", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := alog.NewTest(buf)
		handler := app.NewLoggedQuery[request, response](logger, &requestSuccessHandler{})

		_, err := handler.H(context.Background(), request{})
		assert.NoError(t, err)

		assert.Contains(t, buf.String(), `msg="executing query"`)
		assert.Contains(t, buf.String(), `command=app_test.request`)
		assert.Contains(t, buf.String(), `msg="query executed successfully"`)
	})

	t.Run("failed query", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := alog.NewTest(buf)
		handler := app.NewLoggedQuery[request, response](logger, &requestFailureHandler{})

		_, err := handler.H(context.Background(), request{})
		assert.Error(t, err)

		assert.Contains(t, buf.String(), `msg="executing query"`)
		assert.Contains(t, buf.String(), `command=app_test.request`)
		assert.Contains(t, buf.String(), `msg="failed to execute query"`)
		assert.Contains(t, buf.String(), `error=some-error`)
	})
}

func TestJobLoggingDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful job", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := alog.NewTest(buf)
		handler := app.NewLoggedJob[request](logger, &commandSuccessHandler{})

		err := handler.H(context.Background(), request{})
		assert.NoError(t, err)

		assert.Contains(t, buf.String(), `msg="executing job"`)
		assert.Contains(t, buf.String(), `command=app_test.request`)
		assert.Contains(t, buf.String(), `msg="job executed successfully"`)
	})

	t.Run("failed job", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		logger := alog.NewTest(buf)
		handler := app.NewLoggedJob[request](logger, &commandFailureHandler{})

		err := handler.H(context.Background(), request{})
		assert.Error(t, err)

		assert.Contains(t, buf.String(), `msg="executing job"`)
		assert.Contains(t, buf.String(), `command=app_test.request`)
		assert.Contains(t, buf.String(), `msg="failed to execute job"`)
		assert.Contains(t, buf.String(), `error=some-error`)
	})
}

var errUseCaseFails = errors.New("some-error")

type (
	request  struct{}
	response struct{}

	requestSuccessHandler struct{}
	requestFailureHandler struct{}

	commandSuccessHandler struct{}
	commandFailureHandler struct{}
)

func (h *requestSuccessHandler) H(_ context.Context, _ request) (response, error) {
	return response{}, nil
}

func (h *requestFailureHandler) H(_ context.Context, _ request) (response, error) {
	return response{}, errUseCaseFails
}

func (h *commandSuccessHandler) H(_ context.Context, _ request) error {
	return nil
}

func (h *commandFailureHandler) H(_ context.Context, _ request) error {
	return errUseCaseFails
}
