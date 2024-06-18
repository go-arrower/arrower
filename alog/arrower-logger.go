package alog

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	ctx2 "github.com/go-arrower/arrower/ctx"
	"github.com/go-arrower/arrower/setting"
)

var (
	SettingLogLevel = setting.NewKey("arrower", "log", "level") //nolint:gochecknoglobals
	SettingLogUsers = setting.NewKey("arrower", "log", "users") //nolint:gochecknoglobals
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

func WithSettings(settings setting.Settings) LoggerOpt {
	return func(l *ArrowerLogger) {
		l.settings = settings
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
func NewDevelopment(pgx *pgxpool.Pool, settings setting.Settings) *slog.Logger {
	const batchSize = 10

	return New(
		WithLevel(slog.LevelDebug),
		WithHandler(slog.NewTextHandler(os.Stderr, getDebugHandlerOptions())),
		WithHandler(NewLokiHandler(nil)),
		WithHandler(NewPostgresHandler(pgx, &PostgresHandlerOptions{MaxBatchSize: batchSize, MaxTimeout: time.Second})),
		WithSettings(settings),
	)
}

// NewTest returns a logger suited for test cases. It writes to the given io.Writer.
func NewTest(w io.Writer) *slog.Logger {
	if w == nil {
		w = io.Discard
	}

	return slog.New(NewArrowerHandler(
		WithLevel(LevelDebug),
		WithHandler(slog.NewTextHandler(w, getDebugHandlerOptions())),
	))
}

// NewNoopLogger returns an implementation of Logger that performs no operations.
func NewNoopLogger() *slog.Logger {
	return slog.New(noopHandler{})
}

// NewArrowerHandler implements the main arrower specific logging logic and features.
// It does not output anything directly and relies on other slog.Handlers to do so.
// If no Handlers are provided via WithHandler, a default JSON handler logs to os.Stderr.
//
// For the options, see New.
func NewArrowerHandler(opts ...LoggerOpt) *ArrowerLogger {
	var (
		defaultLevel    = slog.LevelInfo
		defaultHandlers = []slog.Handler{slog.NewJSONHandler(os.Stderr, getDefaultHandlerOptions())}
	)

	logger := &ArrowerLogger{&tracedLogger{
		handlers: []slog.Handler{},
		level:    &defaultLevel,
		settings: nil,
	}}

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
// filtering of users.
// It's also doing all the lifting for observability, see #adl.
type ArrowerLogger struct {
	*tracedLogger
}

func (l *ArrowerLogger) Enabled(_ context.Context, level slog.Level) bool {
	if l.tracedLogger.UsesSettings() {
		return true // the check is done in Handle, so that the actual Enabled and Handle calls are in the same span.
	}

	// if no Settings are used, return early, so the record might not be created if not required anyway.
	return level >= *l.level
}

func (l *ArrowerLogger) Handle(ctx context.Context, record slog.Record) error {
	span := trace.SpanFromContext(ctx)

	newCtx, innerSpan := span.TracerProvider().Tracer("arrower.log").Start(ctx, "log")
	defer innerSpan.End()

	if !l.tracedLogger.Enabled(newCtx, record.Level) {
		return nil
	}

	return l.tracedLogger.Handle(newCtx, record)
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

func addLogsToActiveSpanAsEvent(span trace.Span, attrs []attribute.KeyValue, record slog.Record) {
	span.AddEvent("log", trace.WithAttributes(attrs...))

	if record.Level >= slog.LevelError {
		span.SetStatus(codes.Error, record.Message)
	}
}

func getAttrsFromRecord(record slog.Record) []attribute.KeyValue {
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

	return attrs
}

func (l *ArrowerLogger) WithAttrs(attrs []slog.Attr) slog.Handler {
	return l.tracedLogger.WithAttrs(attrs)
}

func (l *ArrowerLogger) WithGroup(name string) slog.Handler {
	return l.tracedLogger.WithGroup(name)
}

var _ slog.Handler = (*tracedLogger)(nil)

// tracedLogger is used so the whole call to the logger is traced.
// Otherwise, Enabled and Handle could not share the same span.
type tracedLogger struct {
	// settings are used to determine the log level. Especially useful if multiple
	// replicas should log with the same level.
	// The handler discards records with lower levels.
	// This IS the Level for all handlers.
	// The level of individual handlers set via WithHandler is ignored.
	//
	// It also determines if a record should
	// be logged based on a list of users. This is most useful when debugging user-specific issues.
	// If level is nil, the handler assumes slog.LevelInfo.
	settings setting.Settings

	// level reports the minimum record level that will be logged if no settings are present,
	// as a fallback.
	// The level of individual handlers set via WithHandler is ignored.
	level *slog.Level

	// handlers is a list which all get called with the same log message.
	handlers []slog.Handler
}

func (l *tracedLogger) UsesSettings() bool {
	return l.settings != nil
}

// SetLevel changes the level for all loggers set with WithHandler().
// Even the ones "copied" via any WithX method.
// All groups will have the same level.
func (l *tracedLogger) SetLevel(level slog.Level) {
	*l.level = level
}

func (l *tracedLogger) NumHandlers() int {
	return len(l.handlers)
}

// Level returns the log level of the handler.
func (l *tracedLogger) Level() slog.Level {
	return l.level.Level()
}

func (l *tracedLogger) Enabled(ctx context.Context, level slog.Level) bool {
	span := trace.SpanFromContext(ctx)

	if l.UsesSettings() { //nolint:nestif // prefer all rules clearly visible at ones.
		if userID, hasUser := ctx.Value(ctx2.CtxAuthUserID).(string); hasUser {
			val, err := l.settings.Setting(ctx, SettingLogUsers)
			if err == nil {
				var users []string

				val.MustUnmarshal(&users) //nolint:contextcheck // fp

				if slices.Contains(users, userID) {
					span.SetAttributes(
						attribute.Bool("enabled", true),
						attribute.String("userID", userID),
					)

					return true
				}
			}
		}

		val, err := l.settings.Setting(ctx, SettingLogLevel)
		if err == nil {
			enabled := level >= slog.Level(val.MustInt())
			span.SetAttributes(
				attribute.Bool("enabled", enabled),
				attribute.Int("level.record", int(level)),
				attribute.Int("level.setting", val.MustInt()),
			)

			return enabled
		}
	}

	enabled := level >= *l.level
	span.SetAttributes(
		attribute.Bool("enabled", enabled),
		attribute.Int("level.record", int(level)),
		attribute.Int("level.logger", int(*l.level)),
	)

	return enabled
}

func (l *tracedLogger) Handle(ctx context.Context, record slog.Record) error {
	span := trace.SpanFromContext(ctx)

	record = addTraceAndSpanIDsToLogs(span, record)

	attr, ok := FromContext(ctx)
	if ok {
		record.AddAttrs(attr...)
	}

	attrs := getAttrsFromRecord(record)

	span.SetAttributes(attrs...)
	addLogsToActiveSpanAsEvent(span, attrs, record)

	var retErr error

	for _, h := range l.handlers {
		err := h.Handle(ctx, record)
		retErr = errors.Join(retErr, err)
	}

	return retErr
}

func (l *tracedLogger) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(l.handlers))

	for i, h := range l.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}

	return &ArrowerLogger{&tracedLogger{
		handlers: handlers,
		level:    l.level,
		settings: l.settings,
	}}
}

func (l *tracedLogger) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(l.handlers))

	for i, h := range l.handlers {
		handlers[i] = h.WithGroup(name)
	}

	return &ArrowerLogger{&tracedLogger{
		handlers: handlers,
		level:    l.level,
		settings: l.settings,
	}}
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

// getDebugHandlerOptions is to keep the log output more readable, by removing not essential keys.
func getDebugHandlerOptions() *slog.HandlerOptions {
	opt := getDefaultHandlerOptions()
	opt.AddSource = false

	return opt
}
