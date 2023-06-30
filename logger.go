package arrower

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/afiskon/promtail-client/promtail"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/slog"
)

const (
	// LevelDebug is used for arrower debug messages.
	LevelDebug = slog.Level(-8)

	// LevelTrace is used for arrower trace information, if you really want to know what is going on.
	LevelTrace = slog.Level(-12)
)

// LoggerOpt allows to initialise a logger with custom options.
type LoggerOpt func(logger *Logger)

// WithHandler adds a slog.Handler to be logged to. You can set as many as you want.
func WithHandler(h slog.Handler) LoggerOpt {
	return func(l *Logger) {
		l.handlers = append(l.handlers, h)
	}
}

// WithLevel initialises the logger with a starting level.
// To change the level at runtime use *Logger.SetLevel.
func WithLevel(level slog.Level) LoggerOpt {
	return func(l *Logger) {
		l.level = &level
	}
}

// NewLogger returns a production ready logger.
//
// If no options are given it creates a default handler, logging JSON to Stderr.
// Otherwise, use WithHandler to set your own loggers. For an example, see NewDevelopmentLogger.
func NewLogger(opts ...LoggerOpt) *slog.Logger {
	return slog.New(NewLogHandler(opts...))
}

func NewDevelopmentLogger() *slog.Logger {
	return NewLogger(
		WithLevel(slog.LevelDebug),
		WithHandler(slog.NewTextHandler(os.Stderr, getDefaultHandlerOptions())),
		WithHandler(NewLokiHandler(nil)),
	)
}

func NewTestLogger(w io.Writer) *slog.Logger {
	return slog.New(
		NewLogHandler(WithHandler(
			slog.NewTextHandler(w, nil)),
		),
	)
}

