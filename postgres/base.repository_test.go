//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/postgres"
	"github.com/go-arrower/arrower/postgres/models"
	"github.com/go-arrower/arrower/tests"
)

var pgHandler *tests.PostgresDocker

func TestMain(m *testing.M) {
	pgHandler = tests.GetPostgresDockerForIntegrationTestingInstance()

	//
	// Run tests
	code := m.Run()

	pgHandler.Cleanup()
	os.Exit(code)
}

func TestBaseRepository(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("use base repository connection", func(t *testing.T) {
		t.Parallel()

		repo := postgres.NewPostgresBaseRepository(models.New(pgHandler.PGx()))

		res, err := repo.Conn().GetTrue(ctx)
		assert.NoError(t, err)
		assert.True(t, res)
	})

	t.Run("use base repository connection, b/c tx is missing", func(t *testing.T) {
		t.Parallel()

		repo := postgres.NewPostgresBaseRepository(models.New(pgHandler.PGx()))

		res, err := repo.TxOrConn(ctx).GetTrue(ctx)
		assert.NoError(t, err)
		assert.True(t, res)
	})

	t.Run("panics on base repository b/c of missing transaction", func(t *testing.T) {
		t.Parallel()

		repo := postgres.NewPostgresBaseRepository(models.New(pgHandler.PGx()))

		assert.Nil(t, repo.TX(ctx))
		assert.Panics(t, func() {
			res, err := repo.TX(ctx).GetTrue(ctx)
			_ = err
			_ = res
		})
	})

	t.Run("use base repository transaction", func(t *testing.T) {
		t.Parallel()

		repo := postgres.NewPostgresBaseRepository(models.New(pgHandler.PGx()))
		ctx := context.Background()

		tx, err := pgHandler.PGx().Begin(ctx)
		assert.NoError(t, err)

		ctx = context.WithValue(ctx, postgres.CtxTX, tx)
		res, err := repo.TX(ctx).GetTrue(ctx)
		assert.NoError(t, err)
		assert.True(t, res)

		_ = tx.Rollback(ctx)
	})
}
