package mw

import (
	"context"

	"github.com/go-playground/validator/v10"

	"github.com/go-arrower/arrower"
)

const CtxValidated arrower.CTXKey = "arrower.validated"

// Validate validates the incoming request / DTO in according to its validation tags and returns in a case of error.
// If you don't pass a validator a default one is used. Possible drawback: validator.Validate uses caching
// for information about your struct and validations.
// In case the validation is passed, it is recorded in the context and can be checked with PassedValidation.
func Validate[in, out any, F DecoratorFunc[in, out]](validate *validator.Validate, next F) F { //nolint:ireturn,lll // valid use of generics
	if validate == nil {
		validate = validator.New()
	}

	return func(ctx context.Context, in in) (out, error) {
		err := validate.Struct(in)
		if err != nil {
			return *new(out), err //nolint:wrapcheck // validation error is returned on purpose
		}

		return next(context.WithValue(ctx, CtxValidated, true), in)
	}
}

// ValidateU is like Validate but for functions only returning errors, e.g. jobs.
func ValidateU[in any, F DecoratorFuncUnary[in]](validate *validator.Validate, next F) F { //nolint:ireturn,lll // valid use of generics
	if validate == nil {
		validate = validator.New()
	}

	return func(ctx context.Context, in in) error {
		err := validate.Struct(in)
		if err != nil {
			return err //nolint:wrapcheck // validation error is returned on purpose
		}

		return next(context.WithValue(ctx, CtxValidated, true), in)
	}
}

// PassedValidation is a helper giving you feedback, if a request passed validation of this middleware.
// Use it in case you want to ensure that this middleware was called before continuing with your business logic.
// It can help to reduce the amount of "magic" that this middleware introduces to your code and safe guard you
// from a potential wrong setup of dependencies.
func PassedValidation(ctx context.Context) bool {
	if v, ok := ctx.Value(CtxValidated).(bool); ok {
		return v
	}

	return false
}
