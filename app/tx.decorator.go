package app

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-arrower/arrower/postgres"
)

func NewTxRequest[Req any, Res any](pgx *pgxpool.Pool, req Request[Req, Res]) Request[Req, Res] {
	return &requestTxDecorator[Req, Res]{
		pgx:  pgx,
		base: req,
	}
}

type requestTxDecorator[Req any, Res any] struct {
	pgx  *pgxpool.Pool
	base Request[Req, Res]
}

func (d *requestTxDecorator[Req, Res]) H(ctx context.Context, req Req) (Res, error) { //nolint:ireturn,lll // valid use of generics
	tx, err := d.pgx.Begin(ctx)
	if err != nil {
		return *new(Res), fmt.Errorf("could not start transaction: %w", err)
	}

	ctx = context.WithValue(ctx, postgres.CtxTX, tx)

	res, err := d.base.H(ctx, req)
	if err != nil {
		rb := tx.Rollback(ctx)
		if rb != nil {
			return *new(Res), fmt.Errorf("could not rollback transaction: %w", rb)
		}

		return res, err //nolint:wrapcheck // decorate but not change anything
	}

	err = tx.Commit(ctx)
	if err != nil {
		return *new(Res), fmt.Errorf("could not commit transaction: %w", err)
	}

	return res, err //nolint:wrapcheck // decorate but not change anything
}

func NewTxCommand[C any](pgx *pgxpool.Pool, cmd Command[C]) Command[C] {
	return &commandTxDecorator[C]{
		pgx:  pgx,
		base: cmd,
	}
}

type commandTxDecorator[C any] struct {
	pgx  *pgxpool.Pool
	base Command[C]
}

func (d *commandTxDecorator[C]) H(ctx context.Context, cmd C) error {
	tx, err := d.pgx.Begin(ctx)
	if err != nil {
		return fmt.Errorf("could not start transaction: %w", err)
	}

	ctx = context.WithValue(ctx, postgres.CtxTX, tx)

	err = d.base.H(ctx, cmd)
	if err != nil {
		rb := tx.Rollback(ctx)
		if rb != nil {
			return fmt.Errorf("could not rollback transaction: %w", rb)
		}

		return err //nolint:wrapcheck // decorate but not change anything
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("could not commit transaction: %w", err)
	}

	return err //nolint:wrapcheck // decorate but not change anything
}
