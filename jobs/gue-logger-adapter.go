package jobs

import (
	"context"
	"log/slog"

	"github.com/vgarvardt/gue/v5/adapter"

	"github.com/go-arrower/arrower/alog"
)

// gueLogAdapter is a custom adapter.Logger for the gue library.
// Its purpose is to redirect the gue logs to appropriate arrower log levels.
// Gue is offering a slog adapter, but that does not adhere to the
// encouraged behaviour and log levels of arrower, see alog.Logger.
type gueLogAdapter struct {
	l alog.Logger
}

// Debug logs to the `ARROWER:DEBUG` level.
func (l *gueLogAdapter) Debug(msg string, fields ...adapter.Field) {
	l.l.Log(context.Background(), alog.LevelDebug, msg, l.slogFields(fields)...)
}

// Info logs to the `ARROWER:DEBUG` level.
func (l *gueLogAdapter) Info(msg string, fields ...adapter.Field) {
	l.l.Log(context.Background(), alog.LevelDebug, msg, l.slogFields(fields)...)
}

// Error logs to the `ARROWER:INFO` level.
func (l *gueLogAdapter) Error(msg string, fields ...adapter.Field) {
	l.l.Log(context.Background(), alog.LevelInfo, msg, l.slogFields(fields)...)
}

func (l *gueLogAdapter) With(fields ...adapter.Field) adapter.Logger { //nolint:ireturn // required for adapter.Logger
	return &gueLogAdapter{l: l.l.With(l.slogFields(fields)...)}
}

func (l *gueLogAdapter) slogFields(fields []adapter.Field) []any {
	result := make([]any, len(fields))

	for i, f := range fields {
		result[i] = slog.Any(f.Key, f.Value)
	}

	return result
}

var _ adapter.Logger = (*gueLogAdapter)(nil)
