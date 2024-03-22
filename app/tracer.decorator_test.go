package app_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"

	"github.com/go-arrower/arrower/app"
)

func TestRequestTracingDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful request", func(t *testing.T) {
		t.Parallel()

		handler := app.NewTracedRequest[request, response](newFakeTracer(t), &requestSuccessHandler{})

		_, err := handler.H(context.Background(), request{})
		assert.NoError(t, err)
	})

	t.Run("failed request", func(t *testing.T) {
		t.Parallel()

		handler := app.NewTracedRequest[request, response](newFakeTracer(t), &requestFailureHandler{})

		_, err := handler.H(context.Background(), request{})
		assert.Error(t, err)
	})
}

func TestCommandTracingDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful command", func(t *testing.T) {
		t.Parallel()

		handler := app.NewTracedCommand[request](newFakeTracer(t), &commandSuccessHandler{})

		err := handler.H(context.Background(), request{})
		assert.NoError(t, err)
	})

	t.Run("failed command", func(t *testing.T) {
		t.Parallel()

		handler := app.NewTracedCommand[request](newFakeTracer(t), &commandFailureHandler{})

		err := handler.H(context.Background(), request{})
		assert.Error(t, err)
	})
}

func TestQueryTracingDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful query", func(t *testing.T) {
		t.Parallel()

		handler := app.NewTracedQuery[request, response](newFakeTracer(t), &requestSuccessHandler{})

		_, err := handler.H(context.Background(), request{})
		assert.NoError(t, err)
	})

	t.Run("failed query", func(t *testing.T) {
		t.Parallel()

		handler := app.NewTracedQuery[request, response](newFakeTracer(t), &requestFailureHandler{})

		_, err := handler.H(context.Background(), request{})
		assert.Error(t, err)
		assert.Error(t, err)
	})
}

func TestJobTracingDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful job", func(t *testing.T) {
		t.Parallel()

		handler := app.NewTracedJob[request](newFakeTracer(t), &commandSuccessHandler{})

		err := handler.H(context.Background(), request{})
		assert.NoError(t, err)
	})

	t.Run("failed job", func(t *testing.T) {
		t.Parallel()

		handler := app.NewTracedJob[request](newFakeTracer(t), &commandFailureHandler{})

		err := handler.H(context.Background(), request{})
		assert.Error(t, err)
	})
}

func newFakeTracer(t *testing.T) trace.TracerProvider { //nolint:ireturn
	t.Helper()

	return &fakeTracerProvider{t: t}
}

type (
	fakeTracerProvider struct {
		embedded.TracerProvider
		t *testing.T
	}
	fakeTracer struct {
		embedded.Tracer
		t *testing.T
	}
	fakeSpan struct {
		embedded.Span
		t *testing.T
	}
)

var _ trace.TracerProvider = (*fakeTracerProvider)(nil)

func (f fakeTracerProvider) Tracer(name string, _ ...trace.TracerOption) trace.Tracer { //nolint:ireturn
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
	// Would like to add an assertion for the ex. of attribute "command" in opts[0]
	// If you know how to assert on an attribute key, please let me know.

	return ctx, &fakeSpan{t: f.t}
}

var _ trace.Span = (*fakeSpan)(nil)

func (f fakeSpan) End(_ ...trace.SpanEndOption) {
	f.t.Helper()

	// NOTE: this assertion does not work if the call to End() is not made
	// This method has to be called for each use case. No action to take for this fake.
	assert.True(f.t, true)
}

func (f fakeSpan) AddEvent(_ string, _ ...trace.EventOption) {
	panic("implement me")
}

func (f fakeSpan) IsRecording() bool {
	panic("implement me")
}

func (f fakeSpan) RecordError(_ error, _ ...trace.EventOption) {
	panic("implement me")
}

func (f fakeSpan) SpanContext() trace.SpanContext {
	panic("implement me")
}

func (f fakeSpan) SetStatus(code codes.Code, _ string) {
	f.t.Helper()

	// Status is only set in case of error returned by the use case.
	// So if the use case succeeds this assertion is not run and thus passes the test.
	assert.Equal(f.t, codes.Error, code)
}

func (f fakeSpan) SetName(_ string) {
	panic("implement me")
}

func (f fakeSpan) SetAttributes(_ ...attribute.KeyValue) {
	panic("implement me")
}

func (f fakeSpan) TracerProvider() trace.TracerProvider { //nolint:ireturn
	panic("implement me")
}
