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

		handler := app.NewValidatedRequest(validator.New(), app.TestRequestHandler(func(ctx context.Context, _ structWithValidationTags) (response, error) {
			assert.True(t, app.PassedValidation(ctx))
			return response{}, nil
		}))

		res, err := handler.H(context.Background(), passingValidationValue)
		assert.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("failed request", func(t *testing.T) {
		t.Parallel()

		handler := app.NewValidatedRequest(validator.New(), app.TestRequestHandler(func(ctx context.Context, _ structWithValidationTags) (response, error) {
			assert.False(t, app.PassedValidation(ctx))
			return response{}, nil
		}))

		res, err := handler.H(context.Background(), structWithValidationTags{})
		validationErrors := validator.ValidationErrors{}
		_ = errors.As(err, &validationErrors)

		assert.Error(t, err)
		assert.Len(t, validationErrors, 2)
		assert.Empty(t, res)
	})

	t.Run("with default validator", func(t *testing.T) {
		t.Parallel()

		handler := app.NewValidatedRequest(nil, app.TestRequestHandler(func(ctx context.Context, _ structWithValidationTags) (response, error) {
			assert.True(t, app.PassedValidation(ctx))
			return response{}, nil
		}))

		res, err := handler.H(context.Background(), passingValidationValue)
		assert.NoError(t, err)
		assert.Empty(t, res)
	})
}

func TestCommandValidatingDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful command", func(t *testing.T) {
		t.Parallel()

		handler := app.NewValidatedCommand(validator.New(), app.TestCommandHandler(func(ctx context.Context, _ structWithValidationTags) error {
			assert.True(t, app.PassedValidation(ctx))
			return nil
		}))

		err := handler.H(context.Background(), passingValidationValue)
		assert.NoError(t, err)
	})

	t.Run("failed command", func(t *testing.T) {
		t.Parallel()

		handler := app.NewValidatedCommand(validator.New(), app.TestCommandHandler(func(ctx context.Context, _ structWithValidationTags) error {
			assert.False(t, app.PassedValidation(ctx))
			return nil
		}))

		err := handler.H(context.Background(), structWithValidationTags{})
		validationErrors := validator.ValidationErrors{}
		_ = errors.As(err, &validationErrors)

		assert.Error(t, err)
		assert.Len(t, validationErrors, 2)
	})

	t.Run("with default validator", func(t *testing.T) {
		t.Parallel()

		handler := app.NewValidatedCommand(nil, app.TestCommandHandler(func(ctx context.Context, _ structWithValidationTags) error {
			assert.True(t, app.PassedValidation(ctx))
			return nil
		}))

		err := handler.H(context.Background(), passingValidationValue)
		assert.NoError(t, err)
	})
}

//nolint:dupl // decorators are basically identical but need a different signature
func TestQueryValidatingDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful query", func(t *testing.T) {
		t.Parallel()

		handler := app.NewValidatedQuery(validator.New(), app.TestQueryHandler(func(ctx context.Context, _ structWithValidationTags) (response, error) {
			assert.True(t, app.PassedValidation(ctx))
			return response{}, nil
		}))

		res, err := handler.H(context.Background(), passingValidationValue)
		assert.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("failed query", func(t *testing.T) {
		t.Parallel()

		handler := app.NewValidatedQuery(validator.New(), app.TestQueryHandler(func(ctx context.Context, _ structWithValidationTags) (response, error) {
			assert.False(t, app.PassedValidation(ctx))
			return response{}, nil
		}))

		res, err := handler.H(context.Background(), structWithValidationTags{})
		validationErrors := validator.ValidationErrors{}
		_ = errors.As(err, &validationErrors)

		assert.Error(t, err)
		assert.Len(t, validationErrors, 2)
		assert.Empty(t, res)
	})

	t.Run("with default validator", func(t *testing.T) {
		t.Parallel()

		handler := app.NewValidatedQuery(nil, app.TestQueryHandler(func(ctx context.Context, _ structWithValidationTags) (response, error) {
			assert.True(t, app.PassedValidation(ctx))
			return response{}, nil
		}))

		res, err := handler.H(context.Background(), passingValidationValue)
		assert.NoError(t, err)
		assert.Empty(t, res)
	})
}

func TestJobValidatingDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful job", func(t *testing.T) {
		t.Parallel()

		handler := app.NewValidatedJob(validator.New(), app.TestJobHandler(func(ctx context.Context, _ structWithValidationTags) error {
			assert.True(t, app.PassedValidation(ctx))
			return nil
		}))

		err := handler.H(context.Background(), passingValidationValue)
		assert.NoError(t, err)
	})

	t.Run("failed job", func(t *testing.T) {
		t.Parallel()

		handler := app.NewValidatedJob(validator.New(), app.TestJobHandler(func(ctx context.Context, _ structWithValidationTags) error {
			assert.False(t, app.PassedValidation(ctx))
			return nil
		}))

		err := handler.H(context.Background(), structWithValidationTags{})
		validationErrors := validator.ValidationErrors{}
		_ = errors.As(err, &validationErrors)

		assert.Error(t, err)
		assert.Len(t, validationErrors, 2)
	})

	t.Run("with default validator", func(t *testing.T) {
		t.Parallel()

		handler := app.NewValidatedJob(nil, app.TestJobHandler(func(ctx context.Context, _ structWithValidationTags) error {
			assert.True(t, app.PassedValidation(ctx))
			return nil
		}))

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
