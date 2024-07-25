//go:build integration

package repository_test

import (
	"os"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/repository"
	"github.com/go-arrower/arrower/repository/testdata"
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

func TestPostgresRepository(t *testing.T) {
	t.Parallel()

	repository.TestSuite(t,
		func(opts ...repository.Option) repository.Repository[testdata.Entity, testdata.EntityID] {
			return repository.NewPostgresRepository[testdata.Entity, testdata.EntityID](pgHandler.NewTestDatabase(), opts...)
		},
		func(opts ...repository.Option) repository.Repository[testdata.EntityWithIntPK, testdata.EntityIDInt] {
			return repository.NewPostgresRepository[testdata.EntityWithIntPK, testdata.EntityIDInt](pgHandler.NewTestDatabase(), opts...)
		},
		func(opts ...repository.Option) repository.Repository[testdata.EntityWithoutID, testdata.EntityID] {
			return repository.NewPostgresRepository[testdata.EntityWithoutID, testdata.EntityID](pgHandler.NewTestDatabase(), opts...)
		},
	)

	// if the TestSuite is run again with a store, does this cover all cases and this test files does not have to do many store related tests?
}

func TestPostgresRepositoryWithIDField(t *testing.T) {
	t.Parallel()

	pgx := pgHandler.NewTestDatabase()
	initTestSchema(t, pgx)

	repo := repository.NewPostgresRepository[testdata.EntityWithoutID, string](pgx,
		repository.WithIDField("Name"),
	)

	name := gofakeit.Name()

	err := repo.Create(ctx, testdata.EntityWithoutID{Name: name})
	assert.NoError(t, err)
	_, err = repo.Read(ctx, name)
	assert.NoError(t, err)
}

func TestPostgresRepository_Create(t *testing.T) {
	t.Parallel()

	pgx := pgHandler.NewTestDatabase()
	initTestSchema(t, pgx)

	repo := repository.NewPostgresRepository[testdata.Entity, testdata.EntityID](pgx)
	assert.NotNil(t, repo)

	err := repo.Create(ctx, testdata.DefaultEntity)
	assert.NoError(t, err)

	got, err := repo.Read(ctx, testdata.DefaultEntity.ID)
	assert.NoError(t, err)
	assert.Equal(t, testdata.DefaultEntity, got)
}

func initTestSchema(t *testing.T, pgx *pgxpool.Pool) {
	t.Helper()

	_, err := pgx.Exec(ctx, `CREATE TABLE IF NOT EXISTS entity(id TEXT PRIMARY KEY , name TEXT);`)
	assert.NoError(t, err)

	_, err = pgx.Exec(ctx, `CREATE TABLE IF NOT EXISTS entitywithoutid(name TEXT PRIMARY KEY);`)
	assert.NoError(t, err)
}
