package app

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func NewMeteredRequest[Req any, Res any](meterProvider metric.MeterProvider, req Request[Req, Res]) Request[Req, Res] {
	meter := meterProvider.Meter("arrower.application") // trace.WithInstrumentationVersion("0.0.0"),

	counter, _ := meter.Int64Counter("usecases", metric.WithDescription("a simple counter"))
	duration, _ := meter.Float64Histogram("usecases_duration_seconds", metric.WithDescription("a simple hist"))

	return &requestMeteringDecorator[Req, Res]{
		meter:    meter,
		counter:  counter,
		duration: duration,
		base:     req,
	}
}

type requestMeteringDecorator[Req any, Res any] struct {
	meter    metric.Meter
	counter  metric.Int64Counter
	duration metric.Float64Histogram
	base     Request[Req, Res]
}

func (d *requestMeteringDecorator[Req, Res]) H(ctx context.Context, req Req) (Res, error) {
	var (
		opt     metric.MeasurementOption
		cmdName = commandName(req)
	)

	start := time.Now()
	defer func() { //nolint:wsl_v5
		end := time.Since(start)

		d.counter.Add(ctx, 1, opt)
		d.duration.Record(ctx, end.Seconds(), opt)
	}()

	opt = metric.WithAttributes(
		attribute.String("command", cmdName),
		attribute.String("status", "success"),
	)

	result, err := d.base.H(ctx, req)
	if err != nil {
		opt = metric.WithAttributes(
			attribute.String("command", cmdName),
			attribute.String("status", "failure"),
		)
	}

	return result, err //nolint:wrapcheck // decorate but not change anything
}

func NewMeteredCommand[C any](meterProvider metric.MeterProvider, req Command[C]) Command[C] {
	meter := meterProvider.Meter("arrower.application") // trace.WithInstrumentationVersion("0.0.0"),

	counter, _ := meter.Int64Counter("usecases", metric.WithDescription("a simple counter"))
	duration, _ := meter.Float64Histogram("usecases_duration_seconds", metric.WithDescription("a simple hist"))

	return &commandMeteringDecorator[C]{
		meter:    meter,
		counter:  counter,
		duration: duration,
		base:     req,
	}
}

type commandMeteringDecorator[C any] struct {
	meter    metric.Meter
	counter  metric.Int64Counter
	duration metric.Float64Histogram
	base     Command[C]
}

func (d *commandMeteringDecorator[C]) H(ctx context.Context, cmd C) error {
	var (
		opt     metric.MeasurementOption
		cmdName = commandName(cmd)
	)

	start := time.Now()
	defer func() { //nolint:wsl_v5
		end := time.Since(start)

		d.counter.Add(ctx, 1, opt)
		d.duration.Record(ctx, end.Seconds(), opt)
	}()

	opt = metric.WithAttributes(
		attribute.String("command", cmdName),
		attribute.String("status", "success"),
	)

	err := d.base.H(ctx, cmd)
	if err != nil {
		opt = metric.WithAttributes(
			attribute.String("command", cmdName),
			attribute.String("status", "failure"),
		)
	}

	return err //nolint:wrapcheck // decorate but not change anything
}

func NewMeteredQuery[Q any, Res any](meterProvider metric.MeterProvider, query Query[Q, Res]) Query[Q, Res] {
	meter := meterProvider.Meter("arrower.application") // trace.WithInstrumentationVersion("0.0.0"),

	counter, _ := meter.Int64Counter("usecases", metric.WithDescription("a simple counter"))
	duration, _ := meter.Float64Histogram("usecases_duration_seconds", metric.WithDescription("a simple hist"))

	return &queryMeteringDecorator[Q, Res]{
		meter:    meter,
		counter:  counter,
		duration: duration,
		base:     query,
	}
}

type queryMeteringDecorator[Q any, Res any] struct {
	meter    metric.Meter
	counter  metric.Int64Counter
	duration metric.Float64Histogram
	base     Request[Q, Res]
}

func (d *queryMeteringDecorator[Q, Res]) H(ctx context.Context, query Q) (Res, error) {
	var (
		opt     metric.MeasurementOption
		cmdName = commandName(query)
	)

	start := time.Now()
	defer func() { //nolint:wsl_v5
		end := time.Since(start)

		d.counter.Add(ctx, 1, opt)
		d.duration.Record(ctx, end.Seconds(), opt)
	}()

	opt = metric.WithAttributes(
		attribute.String("command", cmdName),
		attribute.String("status", "success"),
	)

	result, err := d.base.H(ctx, query)
	if err != nil {
		opt = metric.WithAttributes(
			attribute.String("command", cmdName),
			attribute.String("status", "failure"),
		)
	}

	return result, err //nolint:wrapcheck // decorate but not change anything
}

func NewMeteredJob[J any](meterProvider metric.MeterProvider, req Job[J]) Job[J] {
	meter := meterProvider.Meter("arrower.application") // trace.WithInstrumentationVersion("0.0.0"),

	counter, _ := meter.Int64Counter("usecases", metric.WithDescription("a simple counter"))
	duration, _ := meter.Float64Histogram("usecases_duration_seconds", metric.WithDescription("a simple hist"))

	return &jobMeteringDecorator[J]{
		meter:    meter,
		counter:  counter,
		duration: duration,
		base:     req,
	}
}

type jobMeteringDecorator[J any] struct {
	meter    metric.Meter
	counter  metric.Int64Counter
	duration metric.Float64Histogram
	base     Job[J]
}

func (d *jobMeteringDecorator[J]) H(ctx context.Context, job J) error {
	var (
		opt     metric.MeasurementOption
		cmdName = commandName(job)
	)

	start := time.Now()
	defer func() { //nolint:wsl_v5
		end := time.Since(start)

		d.counter.Add(ctx, 1, opt)
		d.duration.Record(ctx, end.Seconds(), opt)
	}()

	opt = metric.WithAttributes(
		attribute.String("command", cmdName),
		attribute.String("status", "success"),
	)

	err := d.base.H(ctx, job)
	if err != nil {
		opt = metric.WithAttributes(
			attribute.String("command", cmdName),
			attribute.String("status", "failure"),
		)
	}

	return err //nolint:wrapcheck // decorate but not change anything
}
