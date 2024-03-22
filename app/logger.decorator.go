package app

import (
	"context"
	"log/slog"

	"github.com/go-arrower/arrower/alog"
)

func NewLoggedRequest[Req any, Res any](logger alog.Logger, handler Request[Req, Res]) Request[Req, Res] {
	return &requestLoggingDecorator[Req, Res]{
		logger: logger,
		base:   handler,
	}
}

type requestLoggingDecorator[Req any, Res any] struct {
	logger alog.Logger
	base   Request[Req, Res]
}

func (d *requestLoggingDecorator[Req, Res]) H(ctx context.Context, req Req) (Res, error) { //nolint:ireturn,lll // valid use of generics
	cmdName := commandName(req)

	d.logger.DebugContext(ctx, "executing request",
		slog.String("command", cmdName),
	)

	res, err := d.base.H(ctx, req)

	if err != nil {
		d.logger.DebugContext(ctx, "failed to execute request",
			slog.String("command", cmdName),
			slog.String("error", err.Error()),
		)
	} else {
		d.logger.DebugContext(ctx, "request executed successfully",
			slog.String("command", cmdName))
	}

	return res, err //nolint:wrapcheck // decorate but not change anything
}

func NewLoggedCommand[C any](logger alog.Logger, handler Command[C]) Command[C] {
	return &commandLoggingDecorator[C]{
		logger: logger,
		base:   handler,
	}
}

type commandLoggingDecorator[C any] struct {
	logger alog.Logger
	base   Command[C]
}

func (d *commandLoggingDecorator[C]) H(ctx context.Context, cmd C) error {
	cmdName := commandName(cmd)

	d.logger.DebugContext(ctx, "executing command",
		slog.String("command", cmdName),
	)

	err := d.base.H(ctx, cmd)

	if err != nil {
		d.logger.DebugContext(ctx, "failed to execute command",
			slog.String("command", cmdName),
			slog.String("error", err.Error()),
		)
	} else {
		d.logger.DebugContext(ctx, "command executed successfully",
			slog.String("command", cmdName))
	}

	return err //nolint:wrapcheck // decorate but not change anything
}

func NewLoggedQuery[Q any, Res any](logger alog.Logger, handler Request[Q, Res]) Request[Q, Res] {
	return &queryLoggingDecorator[Q, Res]{
		logger: logger,
		base:   handler,
	}
}

type queryLoggingDecorator[Q any, Res any] struct {
	logger alog.Logger
	base   Request[Q, Res]
}

func (d *queryLoggingDecorator[Q, Res]) H(ctx context.Context, query Q) (Res, error) { //nolint:ireturn,lll // valid use of generics
	cmdName := commandName(query)

	d.logger.DebugContext(ctx, "executing query",
		slog.String("command", cmdName),
	)

	res, err := d.base.H(ctx, query)

	if err != nil {
		d.logger.DebugContext(ctx, "failed to execute query",
			slog.String("command", cmdName),
			slog.String("error", err.Error()),
		)
	} else {
		d.logger.DebugContext(ctx, "query executed successfully",
			slog.String("command", cmdName))
	}

	return res, err //nolint:wrapcheck // decorate but not change anything
}

func NewLoggedJob[J any](logger alog.Logger, handler Command[J]) Command[J] {
	return &jobLoggingDecorator[J]{
		logger: logger,
		base:   handler,
	}
}

type jobLoggingDecorator[J any] struct {
	logger alog.Logger
	base   Command[J]
}

func (d *jobLoggingDecorator[J]) H(ctx context.Context, job J) error {
	cmdName := commandName(job)

	d.logger.DebugContext(ctx, "executing job",
		slog.String("command", cmdName),
	)

	err := d.base.H(ctx, job)

	if err != nil {
		d.logger.DebugContext(ctx, "failed to execute job",
			slog.String("command", cmdName),
			slog.String("error", err.Error()),
		)
	} else {
		d.logger.DebugContext(ctx, "job executed successfully",
			slog.String("command", cmdName))
	}

	return err //nolint:wrapcheck // decorate but not change anything
}
