package alog

import (
	"context"
	"errors"
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
type LoggerOpt func(logger *arrowerHandler)

// WithHandler adds a slog.Handler to be logged to.
// You can set as many as you want.
func WithHandler(h slog.Handler) LoggerOpt {
	return func(l *arrowerHandler) {
		l.handlers = append(l.handlers, h)
	}
}

// WithLevel initialises the logger with a starting level.
// To change the level at runtime use ArrowerLogger.SetLevel:
// Unwrap(logger).SetLevel(LevelInfo).
func WithLevel(level slog.Level) LoggerOpt {
	return func(l *arrowerHandler) {
		l.level = &level
	}
}

// WithSettings initialises the logger to use the settings.
// Via the settings the logger's level and other properties
// are controlled dynamically at run time.
func WithSettings(settings setting.Settings) LoggerOpt {
	return func(l *arrowerHandler) {
		l.settings = settings
	}
}

// New returns a production ready logger.
//
// If no options are given it creates a default handler, logging JSON to Stderr.
// Otherwise, use WithHandler to set your own loggers.
// For an example of options at work, see NewDevelopment.
func New(opts ...LoggerOpt) *slog.Logger {
	return slog.New(newArrowerHandler(opts...))
}

// NewDevelopment returns a logger ready for local development purposes.
// If pgx is nil the returned logger is not initialised to use
// any of the database related features of alog.
func NewDevelopment(pgx *pgxpool.Pool) *slog.Logger {
	const batchSize = 10

	config := []LoggerOpt{
		WithLevel(slog.LevelDebug),
		WithHandler(slog.NewTextHandler(os.Stderr, getDebugHandlerOptions())),
		WithHandler(NewLokiHandler(nil)),
	}

	if pgx != nil {
		config = append(config,
			WithHandler(NewPostgresHandler(pgx, &PostgresHandlerOptions{MaxBatchSize: batchSize, MaxTimeout: time.Second})),
			WithSettings(setting.NewPostgresSettings(pgx)))
	}

	return New(
		config...,
	)
}

// NewNoop returns an implementation of Logger that performs no operations.
// Ideal as dependency in tests.
func NewNoop() *slog.Logger {
	return slog.New(noopHandler{})
}

// newArrowerHandler implements the main arrower specific logging logic and features.
// It does not output anything directly and relies on other slog.Handlers to do so.
// If no Handlers are provided via WithHandler, a default JSON handler logs to os.Stderr.
func newArrowerHandler(opts ...LoggerOpt) *arrowerHandler {
	var (
		defaultLevel    = slog.LevelInfo
		defaultHandlers = []slog.Handler{slog.NewJSONHandler(os.Stderr, getDefaultHandlerOptions())}
	)

	logger := &arrowerHandler{&tracedHandler{
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

// arrowerHandler is the main handler of arrower, offering to log to multiple handlers,
// filtering of users.
// It's also doing all the lifting for observability, see #adl.
type arrowerHandler struct {
	*tracedHandler
}

var _ ArrowerLogger = (*arrowerHandler)(nil)

var _ slog.Handler = (*arrowerHandler)(nil)

func (l *arrowerHandler) Enabled(_ context.Context, level slog.Level) bool {
	if l.tracedHandler.UsesSettings() {
		// the check is done in Handle,
		// so that the actual Enabled and Handle calls are in the same span.
		return true
	}

	// if no settings are used, return early,
	// so the record might not be created if not required anyway.
	return level >= *l.level
}

func (l *arrowerHandler) Handle(ctx context.Context, record slog.Record) error {
	span := trace.SpanFromContext(ctx)

	newCtx, innerSpan := span.TracerProvider().Tracer("arrower.log").Start(ctx, "log")
	defer innerSpan.End()

	if !l.tracedHandler.Enabled(newCtx, record.Level) {
		return nil
	}

	return l.tracedHandler.Handle(newCtx, record)
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

func (l *arrowerHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return l.tracedHandler.WithAttrs(attrs)
}

func (l *arrowerHandler) WithGroup(name string) slog.Handler {
	return l.tracedHandler.WithGroup(name)
}

var _ slog.Handler = (*tracedHandler)(nil)

// tracedHandler is used so the whole call to the logger is traced.
// Otherwise, Enabled and Handle could not share the same span.
type tracedHandler struct {
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

func (l *tracedHandler) Enabled(ctx context.Context, level slog.Level) bool {
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

func (l *tracedHandler) Handle(ctx context.Context, record slog.Record) error {
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

func (l *tracedHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(l.handlers))

	for i, h := range l.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}

	return &arrowerHandler{&tracedHandler{
		handlers: handlers,
		level:    l.level,
		settings: l.settings,
	}}
}

func (l *tracedHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(l.handlers))

	for i, h := range l.handlers {
		handlers[i] = h.WithGroup(name)
	}

	return &arrowerHandler{&tracedHandler{
		handlers: handlers,
		level:    l.level,
		settings: l.settings,
	}}
}

// SetLevel changes the level for all loggers set with WithHandler().
// Even the ones "copied" via any WithX method.
// All groups will have the same level.
func (l *tracedHandler) SetLevel(level slog.Level) {
	*l.level = level
}

// Level returns the log level of the handler.
func (l *tracedHandler) Level() slog.Level {
	return l.level.Level()
}

// UsesSettings returns whether this logger is initialised to use settings.
func (l *tracedHandler) UsesSettings() bool {
	return l.settings != nil
}

func (l *tracedHandler) NumHandlers() int {
	return len(l.handlers)
}

// ArrowerLogger is an extension to Logger and slog.Logger and offers
// additional control over the logger at run time.
// Unwrap a logger to get access to this features.
type ArrowerLogger interface {
	SetLevel(level slog.Level)
	Level() slog.Level
	UsesSettings() bool
	// NumHandlers
}

// Unwrap unwraps the given logger and returns a ArrowerLogger.
// In case of an invalid implementation of logger,
// it returns nil.
func Unwrap(logger Logger) ArrowerLogger { //nolint:ireturn,lll // interface required to return a TestLogger and arrowerHandler
	if l, ok := logger.(*TestLogger); ok {
		return l
	}

	if l, ok := logger.(*slog.Logger).Handler().(*arrowerHandler); ok {
		return l
	}

	return nil
}

func getDefaultHandlerOptions() *slog.HandlerOptions {
	return &slog.HandlerOptions{
		AddSource:   true,
		Level:       nil, // this level is ignored, arrowerHandler's level is used for all handlers.
		ReplaceAttr: MapLogLevelsToName,
	}
}

// getDebugHandlerOptions is to keep the log output more readable, by removing not essential keys.
func getDebugHandlerOptions() *slog.HandlerOptions {
	opt := getDefaultHandlerOptions()
	opt.AddSource = false

	return opt
}
