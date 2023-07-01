package alog

import (
	"context"

	"golang.org/x/exp/slog"
)

// Logger interface is a subset of slog.Logger, with the aim to:
//  1. encourage the use of the methods offering context.Context, so that tracing information can be correlated, see:
//     https://www.arrower.org/docs/basics/observability/logging#correlate-with-tracing
//  2. encourage the use of the levels `DEBUG` and `INFO` over others, but without preventing them, see:
//     https://dave.cheney.net/2015/11/05/lets-talk-about-logging
type Logger interface {
	Log(ctx context.Context, level slog.Level, msg string, args ...any)
	LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr)
	DebugCtx(ctx context.Context, msg string, args ...any)
	InfoCtx(ctx context.Context, msg string, args ...any)
}

const (
	// LevelInfo is used to see what is going on inside arrower.
	LevelInfo = slog.Level(-8)

	// LevelDebug is used by arrower developers, if you really want to know what is going on.
	LevelDebug = slog.Level(-12)
)

// NameLogLevels replaces the default name of a custom log level with an speaking name for the arrower levels.
func NameLogLevels(groups []string, attr slog.Attr) slog.Attr {
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
