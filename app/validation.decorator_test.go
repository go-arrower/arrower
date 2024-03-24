package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/app"
)

//nolint:dupl // decorators are basically identical but need a different signature
func TestRequestValidatingDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful request", func(t *testing.T) {
		t.Parallel()

		handler := app.NewValidatedRequest[structWithValidationTags, response](validator.New(), &validateSuccessRequestHandler{t: t})

		res, err := handler.H(context.Background(), passingValidationValue)
		assert.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("failed request", func(t *testing.T) {
		t.Parallel()

		handler := app.NewValidatedRequest[structWithValidationTags, response](validator.New(), &validateFailureRequestHandler{t: t})

		res, err := handler.H(context.Background(), structWithValidationTags{})
		validationErrors := validator.ValidationErrors{}
		_ = errors.As(err, &validationErrors)

		assert.Error(t, err)
		assert.Len(t, validationErrors, 2)
		assert.Empty(t, res)
	})

	t.Run("with default validator", func(t *testing.T) {
		t.Parallel()

		handler := app.NewValidatedRequest[structWithValidationTags, response](nil, &validateSuccessRequestHandler{t: t})

		res, err := handler.H(context.Background(), passingValidationValue)
		assert.NoError(t, err)
		assert.Empty(t, res)
	})
}

func TestCommandValidatingDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful command", func(t *testing.T) {
		t.Parallel()

		handler := app.NewValidatedCommand[structWithValidationTags](validator.New(), &validateSuccessCommandHandler{t: t})

		err := handler.H(context.Background(), passingValidationValue)
		assert.NoError(t, err)
	})

	t.Run("failed command", func(t *testing.T) {
		t.Parallel()

		handler := app.NewValidatedCommand[structWithValidationTags](validator.New(), &validateFailureCommandHandler{t: t})

		err := handler.H(context.Background(), structWithValidationTags{})
		validationErrors := validator.ValidationErrors{}
		_ = errors.As(err, &validationErrors)

		assert.Error(t, err)
		assert.Len(t, validationErrors, 2)
	})

	t.Run("with default validator", func(t *testing.T) {
		t.Parallel()

		handler := app.NewValidatedCommand[structWithValidationTags](nil, &validateSuccessCommandHandler{t: t})

		err := handler.H(context.Background(), passingValidationValue)
		assert.NoError(t, err)
	})
}

//nolint:dupl // decorators are basically identical but need a different signature
func TestQueryValidatingDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful query", func(t *testing.T) {
		t.Parallel()

		handler := app.NewValidatedQuery[structWithValidationTags, response](validator.New(), &validateSuccessRequestHandler{t: t}) // reuse request handler to reduce boilerplate

		res, err := handler.H(context.Background(), passingValidationValue)
		assert.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("failed query", func(t *testing.T) {
		t.Parallel()

		handler := app.NewValidatedQuery[structWithValidationTags, response](validator.New(), &validateFailureRequestHandler{t: t}) // reuse request handler to reduce boilerplate

		res, err := handler.H(context.Background(), structWithValidationTags{})
		validationErrors := validator.ValidationErrors{}
		_ = errors.As(err, &validationErrors)

		assert.Error(t, err)
		assert.Len(t, validationErrors, 2)
		assert.Empty(t, res)
	})

	t.Run("with default validator", func(t *testing.T) {
		t.Parallel()

		handler := app.NewValidatedQuery[structWithValidationTags, response](nil, &validateSuccessRequestHandler{t: t}) // reuse request handler to reduce boilerplate

		res, err := handler.H(context.Background(), passingValidationValue)
		assert.NoError(t, err)
		assert.Empty(t, res)
	})
}

func TestJobValidatingDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful job", func(t *testing.T) {
		t.Parallel()

		handler := app.NewValidatedJob[structWithValidationTags](validator.New(), &validateSuccessCommandHandler{t: t}) // reuse command handler to reduce boilerplate

		err := handler.H(context.Background(), passingValidationValue)
		assert.NoError(t, err)
	})

	t.Run("failed job", func(t *testing.T) {
		t.Parallel()

		handler := app.NewValidatedJob[structWithValidationTags](validator.New(), &validateFailureCommandHandler{t: t}) // reuse command handler to reduce boilerplate

		err := handler.H(context.Background(), structWithValidationTags{})
		validationErrors := validator.ValidationErrors{}
		_ = errors.As(err, &validationErrors)

		assert.Error(t, err)
		assert.Len(t, validationErrors, 2)
	})

	t.Run("with default validator", func(t *testing.T) {
		t.Parallel()

		handler := app.NewValidatedJob[structWithValidationTags](nil, &validateSuccessCommandHandler{t: t}) // reuse command handler to reduce boilerplate

		err := handler.H(context.Background(), passingValidationValue)
		assert.NoError(t, err)
	})
}

type structWithValidationTags struct {
	Val0 string `validate:"required"`
	Val1 string `validate:"min=2"`
}

var passingValidationValue = structWithValidationTags{
	Val0: "testValue",
	Val1: "testValue",
}

type validateSuccessRequestHandler struct{ t *testing.T }

func (h *validateSuccessRequestHandler) H(ctx context.Context, _ structWithValidationTags) (response, error) {
	h.t.Helper()
	assert.True(h.t, app.PassedValidation(ctx))

	return response{}, nil
}

type validateFailureRequestHandler struct{ t *testing.T }

func (h *validateFailureRequestHandler) H(ctx context.Context, _ structWithValidationTags) (response, error) {
	h.t.Helper()
	assert.False(h.t, app.PassedValidation(ctx))

	return response{}, nil
}

type validateSuccessCommandHandler struct{ t *testing.T }

func (h *validateSuccessCommandHandler) H(ctx context.Context, _ structWithValidationTags) error {
	h.t.Helper()
	assert.True(h.t, app.PassedValidation(ctx))

	return nil
}

type validateFailureCommandHandler struct{ t *testing.T }

func (h *validateFailureCommandHandler) H(ctx context.Context, _ structWithValidationTags) error {
	h.t.Helper()
	assert.False(h.t, app.PassedValidation(ctx))

	return nil
}
