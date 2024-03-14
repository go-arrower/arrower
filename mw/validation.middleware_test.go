package mw_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/mw"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	t.Run("successful command", func(t *testing.T) {
		t.Parallel()

		cmd := mw.Validate(validator.New(), exampleUseCaseEnsureValidated(t))

		res, err := cmd(context.Background(), passingValidationValue)
		assert.NoError(t, err)
		assert.NotEmpty(t, res)
	})

	t.Run("failed command", func(t *testing.T) {
		t.Parallel()

		cmd := mw.Validate(validator.New(), exampleUseCaseEnsureNotValidated(t))

		res, err := cmd(context.Background(), exampleRequest{})
		validationErrors := validator.ValidationErrors{}
		_ = errors.As(err, &validationErrors)

		assert.Error(t, err)
		assert.Len(t, validationErrors, 2)
		assert.Empty(t, res)
	})

	t.Run("with default validator", func(t *testing.T) {
		t.Parallel()

		cmd := mw.Validate(nil, exampleUseCaseEnsureValidated(t))

		res, err := cmd(context.Background(), passingValidationValue)
		assert.NoError(t, err)
		assert.NotEmpty(t, res)
	})
}

func TestValidateU(t *testing.T) {
	t.Parallel()

	t.Run("successful command", func(t *testing.T) {
		t.Parallel()

		cmd := mw.ValidateU(validator.New(), exampleUseCaseUEnsureValidated(t))

		err := cmd(context.Background(), passingValidationValue)
		assert.NoError(t, err)
	})

	t.Run("failed command", func(t *testing.T) {
		t.Parallel()

		cmd := mw.ValidateU(validator.New(), exampleUseCaseUEnsureNotValidated(t))

		err := cmd(context.Background(), exampleRequest{})
		validationErrors := validator.ValidationErrors{}
		_ = errors.As(err, &validationErrors)

		assert.Error(t, err)
		assert.Len(t, validationErrors, 2)
	})

	t.Run("with default validator", func(t *testing.T) {
		t.Parallel()

		cmd := mw.ValidateU(nil, exampleUseCaseUEnsureValidated(t))

		err := cmd(context.Background(), passingValidationValue)
		assert.NoError(t, err)
	})
}

type exampleRequest struct {
	Val0 string `validate:"required"`
	Val1 string `validate:"min=2"`
}
type exampleResponse struct {
	Ret0 string
}

var passingValidationValue = exampleRequest{
	Val0: "testValue",
	Val1: "testValue",
}

func exampleUseCaseEnsureNotValidated(t *testing.T) func(context.Context, exampleRequest) (exampleResponse, error) {
	t.Helper()

	return func(ctx context.Context, _ exampleRequest) (exampleResponse, error) {
		assert.False(t, mw.PassedValidation(ctx))

		return exampleResponse{Ret0: "non-empty-return"}, nil // do nothing, just simulate a use case.
	}
}

func exampleUseCaseEnsureValidated(t *testing.T) func(context.Context, exampleRequest) (exampleResponse, error) {
	t.Helper()

	return func(ctx context.Context, _ exampleRequest) (exampleResponse, error) {
		assert.True(t, mw.PassedValidation(ctx))

		return exampleResponse{Ret0: "non-empty-return"}, nil // do nothing, just simulate a use case.
	}
}

func exampleUseCaseUEnsureNotValidated(t *testing.T) func(context.Context, exampleRequest) error {
	t.Helper()

	return func(ctx context.Context, _ exampleRequest) error {
		assert.False(t, mw.PassedValidation(ctx))

		return nil // do nothing, just simulate a use case.
	}
}

func exampleUseCaseUEnsureValidated(t *testing.T) func(context.Context, exampleRequest) error {
	t.Helper()

	return func(ctx context.Context, _ exampleRequest) error {
		assert.True(t, mw.PassedValidation(ctx))

		return nil // do nothing, just simulate a use case.
	}
}
