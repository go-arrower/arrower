package mw

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metric wraps an application function / command with metric measurements.
//
// This middleware provides two metrics:
// usecases_total as a counter of all usecases called and
// usecases_duration_seconds as a histogram of execution time of the use cases.
// Although usecases_duration_seconds_count could be used, but the naming of usecases_total is more expressive.
func Metric[in, out any, F DecoratorFunc[in, out]](meterProvider metric.MeterProvider, next F) F { //nolint:ireturn,lll // valid use of generics
	meter := meterProvider.Meter("arrower.application") // trace.WithInstrumentationVersion("0.0.0"),

	counter, _ := meter.Int64Counter("usecases", metric.WithDescription("a simple counter"))
	duration, _ := meter.Float64Histogram("usecases_duration_seconds", metric.WithDescription("a simple hist"))

	return func(ctx context.Context, in in) (out, error) {
		var (
			opt     metric.MeasurementOption
			cmdName = commandName(in)
		)

		start := time.Now()
		defer func() {
			end := time.Since(start)

			counter.Add(ctx, 1, opt)
			duration.Record(ctx, end.Seconds(), opt)
		}()

		opt = metric.WithAttributes(
			attribute.String("command", cmdName),
			attribute.String("status", "success"),
		)

		result, err := next(ctx, in)
		if err != nil {
			opt = metric.WithAttributes(
				attribute.String("command", cmdName),
				attribute.String("status", "failure"),
			)
		}

		return result, err
	}
}

// MetricU see Metric.
func MetricU[in any, F DecoratorFuncUnary[in]](meterProvider metric.MeterProvider, next F) F { //nolint:ireturn,lll // valid use of generics
	meter := meterProvider.Meter("arrower.application") // trace.WithInstrumentationVersion("0.0.0"),

	counter, _ := meter.Int64Counter("usecases", metric.WithDescription("a simple counter"))
	duration, _ := meter.Float64Histogram("usecases_duration_seconds", metric.WithDescription("a simple hist"))

	return func(ctx context.Context, in in) error {
		var (
			opt     metric.MeasurementOption
			cmdName = commandName(in)
		)

		start := time.Now()
		defer func() {
			end := time.Since(start)

			counter.Add(ctx, 1, opt)
			duration.Record(ctx, end.Seconds(), opt)
		}()

		opt = metric.WithAttributes(
			attribute.String("command", cmdName),
			attribute.String("status", "success"),
		)

		err := next(ctx, in)
		if err != nil {
			opt = metric.WithAttributes(
				attribute.String("command", cmdName),
				attribute.String("status", "failure"),
			)
		}

		return err
	}
}
