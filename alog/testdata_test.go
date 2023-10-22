package alog_test

import (
	"context"
	"errors"
	"log/slog"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	applicationMsg = "application message"
)

var errSomething = errors.New("some error")

// // //  ///  /// /// ///
// // // fakes /// /// ///
// // //  ///  /// /// ///

// fakeSpan is an implementation of Span that is minimal for asserting tests.
type fakeSpan struct {
	ID byte

	eventName    string
	eventOptions []trace.EventOption

	statusErrorCode codes.Code
	statusErrorMsg  string
}

var _ trace.Span = (*fakeSpan)(nil)

func (fakeSpan) SpanContext() trace.SpanContext {
	return trace.SpanContext{}.WithTraceID([16]byte{1}).WithSpanID([8]byte{1})
}

func (s *fakeSpan) IsRecording() bool { return false }

func (s *fakeSpan) SetStatus(code codes.Code, msg string) {
	s.statusErrorCode = code
	s.statusErrorMsg = msg
}

func (s *fakeSpan) SetError(bool) {}

func (s *fakeSpan) SetAttributes(...attribute.KeyValue) {}

func (s *fakeSpan) End(...trace.SpanEndOption) {}

func (s *fakeSpan) RecordError(error, ...trace.EventOption) {}

func (s *fakeSpan) AddEvent(name string, opts ...trace.EventOption) {
	s.eventName = name
	s.eventOptions = opts
}

func (s *fakeSpan) SetName(string) {}

func (s *fakeSpan) TracerProvider() trace.TracerProvider { return &fakeTraceProvider{} } //nolint:ireturn

type fakeTraceProvider struct{}

func (f *fakeTraceProvider) Tracer(_ string, _ ...trace.TracerOption) trace.Tracer { //nolint:ireturn
	return &fakeTracer{}
}

type fakeTracer struct{}

func (f *fakeTracer) Start(ctx context.Context, spanName string, _ ...trace.SpanStartOption) (context.Context, trace.Span) { //nolint:ireturn
	// Ensures Start() is called from within *ArrowerLogger.Handle.
	// If not equal the nil will make the test panic, without t present
	// Used in "record a span for the handle method itself"
	assert.Equal(nil, "log", spanName)

	return ctx, &fakeSpan{}
}

type failingHandler struct{}

var _ slog.Handler = (*failingHandler)(nil)

func (f failingHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (f failingHandler) Handle(_ context.Context, _ slog.Record) error {
	return errSomething
}

func (f failingHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	panic("implement me")
}

func (f failingHandler) WithGroup(_ string) slog.Handler {
	panic("implement me")
}
