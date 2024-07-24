package alog

import (
	"context"
	"log/slog"

	ctx2 "github.com/go-arrower/arrower/ctx"
)

// Logger interface is a subset of slog.Logger, with the aim to:
//  1. encourage the use of the methods offering context.Context, so that tracing information can be correlated, see:
//     https://www.arrower.org/docs/basics/observability/logging#correlate-with-tracing
//  2. encourage the use of the levels `DEBUG` and `INFO` over others, but without preventing them, see:
//     https://dave.cheney.net/2015/11/05/lets-talk-about-logging
type Logger interface {
	Log(ctx context.Context, level slog.Level, msg string, args ...any)
	LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr)
	DebugContext(ctx context.Context, msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)

	With(args ...any) *slog.Logger
	WithGroup(name string) *slog.Logger
}

const (
	// LevelInfo is used to see what is going on inside arrower.
	LevelInfo = slog.Level(-8)

	// LevelDebug is used by arrower developers, if you really want to know what is going on.
	LevelDebug = slog.Level(-12)
)

// MapLogLevelsToName replaces the default name of a custom log level with an speaking name for the arrower levels.
func MapLogLevelsToName(_ []string, attr slog.Attr) slog.Attr {
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

// getLevelNames maps the arrower log levels to human-readable names.
func getLevelNames() map[slog.Leveler]string {
	return map[slog.Leveler]string{
		LevelInfo:  "ARROWER:INFO",
		LevelDebug: "ARROWER:DEBUG",
	}
}

// CtxAttr contains request scoped attributes.
const CtxAttr ctx2.CTXKey = "arrower.slog"

// AddAttr adds a single attribute to ctx. All attrs in CtxAttr will be logged automatically by the arrowerHandler.
func AddAttr(ctx context.Context, attr slog.Attr) context.Context {
	if attrs, ok := FromContext(ctx); ok {
		return context.WithValue(ctx, CtxAttr, append(attrs, attr))
	}

	return context.WithValue(ctx, CtxAttr, []slog.Attr{attr})
}

// AddAttrs adds multiple attributes to ctx. All attrs in CtxAttr will be logged automatically by the arrowerHandler.
func AddAttrs(ctx context.Context, newAttrs ...slog.Attr) context.Context {
	if attrs, ok := FromContext(ctx); ok {
		return context.WithValue(ctx, CtxAttr, append(attrs, newAttrs...))
	}

	return context.WithValue(ctx, CtxAttr, newAttrs)
}

// ResetAttrs does remove all attributes from CtxAttr.
func ResetAttrs(ctx context.Context) context.Context {
	return context.WithValue(ctx, CtxAttr, nil)
}

// FromContext returns the attributes stored in ctx, if any.
func FromContext(ctx context.Context) ([]slog.Attr, bool) {
	attrs, ok := ctx.Value(CtxAttr).([]slog.Attr)

	return attrs, ok
}
