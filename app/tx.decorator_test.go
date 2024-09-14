//go:build integration

package app_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/app"
	"github.com/go-arrower/arrower/postgres"
	"github.com/go-arrower/arrower/tests"
)

var (
	errUseCaseFailed = errors.New("usecase failed")
	pgHandler        *tests.PostgresDocker
	ctx              = context.Background()
)

func TestMain(m *testing.M) {
	pgHandler = tests.GetPostgresDockerForIntegrationTestingInstance()

	//
	// Run tests
	code := m.Run()

	pgHandler.Cleanup()
	os.Exit(code)
}

func TestRequestTxDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful request", func(t *testing.T) {
		t.Parallel()

		pgHandler := pgHandler.NewTestDatabase()
		handler := app.NewTxRequest(pgHandler, app.TestRequestHandler(func(ctx context.Context, _ request) (response, error) {
			tx, txOk := ctx.Value(postgres.CtxTX).(pgx.Tx)
			assert.True(t, txOk)
			assert.NotEmpty(t, tx)

			_, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS some_table(id SERIAL PRIMARY KEY);`)
			assert.NoError(t, err)

			return response{}, nil
		}))

		_, err := handler.H(ctx, request{})
		assert.NoError(t, err)

		_, err = pgHandler.Query(ctx, `SELECT * FROM some_table;`) //nolint:sqlclosecheck // ignore connection pool exhaustion
		assert.NoError(t, err, "table should exist")
	})

	t.Run("failed request", func(t *testing.T) {
		t.Parallel()

		pgHandler := pgHandler.NewTestDatabase()
		handler := app.NewTxRequest(pgHandler, app.TestRequestHandler(func(ctx context.Context, _ request) (response, error) {
			tx, txOk := ctx.Value(postgres.CtxTX).(pgx.Tx)
			assert.True(t, txOk)
			assert.NotEmpty(t, tx)

			_, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS some_table(id SERIAL PRIMARY KEY);`)
			assert.NoError(t, err)

			return response{}, errUseCaseFailed
		}))

		_, err := handler.H(ctx, request{})
		assert.Error(t, err)

		_, err = pgHandler.Query(ctx, `SELECT * FROM some_table;`) //nolint:sqlclosecheck // ignore connection pool exhaustion
		assert.Error(t, err, "table should not exist, as tx is rolled back")
	})

	t.Run("nested requests - all succeed", func(t *testing.T) {
		t.Parallel()

		pgHandler := pgHandler.NewTestDatabase()
		_, err := pgHandler.Exec(ctx, `CREATE TABLE IF NOT EXISTS some_table(id SERIAL PRIMARY KEY);`)
		assert.NoError(t, err)

		handler := app.NewTxRequest(pgHandler, app.TestRequestHandler(func(ctx context.Context, req request) (response, error) {
			tx, _ := ctx.Value(postgres.CtxTX).(pgx.Tx)

			_, err = tx.Exec(ctx, `INSERT INTO some_table(id) VALUES (DEFAULT);`)
			assert.NoError(t, err)

			innerHandler := app.NewTxRequest(pgHandler, app.TestRequestHandler(func(ctx context.Context, _ request) (response, error) {
				tx, _ := ctx.Value(postgres.CtxTX).(pgx.Tx)

				_, err = tx.Exec(ctx, `INSERT INTO some_table(id) VALUES (DEFAULT);`)
				assert.NoError(t, err)

				return response{}, nil
			}))

			return innerHandler.H(ctx, req)
		}))

		_, err = handler.H(ctx, request{})
		assert.NoError(t, err)

		var ids []int
		err = pgxscan.Select(ctx, pgHandler, &ids, `SELECT * FROM some_table;`)
		assert.NoError(t, err, "table should exist")
		assert.Len(t, ids, 2)
	})

	t.Run("nested request - inner fails", func(t *testing.T) {
		t.Parallel()

		pgHandler := pgHandler.NewTestDatabase()
		_, err := pgHandler.Exec(ctx, `CREATE TABLE IF NOT EXISTS some_table(id SERIAL PRIMARY KEY);`)
		assert.NoError(t, err)

		handler := app.NewTxRequest(pgHandler, app.TestRequestHandler(func(ctx context.Context, req request) (response, error) {
			tx, _ := ctx.Value(postgres.CtxTX).(pgx.Tx)

			_, err = tx.Exec(ctx, `INSERT INTO some_table(id) VALUES (DEFAULT);`)
			assert.NoError(t, err)

			innerHandler := app.NewTxRequest(pgHandler, app.TestRequestHandler(func(ctx context.Context, _ request) (response, error) {
				tx, _ := ctx.Value(postgres.CtxTX).(pgx.Tx)

				_, err = tx.Exec(ctx, `INSERT INTO some_table(id) VALUES (DEFAULT);`)
				assert.NoError(t, err)

				return response{}, errUseCaseFailed
			}))

			_, _ = innerHandler.H(ctx, req)

			return response{}, nil
		}))

		_, err = handler.H(ctx, request{})
		assert.NoError(t, err)

		var ids []int
		err = pgxscan.Select(ctx, pgHandler, &ids, `SELECT * FROM some_table;`)
		assert.NoError(t, err, "table should exist")
		assert.Len(t, ids, 1)
	})
}

func TestCommandTxDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful command", func(t *testing.T) {
		t.Parallel()

		pgHandler := pgHandler.NewTestDatabase()
		handler := app.NewTxCommand(pgHandler, app.TestCommandHandler(func(ctx context.Context, _ command) error {
			tx, txOk := ctx.Value(postgres.CtxTX).(pgx.Tx)
			assert.True(t, txOk)
			assert.NotEmpty(t, tx)

			_, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS some_table(id SERIAL PRIMARY KEY);`)
			assert.NoError(t, err)

			return nil
		}))

		err := handler.H(ctx, command{})
		assert.NoError(t, err)

		_, err = pgHandler.Query(ctx, `SELECT * FROM some_table;`) //nolint:sqlclosecheck // ignore connection pool exhaustion
		assert.NoError(t, err, "table should exist")
	})

	t.Run("failed command", func(t *testing.T) {
		t.Parallel()

		pgHandler := pgHandler.NewTestDatabase()
		handler := app.NewTxCommand(pgHandler, app.TestCommandHandler(func(ctx context.Context, _ command) error {
			tx, txOk := ctx.Value(postgres.CtxTX).(pgx.Tx)
			assert.True(t, txOk)
			assert.NotEmpty(t, tx)

			_, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS some_table(id SERIAL PRIMARY KEY);`)
			assert.NoError(t, err)

			return errUseCaseFailed
		}))

		err := handler.H(ctx, command{})
		assert.Error(t, err)

		_, err = pgHandler.Query(ctx, `SELECT * FROM some_table;`) //nolint:sqlclosecheck // ignore connection pool exhaustion
		assert.Error(t, err, "table should not exist, as tx is rolled back")
	})

	t.Run("nested command - all succeed", func(t *testing.T) {
		t.Parallel()

		pgHandler := pgHandler.NewTestDatabase()
		_, err := pgHandler.Exec(ctx, `CREATE TABLE IF NOT EXISTS some_table(id SERIAL PRIMARY KEY);`)
		assert.NoError(t, err)

		handler := app.NewTxCommand(pgHandler, app.TestCommandHandler(func(ctx context.Context, cmd command) error {
			tx, _ := ctx.Value(postgres.CtxTX).(pgx.Tx)

			_, err = tx.Exec(ctx, `INSERT INTO some_table(id) VALUES (DEFAULT);`)
			assert.NoError(t, err)

			innerHandler := app.NewTxCommand(pgHandler, app.TestCommandHandler(func(ctx context.Context, _ command) error {
				tx, _ := ctx.Value(postgres.CtxTX).(pgx.Tx)

				_, err = tx.Exec(ctx, `INSERT INTO some_table(id) VALUES (DEFAULT);`)
				assert.NoError(t, err)

				return nil
			}))

			return innerHandler.H(ctx, cmd)
		}))

		err = handler.H(ctx, command{})
		assert.NoError(t, err)

		var ids []int
		err = pgxscan.Select(ctx, pgHandler, &ids, `SELECT * FROM some_table;`)
		assert.NoError(t, err, "table should exist")
		assert.Len(t, ids, 2)
	})

	t.Run("nested command - inner fails", func(t *testing.T) {
		t.Parallel()

		pgHandler := pgHandler.NewTestDatabase()
		_, err := pgHandler.Exec(ctx, `CREATE TABLE IF NOT EXISTS some_table(id SERIAL PRIMARY KEY);`)
		assert.NoError(t, err)

		handler := app.NewTxCommand(pgHandler, app.TestCommandHandler(func(ctx context.Context, cmd command) error {
			tx, _ := ctx.Value(postgres.CtxTX).(pgx.Tx)

			_, err = tx.Exec(ctx, `INSERT INTO some_table(id) VALUES (DEFAULT);`)
			assert.NoError(t, err)

			innerHandler := app.NewTxCommand(pgHandler, app.TestCommandHandler(func(ctx context.Context, _ command) error {
				tx, _ := ctx.Value(postgres.CtxTX).(pgx.Tx)

				_, err = tx.Exec(ctx, `INSERT INTO some_table(id) VALUES (DEFAULT);`)
				assert.NoError(t, err)

				return errUseCaseFailed
			}))

			_ = innerHandler.H(ctx, cmd)

			return nil
		}))

		err = handler.H(ctx, command{})
		assert.NoError(t, err)

		var ids []int
		err = pgxscan.Select(ctx, pgHandler, &ids, `SELECT * FROM some_table;`)
		assert.NoError(t, err, "table should exist")
		assert.Len(t, ids, 1)
	})
}
