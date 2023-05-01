package arrower

import (
	"context"
	"io"

	"golang.org/x/exp/slog"
)

const (
	// LevelDebug is used for arrower debug messages.
	LevelDebug = slog.Level(-8)

	// LevelTrace is used for arrower trace information, if you really want to know what is going on.
	LevelTrace = slog.Level(-12)
)

func NewFilteredLogger(w io.Writer) *Handler {
	level := slog.LevelDebug

	opts := slog.HandlerOptions{
		Level:       &level,
		AddSource:   false,
		ReplaceAttr: nameArrowerLogLevels,
	}.NewTextHandler(w)

	filteredLogger := &filteredLogger{
		orig:          opts,
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

//nolint:gochecknoglobals // recommended by slog documentation.
var (
	// levelNames maps the arrower log levels to human-readable names.
	levelNames = map[slog.Leveler]string{
		LevelDebug: "ARROWER:DEBUG",
		LevelTrace: "ARROWER:TRACE",
	}
)

func nameArrowerLogLevels(groups []string, attr slog.Attr) slog.Attr {
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
