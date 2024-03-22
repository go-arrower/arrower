package app

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func NewTracedRequest[Req any, Res any](traceProvider trace.TracerProvider, req Request[Req, Res]) Request[Req, Res] {
	return &requestTracingDecorator[Req, Res]{
		tracer: traceProvider.Tracer("arrower.application"), // trace.WithInstrumentationVersion("0.0.0")
		base:   req,
	}
}

type requestTracingDecorator[Req any, Res any] struct {
	tracer trace.Tracer
	base   Request[Req, Res]
}

func (d *requestTracingDecorator[Req, Res]) H(ctx context.Context, req Req) (Res, error) { //nolint:ireturn,lll // valid use of generics
	cmdName := commandName(req)

	newCtx, span := d.tracer.Start(ctx, "usecase",
		trace.WithAttributes(attribute.String("command", cmdName)),
	)
	defer span.End()

	result, err := d.base.H(newCtx, req)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return result, err //nolint:wrapcheck // decorate but not change anything
}

func NewTracedCommand[C any](traceProvider trace.TracerProvider, cmd Command[C]) Command[C] {
	return &commandTracingDecorator[C]{
		tracer: traceProvider.Tracer("arrower.application"),
		base:   cmd,
	}
}

type commandTracingDecorator[C any] struct {
	tracer trace.Tracer
	base   Command[C]
}

func (d *commandTracingDecorator[C]) H(ctx context.Context, cmd C) error {
	cmdName := commandName(cmd)

	newCtx, span := d.tracer.Start(ctx, "usecase",
		trace.WithAttributes(attribute.String("command", cmdName)),
	)
	defer span.End()

	err := d.base.H(newCtx, cmd)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return err //nolint:wrapcheck // decorate but not change anything
}

func NewTracedQuery[Q any, Res any](traceProvider trace.TracerProvider, query Query[Q, Res]) Query[Q, Res] {
	return &queryTracingDecorator[Q, Res]{
		tracer: traceProvider.Tracer("arrower.application"), // trace.WithInstrumentationVersion("0.0.0")
		base:   query,
	}
}

type queryTracingDecorator[Q any, Res any] struct {
	tracer trace.Tracer
	base   Query[Q, Res]
}

func (d *queryTracingDecorator[Q, Res]) H(ctx context.Context, query Q) (Res, error) { //nolint:ireturn,lll // valid use of generics
	cmdName := commandName(query)

	newCtx, span := d.tracer.Start(ctx, "usecase",
		trace.WithAttributes(attribute.String("command", cmdName)),
	)
	defer span.End()

	result, err := d.base.H(newCtx, query)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return result, err //nolint:wrapcheck // decorate but not change anything
}

func NewTracedJob[J any](traceProvider trace.TracerProvider, job Job[J]) Job[J] {
	return &jobTracingDecorator[J]{
		tracer: traceProvider.Tracer("arrower.application"),
		base:   job,
	}
}

type jobTracingDecorator[J any] struct {
	tracer trace.Tracer
	base   Job[J]
}

func (d *jobTracingDecorator[J]) H(ctx context.Context, job J) error {
	cmdName := commandName(job)

	newCtx, span := d.tracer.Start(ctx, "usecase",
		trace.WithAttributes(attribute.String("command", cmdName)),
	)
	defer span.End()

	err := d.base.H(newCtx, job)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return err //nolint:wrapcheck // decorate but not change anything
}
