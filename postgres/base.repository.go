package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

var ErrQueryFailed = errors.New("query failed")

// BaseRepository can be used in repository implementations.
// Because sqlc generates an own Queries struct for each  corresponding models/db.go file,
// a generic approach is used, so each repo can create a BaseRepository with the fitting *models.Queries type.
type BaseRepository[T interface{ WithTx(tx pgx.Tx) T }] struct {
	queries T
}

func NewPostgresBaseRepository[T interface{ WithTx(tx pgx.Tx) T }](queries T) BaseRepository[T] {
	return BaseRepository[T]{
		queries: queries,
	}
}

// TxOrConn wraps the models.Queries into the transaction in ctx.
// If no transaction is in the context, it falls back to the raw Queries struct.
func (repo BaseRepository[T]) TxOrConn(ctx context.Context) T { //nolint:ireturn,lll // fp, as it is not recognised even with "generic" setting
	if tx, ok := ctx.Value(CtxTX).(pgx.Tx); ok {
		return repo.queries.WithTx(tx)
	}

	// in case no transaction is present return the default DB access.
	return repo.queries
}

// TX wraps the models.Queries into the transaction in ctx.
// If no transaction is present in the given context, it returns nil.
func (repo BaseRepository[T]) TX(ctx context.Context) T { //nolint:ireturn,lll // fp, as it is not recognised even with "generic" setting
	if tx, ok := ctx.Value(CtxTX).(pgx.Tx); ok {
		return repo.queries.WithTx(tx)
	}

	var result T

	return result
}

// Conn returns the models.Queries using the underlying db connection.
func (repo BaseRepository[T]) Conn() T { //nolint:ireturn // fp, as it is not recognised even with "generic" setting
	return repo.queries
}
