// Package app provides common decorators for use cases in the application layer.
package app

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-arrower/arrower/alog"
)

// Request can produce side effects and return data.
type Request[Req any, Res any] interface {
	H(ctx context.Context, req Req) (Res, error)
}

// Command produces side effects, e.g. mutate state.
type Command[C any] interface {
	H(ctx context.Context, cmd C) error
}

// Query does not produce side effects and returns data.
type Query[Q any, Res any] interface {
	H(ctx context.Context, query Q) (Res, error)
}

// Job produces side effects.
type Job[J any] interface {
	H(ctx context.Context, job J) error
}

// NewInstrumentedRequest is a convenience helper for easy dependency setup.
// The order of dependencies represents the order of calling.
func NewInstrumentedRequest[Req any, Res any](
	traceProvider trace.TracerProvider,
	meterProvider metric.MeterProvider,
	logger alog.Logger,
	cmd Request[Req, Res],
) Request[Req, Res] {
	return NewTracedRequest(traceProvider, NewMeteredRequest(meterProvider, NewLoggedRequest(logger, cmd)))
}

// NewInstrumentedCommand is a convenience helper for easy dependency setup.
// The order of dependencies represents the order of calling.
func NewInstrumentedCommand[C any](
	traceProvider trace.TracerProvider,
	meterProvider metric.MeterProvider,
	logger alog.Logger,
	cmd Command[C],
) Command[C] {
	return NewTracedCommand(traceProvider, NewMeteredCommand(meterProvider, NewLoggedCommand(logger, cmd)))
}

// NewInstrumentedQuery is a convenience helper for easy dependency setup.
// The order of dependencies represents the order of calling.
func NewInstrumentedQuery[Q any, Res any](
	traceProvider trace.TracerProvider,
	meterProvider metric.MeterProvider,
	logger alog.Logger,
	query Query[Q, Res],
) Query[Q, Res] {
	return NewTracedQuery(traceProvider, NewMeteredQuery(meterProvider, NewLoggedQuery(logger, query)))
}

// NewInstrumentedJob is a convenience helper for easy dependency setup.
// The order of dependencies represents the order of calling.
func NewInstrumentedJob[J any](
	traceProvider trace.TracerProvider,
	meterProvider metric.MeterProvider,
	logger alog.Logger,
	job Job[J],
) Job[J] {
	return NewTracedJob(traceProvider, NewMeteredJob(meterProvider, NewLoggedJob(logger, job)))
}

// commandName extracts a printable name from cmd in the format of: functionName.
//
// structName	 								=> strings.Split(fmt.Sprintf("%T", cmd), ".")[1]
// structname	 								=> strings.ToLower(strings.Split(fmt.Sprintf("%T", cmd), ".")[1])
// packageName.structName	 					=> fmt.Sprintf("%T", cmd)
// github.com/go-arrower/skeleton/.../package	=> fmt.Sprintln(reflect.TypeOf(cmd).PkgPath())
// structName is used, the other examples are for inspiration.
// The use case function can not be used, as it is anonymous / a closure returned by the use case constructor.
// Accessing the function name with runtime.Caller(4) will always lead to ".func1".
func commandName(cmd any) string {
	pkgPath := reflect.TypeOf(cmd).PkgPath()

	// example: github.com/go-arrower/skeleton/contexts/admin/internal/application_test
	// take string after /contexts/ and then take string before /internal/
	pkg0 := strings.Split(pkgPath, "/contexts/")

	hasContext := len(pkg0) == 2 //nolint:gomnd
	if hasContext {
		pkg1 := strings.Split(pkg0[1], "/internal/")
		if len(pkg1) == 2 { //nolint:gomnd
			context := pkg1[0]

			return fmt.Sprintf("%s.%T", context, cmd)
		}
	}

	// fallback: if the function is not called from a proper Context => packageName.structName
	return fmt.Sprintf("%T", cmd)
}
