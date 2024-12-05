//go:build integration

package repository_test

import (
	"os"
	"testing"
	"time"

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
			pgx := pgHandler.NewTestDatabase()
			initTestSchema(t, pgx)

			return repository.NewPostgresRepository[testdata.Entity, testdata.EntityID](pgx, opts...)
		},
		func(opts ...repository.Option) repository.Repository[testdata.EntityWithIntPK, testdata.EntityIDInt] {
			pgx := pgHandler.NewTestDatabase()
			initTestSchema(t, pgx)

			return repository.NewPostgresRepository[testdata.EntityWithIntPK, testdata.EntityIDInt](pgx, opts...)
		},
		func(opts ...repository.Option) repository.Repository[testdata.EntityWithoutID, testdata.EntityID] {
			pgx := pgHandler.NewTestDatabase()
			initTestSchema(t, pgx)

			return repository.NewPostgresRepository[testdata.EntityWithoutID, testdata.EntityID](pgx, opts...)
		},
	)
}

// todo test with and without tx

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

	_, err := pgx.Exec(ctx, `CREATE TABLE IF NOT EXISTS entity(id TEXT PRIMARY KEY, name TEXT);`)
	assert.NoError(t, err)

	_, err = pgx.Exec(ctx, `CREATE TABLE IF NOT EXISTS entitywithoutid(name TEXT PRIMARY KEY);`)
	assert.NoError(t, err)

	_, err = pgx.Exec(ctx, `CREATE TABLE IF NOT EXISTS entitywithintpk(id SERIAL PRIMARY KEY, uintid INTEGER, name TEXT);`)
	assert.NoError(t, err)
}

func TestPostgresRepository_All(t *testing.T) {
	initTestSchema(t, pgHandler.PGx())

	for i := range 100 {
		_ = i
		_, err := pgHandler.PGx().Exec(ctx, `INSERT INTO entity (id, name) VALUES (uuid_generate_v4(),uuid_generate_v4());`)
		assert.NoError(t, err)
	}

	repo := repository.NewPostgresRepository[testdata.Entity, testdata.EntityID](pgHandler.PGx())
	iter := repo.AllIter(ctx)
	_ = iter

	start := time.Now()
	// all, _ := repo.All(ctx)
	// for _, e := range all {
	// 	_ = e
	// }
	for e, err := range iter.Next() {
		assert.NoError(t, err)
		_ = e
	}
	t.Log(time.Since(start))

	// 									batch(cursor fetch, channel buffer)
	// 		all(memory)	single			batch(10,2)		batch(100,100)	batch(100,250)	batch(1000,1000)	batch(1000,10.000)
	// 1k 	570.828µs	36.446862ms		4.08922ms
	// 1k 	598.836µs	43.675608ms		5.853376ms
	// 1k 	1.360914ms	35.811862ms     4.473658ms
	// 10k	4.077662ms	488.063487ms 	39.897611ms		12.328828ms		9.206824ms		5.890355ms			5.433562ms
	// 10k 	3.969914ms	338.096554ms	39.821585ms		9.013266ms		9.851891ms		6.038974ms			5.615281ms
	// 10k 	4.026195ms	341.875893ms	43.254681ms		9.933388ms		9.774993ms		5.194709ms			5.210752ms
	// 100k	47.302105ms																	47.697103ms			46.811288ms
	//
	// evaluation of the benchmark:
	// - the laptop was running all kinds of other "regular office" software such as the browser, slack etc.
	// - each run is in a new docker container with random values in the DB, so disc caching is not an issue
	// - the database table is trivial. Does not represent anything real world data
	// - network latency is not considered / tests run all on same machine
}
