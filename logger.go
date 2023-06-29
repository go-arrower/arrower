package arrower

import (
	"bytes"
	"context"
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

// Ensure ArrowerLogger implements slog.Handler.
var _ slog.Handler = (*ArrowerLogger)(nil)

// ArrowerLogger is the main handler of arrower, offering to log to multiple handlers,
// filtering of context & users.
// It's also doing all the lifting for observability, see #adl.
type ArrowerLogger struct {
	handlers []slog.Handler

	// level reports the minimum record level that will be logged.
	// The handler discards records with lower levels.
	// If level is nil, the handler assumes LevelInfo.
	// The handler calls level.level for each record processed;
	// to adjust the minimum level dynamically, use a LevelVar.
	// This IS the level for all handlers. The level of the handlers set via
	// WithHandler are ignored.
	level *slog.Level
}

func (l *ArrowerLogger) Handlers() []slog.Handler {
	return l.handlers
}

func (l *ArrowerLogger) Level() slog.Level {
	return l.level.Level()
}

// SetLevel changes the level for all Loggers. Even the ones "copied" via any WithX method.
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
	for _, h := range l.handlers {
		_ = h.Handle(ctx, record)
	}

	return nil
}

func (l *ArrowerLogger) WithAttrs(attrs []slog.Attr) slog.Handler { //nolint:ireturn // required to implement slog.Handler
	handlers := make([]slog.Handler, len(l.handlers))

	for i, h := range l.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}

	return &ArrowerLogger{
		handlers: handlers,
		level:    l.level,
	}
}

func (l *ArrowerLogger) WithGroup(name string) slog.Handler { //nolint:ireturn // required to implement slog.Handler
	handlers := make([]slog.Handler, len(l.handlers))

	for i, h := range l.handlers {
		handlers[i] = h.WithGroup(name)
	}

	return &ArrowerLogger{
		handlers: handlers,
		level:    l.level,
	}
}

type LoggerOpt func(logger *ArrowerLogger)

func WithHandler(h slog.Handler) LoggerOpt {
	return func(l *ArrowerLogger) {
		l.handlers = append(l.handlers, h)
	}
}

func WithLevel(level slog.Level) LoggerOpt {
	return func(l *ArrowerLogger) {
		l.level = &level
	}
}

