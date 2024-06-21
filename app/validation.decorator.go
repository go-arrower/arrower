package app

import (
	"context"

	"github.com/go-playground/validator/v10"

	ctx2 "github.com/go-arrower/arrower/ctx"
)

const CtxValidated ctx2.CTXKey = "arrower.validated"

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

func NewValidatedRequest[Req any, Res any](validate *validator.Validate, req Request[Req, Res]) Request[Req, Res] {
	if validate == nil {
		validate = validator.New()
	}

	return &requestValidatingDecorator[Req, Res]{
		validate: validate,
		base:     req,
	}
}

type requestValidatingDecorator[Req any, Res any] struct {
	validate *validator.Validate
	base     Request[Req, Res]
}

func (d *requestValidatingDecorator[Req, Res]) H(ctx context.Context, req Req) (Res, error) { //nolint:ireturn,lll // valid use of generics
	err := d.validate.Struct(req)
	if err != nil {
		return *new(Res), err //nolint:wrapcheck // validation error is returned on purpose
	}

	return d.base.H(context.WithValue(ctx, CtxValidated, true), req) //nolint:wrapcheck // decorate but not change anything
}

func NewValidatedCommand[C any](validate *validator.Validate, cmd Command[C]) Command[C] {
	if validate == nil {
		validate = validator.New()
	}

	return &commandValidatingDecorator[C]{
		validate: validate,
		base:     cmd,
	}
}

type commandValidatingDecorator[C any] struct {
	validate *validator.Validate
	base     Command[C]
}

func (d *commandValidatingDecorator[C]) H(ctx context.Context, cmd C) error {
	err := d.validate.Struct(cmd)
	if err != nil {
		return err //nolint:wrapcheck // validation error is returned on purpose
	}

	return d.base.H(context.WithValue(ctx, CtxValidated, true), cmd) //nolint:wrapcheck // decorate but not change anything
}

func NewValidatedQuery[Q any, Res any](validate *validator.Validate, query Query[Q, Res]) Query[Q, Res] {
	if validate == nil {
		validate = validator.New()
	}

	return &queryValidatingDecorator[Q, Res]{
		validate: validate,
		base:     query,
	}
}

type queryValidatingDecorator[Q any, Res any] struct {
	validate *validator.Validate
	base     Query[Q, Res]
}

func (d *queryValidatingDecorator[Q, Res]) H(ctx context.Context, query Q) (Res, error) { //nolint:ireturn,lll // valid use of generics
	err := d.validate.Struct(query)
	if err != nil {
		return *new(Res), err //nolint:wrapcheck // validation error is returned on purpose
	}

	return d.base.H(context.WithValue(ctx, CtxValidated, true), query) //nolint:wrapcheck,lll // decorate but not change anything
}

func NewValidatedJob[J any](validate *validator.Validate, job Job[J]) Job[J] {
	if validate == nil {
		validate = validator.New()
	}

	return &jobValidatingDecorator[J]{
		validate: validate,
		base:     job,
	}
}

type jobValidatingDecorator[J any] struct {
	validate *validator.Validate
	base     Job[J]
}

func (d *jobValidatingDecorator[J]) H(ctx context.Context, job J) error {
	err := d.validate.Struct(job)
	if err != nil {
		return err //nolint:wrapcheck // validation error is returned on purpose
	}

	return d.base.H(context.WithValue(ctx, CtxValidated, true), job) //nolint:wrapcheck // decorate but not change anything
}
