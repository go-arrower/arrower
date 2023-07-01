package alog

import (
	"context"
	"errors"
	"io"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/slog"
)

// LoggerOpt allows to initialise a logger with custom options.
type LoggerOpt func(logger *ArrowerLogger)

// WithHandler adds a slog.Handler to be logged to. You can set as many as you want.
func WithHandler(h slog.Handler) LoggerOpt {
	return func(l *ArrowerLogger) {
		l.handlers = append(l.handlers, h)
	}
}

// WithLevel initialises the logger with a starting level.
// To change the level at runtime use *ArrowerLogger.SetLevel.
func WithLevel(level slog.Level) LoggerOpt {
	return func(l *ArrowerLogger) {
		l.level = &level
	}
}

// New returns a production ready logger.
//
// If no options are given it creates a default handler, logging JSON to Stderr.
// Otherwise, use WithHandler to set your own loggers. For an example, see NewDevelopment.
func New(opts ...LoggerOpt) *slog.Logger {
	return slog.New(NewArrowerHandler(opts...))
}

// NewDevelopment returns a logger ready for local development purposes.
func NewDevelopment() *slog.Logger {
	return New(
		WithLevel(slog.LevelDebug),
		WithHandler(slog.NewTextHandler(os.Stderr, getDefaultHandlerOptions())),
		WithHandler(NewLokiHandler(nil)),
	)
}

// NewTest returns a logger suited for test cases. It writes to the given io.Writer.
func NewTest(w io.Writer) *slog.Logger {
	return slog.New(NewArrowerHandler(
		WithLevel(slog.LevelDebug),
		WithHandler(slog.NewTextHandler(w, nil)),
	))
}

// NewArrowerHandler implements the main arrower specific logging logic and features.
// It does not output anything directly and relies on other slog.Handlers to do so.
//
// For the options, see New.
func NewArrowerHandler(opts ...LoggerOpt) *ArrowerLogger {
	var (
		defaultLevel    = slog.LevelInfo
		defaultHandlers = []slog.Handler{slog.NewJSONHandler(os.Stderr, getDefaultHandlerOptions())}
	)

	logger := &ArrowerLogger{
		handlers: []slog.Handler{},
		level:    &defaultLevel,
	}

	for _, opt := range opts {
		opt(logger)
	}

	hasCustomHandlers := len(logger.handlers) != 0
	if !hasCustomHandlers {
		logger.handlers = defaultHandlers
	}

	return logger
}

// Ensure ArrowerLogger implements slog.Handler.
var _ slog.Handler = (*ArrowerLogger)(nil)

// ArrowerLogger is the main handler of arrower, offering to log to multiple handlers,
// filtering of context & users.
// It's also doing all the lifting for observability, see #adl.
type ArrowerLogger struct {
	// level reports the minimum record level that will be logged.
	// The handler discards records with lower levels.
	// If level is nil, the handler assumes LevelInfo.
	// The handler calls level.level for each record processed;
	// to adjust the minimum level dynamically, use a LevelVar.
	// This IS the level for all handlers. The level of the handlers set via
	// WithHandler are ignored.
	level *slog.Level

	// handlers is a list which all get called with the same log message.
	handlers []slog.Handler
}

func (l *ArrowerLogger) Handlers() []slog.Handler {
	return l.handlers
}

// Level returns the log level of the handler.
func (l *ArrowerLogger) Level() slog.Level {
	return l.level.Level()
}

// SetLevel changes the level for all loggers set with WithHandler(). Even the ones "copied" via any WithX method.
// All groups will have the same level.
func (l *ArrowerLogger) SetLevel(level slog.Level) {
	*l.level = level
}

func (l *ArrowerLogger) Enabled(ctx context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo

	if l.level != nil {
		minLevel = l.level.Level()
	}

	return level >= minLevel
}

func (l *ArrowerLogger) Handle(ctx context.Context, record slog.Record) error {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() { //nolint:staticcheck // here as a tmp note => remove once this method is cleaned up
		//	return s.h.Handle(r)
	}

	record = addTraceAndSpanIDsToLogs(span, record)
	addLogsToActiveSpanAsEvent(span, record)

	var retErr error

	for _, h := range l.handlers {
		err := h.Handle(ctx, record)
		retErr = errors.Join(retErr, err)
	}

	return retErr
}

func addTraceAndSpanIDsToLogs(span trace.Span, record slog.Record) slog.Record {
	sCtx := span.SpanContext()
	attrs := make([]slog.Attr, 0)

	if sCtx.HasTraceID() {
		attrs = append(attrs,
			slog.Attr{Key: "traceID", Value: slog.StringValue(sCtx.TraceID().String())},
		)
	}

	if sCtx.HasSpanID() {
		attrs = append(attrs,
			slog.Attr{Key: "spanID", Value: slog.StringValue(sCtx.SpanID().String())},
		)
	}

	if len(attrs) > 0 {
		record.AddAttrs(attrs...)
	}

	return record
}

func addLogsToActiveSpanAsEvent(span trace.Span, record slog.Record) {
	attrs := make([]attribute.KeyValue, 0)

	logSeverityKey := attribute.Key("log.severity")
	logMessageKey := attribute.Key("log.message")

	attrs = append(attrs, logSeverityKey.String(record.Level.String()))
	attrs = append(attrs, logMessageKey.String(record.Message))

	record.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs,
			attribute.KeyValue{
				Key:   attribute.Key(a.Key),
				Value: attribute.StringValue(a.Value.String()),
			},
		)

		return true // process next attr
	})

	span.AddEvent("log", trace.WithAttributes(attrs...))

	if record.Level >= slog.LevelError {
		span.SetStatus(codes.Error, record.Message)
	}
}

func (l *ArrowerLogger) WithAttrs(attrs []slog.Attr) slog.Handler { //nolint:ireturn // required for slog.Handler
	handlers := make([]slog.Handler, len(l.handlers))

	for i, h := range l.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}

	return &ArrowerLogger{
		handlers: handlers,
		level:    l.level,
	}
}

func (l *ArrowerLogger) WithGroup(name string) slog.Handler { //nolint:ireturn // required for slog.Handler
	handlers := make([]slog.Handler, len(l.handlers))

	for i, h := range l.handlers {
		handlers[i] = h.WithGroup(name)
	}

	return &ArrowerLogger{
		handlers: handlers,
		level:    l.level,
	}
}

// Unwrap unwraps the given logger and returns a ArrowerLogger.
// In case of an invalid implementation of logger, it returns nil instead of an empty ArrowerLogger.
func Unwrap(logger Logger) *ArrowerLogger {
	if l, ok := logger.(*slog.Logger).Handler().(*ArrowerLogger); ok {
		return l
	}

	return nil
}

func getDefaultHandlerOptions() *slog.HandlerOptions {
	return &slog.HandlerOptions{
		AddSource:   true,
		Level:       nil, // this level is ignored, ArrowerLogger's level is used for all handlers.
		ReplaceAttr: MapLogLevelsToName,
	}
}
