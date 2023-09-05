package alog

import (
	"context"

	"golang.org/x/exp/slog"
)

type noopHandler struct{}

func (n noopHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return false
}

func (n noopHandler) Handle(ctx context.Context, record slog.Record) error {
	return nil
}

func (n noopHandler) WithAttrs(attrs []slog.Attr) slog.Handler { //nolint:ireturn
	return n
}

func (n noopHandler) WithGroup(name string) slog.Handler { //nolint:ireturn
	return n
}

var _ slog.Handler = (*noopHandler)(nil)
