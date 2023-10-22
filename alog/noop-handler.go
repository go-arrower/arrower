package alog

import (
	"context"
	"log/slog"
)

type noopHandler struct{}

func (n noopHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return false
}

func (n noopHandler) Handle(_ context.Context, _ slog.Record) error {
	return nil
}

func (n noopHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return n
}

func (n noopHandler) WithGroup(_ string) slog.Handler {
	return n
}

var _ slog.Handler = (*noopHandler)(nil)
