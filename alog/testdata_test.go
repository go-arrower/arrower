package alog_test

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/slog"
)

const (
	applicationMsg = "application message"
)

var errSomething = errors.New("some error")

// // //  ///  /// /// ///
// // // fakes /// /// ///
// // //  ///  /// /// ///

// fakeSpan is an implementation of Span that is minimal for asserting tests.
type fakeSpan struct { //nolint:govet // fieldalignment does not matter in test cases
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

func (s *fakeSpan) TracerProvider() trace.TracerProvider { return nil } //nolint:ireturn

type failingHandler struct{}

var _ slog.Handler = (*failingHandler)(nil)

func (f failingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (f failingHandler) Handle(ctx context.Context, record slog.Record) error {
	return errSomething
}

func (f failingHandler) WithAttrs(attrs []slog.Attr) slog.Handler { //nolint:ireturn
	panic("implement me")
}

func (f failingHandler) WithGroup(name string) slog.Handler { //nolint:ireturn
	panic("implement me")
}
