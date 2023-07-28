package mw_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var errUseCaseFails = errors.New("some-error")

type exampleCommand struct{}

func newFakeTracer(t *testing.T) trace.TracerProvider { //nolint:ireturn
	t.Helper()

	return &fakeTracerProvider{t: t}
}

type (
	fakeTracerProvider struct {
		t *testing.T
	}
	fakeTracer struct {
		t *testing.T
	}
	fakeSpan struct {
		t *testing.T
	}
)

var _ trace.TracerProvider = (*fakeTracerProvider)(nil)

func (f fakeTracerProvider) Tracer(name string, options ...trace.TracerOption) trace.Tracer { //nolint:ireturn
	f.t.Helper()

	assert.Equal(f.t, "arrower.application", name)

	return &fakeTracer{t: f.t}
}

var _ trace.Tracer = (*fakeTracer)(nil)

func (f fakeTracer) Start( //nolint:ireturn
	ctx context.Context,
	spanName string,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	f.t.Helper()

	assert.Equal(f.t, "usecase", spanName)

	assert.Len(f.t, opts, 1)
	// If you know how to assert on a attribute key, please let me know.
	// Would like to add an assertion for the ex. of attribute "command" in opts[0]

	return ctx, &fakeSpan{t: f.t}
}

var _ trace.Span = (*fakeSpan)(nil)

func (f fakeSpan) End(options ...trace.SpanEndOption) {
	f.t.Helper()

	// This method has to be called for each use case. No action to take for this fake.
	assert.True(f.t, true)
}

func (f fakeSpan) AddEvent(name string, options ...trace.EventOption) {
	panic("implement me")
}

func (f fakeSpan) IsRecording() bool {
	panic("implement me")
}

func (f fakeSpan) RecordError(err error, options ...trace.EventOption) {
	panic("implement me")
}

func (f fakeSpan) SpanContext() trace.SpanContext {
	panic("implement me")
}

func (f fakeSpan) SetStatus(code codes.Code, description string) {
	f.t.Helper()

	// Status is only set in case of error returned by the use case.
	// So if the use case succeeds this assertion is not run and thus passes the test.
	assert.Equal(f.t, codes.Error, code)
}

func (f fakeSpan) SetName(name string) {
	panic("implement me")
}

func (f fakeSpan) SetAttributes(kv ...attribute.KeyValue) {
	panic("implement me")
}

func (f fakeSpan) TracerProvider() trace.TracerProvider { //nolint:ireturn
	panic("implement me")
}

var (
	//nolint:lll // needs to match the exact format expected from the prometheus endpoint
	// prepare the output, so it can be read multiple times by different tests, see:https://siongui.github.io/2018/10/28/go-read-twice-from-same-io-reader/
	rawMetricsForSucceedingUseCase, _ = io.ReadAll(strings.NewReader(`
# HELP usecases_duration_seconds a simple hist
# TYPE usecases_duration_seconds histogram
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="0"} 0
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="5"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="10"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="25"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="50"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="75"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="100"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="250"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="500"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="750"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="1000"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="2500"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="5000"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="7500"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="10000"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="+Inf"} 1
usecases_duration_seconds_sum{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version=""} 2.198e-06
usecases_duration_seconds_count{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version=""} 1
# HELP usecases_total a simple counter
# TYPE usecases_total counter
usecases_total{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",status="success"} 1
`))
	metricsForSucceedingUseCase  = bytes.NewReader(rawMetricsForSucceedingUseCase)
	metricsForSucceedingUseCaseU = bytes.NewReader(rawMetricsForSucceedingUseCase)

	//nolint:lll // needs to match the exact format expected from the prometheus endpoint
	rawMetricsForFailingUseCase, _ = io.ReadAll(strings.NewReader(`
# HELP usecases_duration_seconds a simple hist
# TYPE usecases_duration_seconds histogram
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="0"} 0
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="5"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="10"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="25"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="50"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="75"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="100"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="250"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="500"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="750"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="1000"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="2500"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="5000"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="7500"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="10000"} 1
usecases_duration_seconds_bucket{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",le="+Inf"} 1
usecases_duration_seconds_sum{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version=""} 2.198e-06
usecases_duration_seconds_count{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version=""} 1
# HELP usecases_total a simple counter
# TYPE usecases_total counter
usecases_total{command="mw_test.exampleCommand",otel_scope_name="arrower.application",otel_scope_version="",status="failure"} 1
`))
	metricsForFailingUseCase  = bytes.NewReader(rawMetricsForFailingUseCase)
	metricsForFailingUseCaseU = bytes.NewReader(rawMetricsForFailingUseCase)
)
