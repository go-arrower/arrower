package alog_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
)

var ctx = context.Background()

const (
	applicationMsg = "application message"
)

var errSomething = errors.New("some error")

// // //  ///  /// /// ///
// // // fakes /// /// ///
// // //  ///  /// /// ///

// fakeSpan is an implementation of Span that is minimal for asserting tests.
type fakeSpan struct {
	t *testing.T

	embedded.Span

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

func (s *fakeSpan) TracerProvider() trace.TracerProvider { //nolint:ireturn
	return &fakeTraceProvider{t: s.t}
}

func (s fakeSpan) AddLink(_ trace.Link) {
	panic("implement me")
}

type fakeTraceProvider struct {
	t *testing.T

	embedded.TracerProvider
}

func (f *fakeTraceProvider) Tracer(_ string, _ ...trace.TracerOption) trace.Tracer { //nolint:ireturn
	return &fakeTracer{t: f.t}
}

type fakeTracer struct {
	t *testing.T

	embedded.Tracer
}

func (f *fakeTracer) Start(ctx context.Context, spanName string, _ ...trace.SpanStartOption) (context.Context, trace.Span) { //nolint:ireturn
	// Ensures Start() is called from within *arrowerHandler.Handle.
	// If not equal the nil will make the test panic, without t present
	// Used in "record a span for the handle method itself"
	assert.Equal(f.t, "log", spanName)

	return ctx, &fakeSpan{t: f.t}
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

type fakeNoSettingsSpan struct {
	embedded.Span
}

func (f fakeNoSettingsSpan) End(_ ...trace.SpanEndOption) {
	panic("implement me")
}

func (f fakeNoSettingsSpan) AddEvent(_ string, _ ...trace.EventOption) {
	panic("implement me")
}

func (f fakeNoSettingsSpan) IsRecording() bool {
	panic("implement me")
}

func (f fakeNoSettingsSpan) RecordError(_ error, _ ...trace.EventOption) {
	panic("implement me")
}

func (f fakeNoSettingsSpan) SpanContext() trace.SpanContext {
	panic("implement me")
}

func (f fakeNoSettingsSpan) SetStatus(_ codes.Code, _ string) {
	panic("implement me")
}

func (f fakeNoSettingsSpan) SetName(_ string) {
	panic("implement me")
}

func (f fakeNoSettingsSpan) SetAttributes(_ ...attribute.KeyValue) {
	panic("implement me")
}

func (f fakeNoSettingsSpan) TracerProvider() trace.TracerProvider {
	panic("implement me")
}

func (f fakeNoSettingsSpan) AddLink(_ trace.Link) {
	panic("implement me")
}

var _ trace.Span = (*fakeNoSettingsSpan)(nil)