// NewLogHandler implements the main arrower specific logging logic and features.
// It does not output anything directly and relies on other slog.Handlers to do so.
//
// For the options, see NewLogger.
func NewLogHandler(opts ...LoggerOpt) *Logger {
	var (
		defaultLevel    = slog.LevelInfo
		defaultHandlers = []slog.Handler{slog.NewJSONHandler(os.Stderr, getDefaultHandlerOptions())}
	)

	logger := &Logger{
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

// Ensure Logger implements slog.Handler.
var _ slog.Handler = (*Logger)(nil)

// Logger is the main handler of arrower, offering to log to multiple handlers,
// filtering of context & users.
// It's also doing all the lifting for observability, see #adl.
type Logger struct {
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

func (l *Logger) Handlers() []slog.Handler {
	return l.handlers
}

func (l *Logger) Level() slog.Level {
	return l.level.Level()
}

// SetLevel changes the level for all Loggers. Even the ones "copied" via any WithX method.
// All groups will have the same level.
func (l *Logger) SetLevel(level slog.Level) {
	*l.level = level
}

func (l *Logger) Enabled(ctx context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo

	if l.level != nil {
		minLevel = l.level.Level()
	}

	return level >= minLevel
}

func (l *Logger) Handle(ctx context.Context, record slog.Record) error {
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

func (l *Logger) WithAttrs(attrs []slog.Attr) slog.Handler { //nolint:ireturn // required for slog.Handler
	handlers := make([]slog.Handler, len(l.handlers))

	for i, h := range l.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}

	return &Logger{
		handlers: handlers,
		level:    l.level,
	}
}

func (l *Logger) WithGroup(name string) slog.Handler { //nolint:ireturn // required for slog.Handler
	handlers := make([]slog.Handler, len(l.handlers))

	for i, h := range l.handlers {
		handlers[i] = h.WithGroup(name)
	}

	return &Logger{
		handlers: handlers,
		level:    l.level,
	}
}

func LogHandlerFromLogger(logger *slog.Logger) *Logger {
	if l, ok := logger.Handler().(*Logger); ok {
		return l
	}

	return nil
}

type (
	LokiHandlerOptions struct {
		PushURL string
		Label   string
	}

	// LokiHandler, see NewLokiHandler.
	LokiHandler struct {
		client   promtail.Client
		renderer slog.Handler
		output   *bytes.Buffer
	}
)

var _ slog.Handler = (*LokiHandler)(nil)

func (l LokiHandler) Handle(ctx context.Context, record slog.Record) error {
	_ = l.renderer.Handle(ctx, record)

	var attrs string

	record.Attrs(func(a slog.Attr) bool {
		// this is high cardinality and can kill loki
		// https://grafana.com/docs/loki/latest/fundamentals/labels/#cardinality
		attrs += fmt.Sprintf(",%s=%s ", a.Key, a.Value.String())

		return true // process next attr
	})

	attrs = strings.TrimPrefix(attrs, ",")
	attrs = strings.TrimSpace(attrs)

	l.client.Infof(record.Message + " " + attrs) // in grafana: green
	l.client.Infof(l.output.String())            // in grafana: green

	l.output.Reset()

	// client.Debugf(record.Message) // in grafana: blue
	// client.Errorf(record.Message) // in grafana: red
	// client.Warnf(record.Message)  // in grafana: yellow

	// !!! Query in grafana with !!!
	// {job="somejob"} | logfmt | command="GetWorkersRequest"
	//
	// https://grafana.com/docs/loki/latest/logql/log_queries/#logfmt

	return nil
}

func (l LokiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (l LokiHandler) WithAttrs(attrs []slog.Attr) slog.Handler { //nolint:ireturn // required for slog.Handler
	return &LokiHandler{
		client:   l.client,
		renderer: l.renderer.WithAttrs(attrs),
		output:   l.output,
	}
}

func (l LokiHandler) WithGroup(name string) slog.Handler { //nolint:ireturn // required for slog.Handler
	return &LokiHandler{
		client:   l.client,
		renderer: l.renderer.WithGroup(name),
		output:   l.output,
	}
}

// NewLokiHandler use this handler only for local development!
//
// Its purpose is to mimic your production setting in case you're using loki & grafana.
// It ships your logs to a local loki instance, so you can use the same setup for development.
// It does not care about performance, as in production you would log to `stdout` and the
// container-runtime's drivers (docker, kubernetes) or something will ship your logs to loki.
func NewLokiHandler(opt *LokiHandlerOptions) *LokiHandler {
	defaultOpt := &LokiHandlerOptions{
		PushURL: "http://localhost:3100/api/prom/push",
		Label:   fmt.Sprintf("{%s=\"%s\"}", "arrower", "skeleton"),
	}

	if opt == nil {
		opt = defaultOpt
	}

	if opt.PushURL != "" {
		opt.PushURL = defaultOpt.PushURL
	}

	if opt.Label != "" {
		opt.Label = defaultOpt.Label
	}

	conf := promtail.ClientConfig{
		PushURL:            opt.PushURL,
		BatchWait:          1 * time.Second,
		BatchEntriesNumber: 1,
		SendLevel:          promtail.DEBUG,
		PrintLevel:         promtail.DISABLE,
		Labels:             opt.Label,
	}

	// Do not handle error here, because promtail method always returns `nil`.
	client, _ := promtail.NewClientJson(conf)

	// generate json log by writing to local buffer with slog default json
	buf := &bytes.Buffer{}
	jsonLog := slog.HandlerOptions{
		Level:       LevelTrace, // allow all messages, as the level gets controlled by the Logger instead.
		AddSource:   false,
		ReplaceAttr: NameArrowerLogLevels,
	}
	renderer := slog.NewJSONHandler(buf, &jsonLog)

	return &LokiHandler{
		client:   client,
		renderer: renderer,
		output:   buf,
	}
}

// getLevelNames maps the arrower log levels to human-readable names.
func getLevelNames() map[slog.Leveler]string {
	return map[slog.Leveler]string{
		LevelDebug: "ARROWER:DEBUG",
		LevelTrace: "ARROWER:TRACE",
	}
}

func NameArrowerLogLevels(groups []string, attr slog.Attr) slog.Attr {
	if attr.Key == slog.LevelKey {
		level, _ := attr.Value.Any().(slog.Level)

		levelLabel, exists := getLevelNames()[level]
		if !exists {
			levelLabel = level.String()
		}

		attr.Value = slog.StringValue(levelLabel)
	}

	return attr
}

func getDefaultHandlerOptions() *slog.HandlerOptions {
	return &slog.HandlerOptions{
		AddSource:   true,
		Level:       nil, // this level is ignored, Logger's level is used for all handlers.
		ReplaceAttr: NameArrowerLogLevels,
	}
}