func NewLogHandler(opts ...LoggerOpt) *ArrowerLogger {
	var (
		defaultLevel    = slog.LevelInfo
		defaultHandlers = []slog.Handler{
			slog.NewJSONHandler(
				os.Stderr,
				&slog.HandlerOptions{
					AddSource:   false,
					Level:       defaultLevel,
					ReplaceAttr: nil, // todo create an assertion for this: NameArrowerLogLevels,
				},
			),
		}
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

func NewLogger(opts ...LoggerOpt) *slog.Logger {
	return slog.New(NewLogHandler(opts...))
}

func LogHandlerFromLogger(logger *slog.Logger) *ArrowerLogger {
	if l, ok := logger.Handler().(*ArrowerLogger); ok {
		return l
	}

	return &ArrowerLogger{}
}

// --- --- --- --- --- ---

func NewFilteredLogger(w io.Writer) *Handler {
	level := slog.LevelDebug

	opts := &slog.HandlerOptions{
		Level:       &level,
		AddSource:   false,
		ReplaceAttr: NameArrowerLogLevels,
	}

	filteredLogger := &filteredLogger{
		orig:          slog.NewTextHandler(w, opts),
		observedUsers: map[string]struct{}{},
	}

	return &Handler{
		Logger:       slog.New(filteredLogger),
		filter:       filteredLogger,
		arrowerLevel: &level,
	}
}

type Handler struct {
	Logger *slog.Logger
	filter *filteredLogger

	// arrowerLevel controls the level of the logger globally. Info by default.
	arrowerLevel *slog.Level
}

// SetLogLevel allows to set the global level for the loggers.
func (f *Handler) SetLogLevel(l slog.Level) {
	*f.arrowerLevel = l
}

func (f *Handler) SetUserFilter(u string) {
	f.filter.observedUsers[u] = struct{}{}
}

func (f *Handler) RemoveUserFilter(u string) {
	delete(f.filter.observedUsers, u)
}

type filteredLogger struct {
	orig          *slog.TextHandler
	observedUsers map[string]struct{}
}

func (f *filteredLogger) Enabled(ctx context.Context, level slog.Level) bool {
	enabled := f.orig.Enabled(ctx, level)

	if enabled || len(f.observedUsers) > 0 {
		return true
	}

	return false
}

func (f *filteredLogger) Handle(ctx context.Context, record slog.Record) error {
	if f.orig.Enabled(ctx, record.Level) {
		label := fmt.Sprintf("{%s=\"%s\"}", "arrower", "skeleton")

		span := trace.SpanFromContext(ctx)
		if !span.IsRecording() {
			fmt.Println("NOT RECORDING")
			//	return s.h.Handle(r)
		}

		{ // (a) adds TraceIds & spanIds to logs.
			//
			// TODO: (komuw) add stackTraces maybe.
			//
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
		}

		{ // (b) adds logs to the active span as events.

			// code from: https://github.com/uptrace/opentelemetry-go-extra/tree/main/otellogrus
			// which is BSD 2-Clause license.

			attrs := make([]attribute.KeyValue, 0)

			logSeverityKey := attribute.Key("log.severity")
			logMessageKey := attribute.Key("log.message")
			attrs = append(attrs, logSeverityKey.String(record.Level.String()))
			attrs = append(attrs, logMessageKey.String(record.Message))

			// TODO: Obey the following rules form the slog documentation:
			//
			// Handle methods that produce output should observe the following rules:
			//   - If r.Time is the zero time, ignore the time.
			//   - If an Attr's key is the empty string, ignore the Attr.
			//
			record.Attrs(func(a slog.Attr) bool {
				attrs = append(attrs,
					attribute.KeyValue{
						Key:   attribute.Key(a.Key),
						Value: attribute.StringValue(a.Value.String()),
					},
				)

				return true //process next attr
			})
			// todo: add caller info.

			span.AddEvent("log", trace.WithAttributes(attrs...))
			if record.Level >= slog.LevelError {
				span.SetStatus(codes.Error, record.Message)
			}
		}

		attrs := ""
		record.Attrs(func(a slog.Attr) bool {
			// this is high cardinality and can kill loki
			// https://grafana.com/docs/loki/latest/fundamentals/labels/#cardinality
			attrs += fmt.Sprintf(",%s=%s ", a.Key, a.Value.String())

			return true // process next attr
		})
		attrs = strings.TrimPrefix(attrs, ",")
		attrs = strings.TrimSpace(attrs)

		conf := promtail.ClientConfig{
			PushURL:            "http://localhost:3100/api/prom/push",
			BatchWait:          1 * time.Second,
			BatchEntriesNumber: 1,
			SendLevel:          promtail.DEBUG,
			PrintLevel:         promtail.DISABLE,
			//Labels:             "{job=\"somejob\"}",
			Labels: label,
		}

		// Do not handle error here, because promtail method always returns `nil`.
		client, _ := promtail.NewClientJson(conf)

		// generate json log by writing to local buffer with slog default json
		buf := &bytes.Buffer{}
		jsonLog := slog.HandlerOptions{
			Level:       &record.Level,
			AddSource:   false,
			ReplaceAttr: NameArrowerLogLevels,
		}
		_ = slog.NewJSONHandler(buf, &jsonLog).Handle(ctx, record)

		client.Infof(record.Message + " " + attrs) // in grafana: green
		client.Infof(buf.String())                 // in grafana: green
		fmt.Println(buf.String())

		//client.Debugf(record.Message) // in grafana: blue
		//client.Errorf(record.Message) // in grafana: red
		//client.Warnf(record.Message)  // in grafana: yellow

		// !!! Query in grafana with !!!
		// {job="somejob"} | logfmt | command="GetWorkersRequest"
		//
		// https://grafana.com/docs/loki/latest/logql/log_queries/#logfmt

		/*
			TODO TRY THE FOLLOWING ALTERNATIVE TO SEE IF THIS LOGGER CAn be not required and not implemented at all
			PROBABLY WORKS FOR OTHER SERVICES IN DOCKER-COMPOSE, BUT NOT FOR THIS APP
			    logging:
			      driver: loki
			      options:
			        loki-url: 'http://localhost:3100/api/prom/push'
		*/

		return f.orig.Handle(ctx, record) //nolint:wrapcheck // slog.Logger is exposed and used by the app developer.
	}

	var hasUserID bool

	record.Attrs(func(attr slog.Attr) bool {
		if attr.Key == "user_id" {
			if _, ok := f.observedUsers[attr.Value.String()]; ok {
				hasUserID = true

				return false // stop iteration
			}
		}

		return true // continue iteration
	})

	if hasUserID {
		return f.orig.Handle(ctx, record) //nolint:wrapcheck // slog.Logger is exposed and used by the app developer.
	}

	return nil
}

func (f *filteredLogger) WithAttrs(attrs []slog.Attr) slog.Handler { //nolint:ireturn // is required to implement slog.
	return f.orig.WithAttrs(attrs)
}

func (f *filteredLogger) WithGroup(name string) slog.Handler { //nolint:ireturn // is required to implement slog.
	return f.orig.WithGroup(name)
}

//nolint:gochecknoglobals // recommended by slog documentation. // todo make it a function to prevent global variable
var (
	// levelNames maps the arrower log levels to human-readable names.
	levelNames = map[slog.Leveler]string{
		LevelDebug: "ARROWER:DEBUG",
		LevelTrace: "ARROWER:TRACE",
	}
)

func NameArrowerLogLevels(groups []string, attr slog.Attr) slog.Attr {
	if attr.Key == slog.LevelKey {
		level, _ := attr.Value.Any().(slog.Level)

		levelLabel, exists := levelNames[level]
		if !exists {
			levelLabel = level.String()
		}

		attr.Value = slog.StringValue(levelLabel)
	}

	return attr
}
