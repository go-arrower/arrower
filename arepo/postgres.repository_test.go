//go:build integration

package arepo_test

import (
	"os"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/arepo"
	"github.com/go-arrower/arrower/arepo/testdata"
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

	arepo.TestSuite(t,
		func(opts ...arepo.Option) arepo.Repository[testdata.Entity, testdata.EntityID] {
			pgx := pgHandler.NewTestDatabase()
			initTestSchema(t, pgx)

			repo, err := arepo.NewPostgresRepository[testdata.Entity, testdata.EntityID](pgx, opts...)
			if err != nil {
				panic(err) // the TestSuite expects a panic in case of a failing constructor
			}

			return repo
		},
		func(opts ...arepo.Option) arepo.Repository[testdata.EntityWithNamePK, string] {
			pgx := pgHandler.NewTestDatabase()
			initTestSchema(t, pgx)

			repo, err := arepo.NewPostgresRepository[testdata.EntityWithNamePK, string](pgx, opts...)
			if err != nil {
				panic(err) // the TestSuite expects a panic in case of a failing constructor
			}

			return repo
		},
		func(opts ...arepo.Option) arepo.Repository[testdata.EntityWithIntPK, testdata.EntityIDInt] {
			pgx := pgHandler.NewTestDatabase()
			initTestSchema(t, pgx)

			repo, err := arepo.NewPostgresRepository[testdata.EntityWithIntPK, testdata.EntityIDInt](pgx, opts...)
			if err != nil {
				panic(err) // the TestSuite expects a panic in case of a failing constructor
			}

			return repo
		},
	)
}

// todo test with and without tx

func TestPostgresRepositoryWithIDField(t *testing.T) {
	t.Parallel()

	// TODO: what value does this TC add?
	// TODO: is it different from "ID field overwrite" from the TestSuite?

	pgx := pgHandler.NewTestDatabase()
	initTestSchema(t, pgx)

	repo, err := arepo.NewPostgresRepository[testdata.EntityWithoutID, string](pgx,
		arepo.WithIDField("Name"),
	)
	assert.NoError(t, err)

	name := gofakeit.Name()

	err = repo.Create(ctx, testdata.EntityWithoutID{Name: name})
	assert.NoError(t, err)
	_, err = repo.Read(ctx, name)
	assert.NoError(t, err)
}

func TestPostgresRepository_Create(t *testing.T) {
	t.Parallel()

	pgx := pgHandler.NewTestDatabase()
	initTestSchema(t, pgx)

	repo, err := arepo.NewPostgresRepository[testdata.Entity, testdata.EntityID](pgx)
	assert.NotNil(t, repo)
	assert.NoError(t, err)

	err = repo.Create(ctx, testdata.DefaultEntity)
	assert.NoError(t, err)

	got, err := repo.Read(ctx, testdata.DefaultEntity.ID)
	assert.NoError(t, err)
	assert.Equal(t, testdata.DefaultEntity, got)
}

func TestPostgresRepository_DbTags(t *testing.T) {
	t.Parallel()

	t.Run("db tags", func(t *testing.T) {
		t.Parallel()

		type MyType struct {
			ID     string `db:"id"`
			MyName string `db:"name"`
		}

		pgx := pgHandler.NewTestDatabase()
		initTestSchema(t, pgx)

		repo, err := arepo.NewPostgresRepository[MyType, string](pgx)
		repo.Table = "entity"
		assert.NotNil(t, repo)
		assert.NoError(t, err)

		id := "my-pk"
		myType := MyType{ID: id, MyName: gofakeit.Name()}

		err = repo.Create(ctx, myType)
		assert.NoError(t, err)

		entity, err := repo.FindByID(ctx, id)
		assert.NoError(t, err)
		assert.Equal(t, myType, entity)

		entity.MyName = gofakeit.Name()
		err = repo.Update(ctx, entity)
		assert.NoError(t, err)

		entity.MyName = gofakeit.Name()
		err = repo.Save(ctx, entity)
		assert.NoError(t, err)

		err = repo.AddAll(ctx, []MyType{
			{ID: gofakeit.UUID(), MyName: gofakeit.Name()},
			{ID: gofakeit.UUID(), MyName: gofakeit.Name()},
		})
		assert.NoError(t, err)

		all, err := repo.FindAll(ctx)
		assert.NoError(t, err)
		assert.Len(t, all, 3)
	})

	t.Run("escaped tags", func(t *testing.T) {
		t.Parallel()

		type NestedStruct struct {
			ID     string `db:"id"`
			MyName string `db:"custom.name"`
		}

		pgx := pgHandler.NewTestDatabase()
		initTestSchema(t, pgx)

		repo, err := arepo.NewPostgresRepository[NestedStruct, string](pgx)
		assert.NotNil(t, repo)
		assert.NoError(t, err)

		id := "my-pk"
		myType := NestedStruct{ID: id, MyName: gofakeit.Name()}

		err = repo.Create(ctx, myType)
		assert.NoError(t, err)
	})

	t.Run("nested tags", func(t *testing.T) {
		t.Parallel()

		type Name struct {
			Name string `db:"name"`
		}
		type NestedStruct struct {
			ID     string `db:"id"`
			MyName Name   `db:"custom"`
		}

		pgx := pgHandler.NewTestDatabase()
		initTestSchema(t, pgx)

		repo, err := arepo.NewPostgresRepository[NestedStruct, string](pgx)
		assert.NotNil(t, repo)
		assert.NoError(t, err)

		id := "my-pk"
		myType := NestedStruct{ID: id, MyName: Name{Name: gofakeit.Name()}}

		err = repo.Create(ctx, myType)
		assert.NoError(t, err)

		//
		myType.MyName.Name = gofakeit.Name()
		err = repo.Update(ctx, myType)
		assert.NoError(t, err)
	})

	t.Run("embedded tags", func(t *testing.T) {
		t.Parallel()

		type Name struct {
			Name string `db:"name"`
		}
		type NestedStruct struct {
			ID   string `db:"id"`
			Name `db:"custom"`
		}

		pgx := pgHandler.NewTestDatabase()
		initTestSchema(t, pgx)

		repo, err := arepo.NewPostgresRepository[NestedStruct, string](pgx)
		assert.NotNil(t, repo)
		assert.NoError(t, err)

		id := "my-pk"
		myType := NestedStruct{ID: id, Name: Name{Name: gofakeit.Name()}}

		err = repo.Create(ctx, myType)
		assert.NoError(t, err)
	})
}

func initTestSchema(t *testing.T, pgx *pgxpool.Pool) {
	t.Helper()

	_, err := pgx.Exec(ctx, `CREATE TABLE IF NOT EXISTS entity(id TEXT PRIMARY KEY, name TEXT);`)
	assert.NoError(t, err)

	_, err = pgx.Exec(ctx, `CREATE TABLE IF NOT EXISTS entitywithnamepk(name TEXT PRIMARY KEY, description TEXT);`)
	assert.NoError(t, err)

	_, err = pgx.Exec(ctx, `CREATE TABLE IF NOT EXISTS entitywithoutid(name TEXT PRIMARY KEY);`)
	assert.NoError(t, err)

	_, err = pgx.Exec(ctx, `CREATE TABLE IF NOT EXISTS entitywithintpk(id SERIAL PRIMARY KEY, uint_id INTEGER, name TEXT);`)
	assert.NoError(t, err)

	_, err = pgx.Exec(ctx, `CREATE TABLE IF NOT EXISTS nestedstruct(id TEXT PRIMARY KEY, "custom.name" TEXT);`)
	assert.NoError(t, err)
}

func TestPostgresRepository_All(t *testing.T) {
	t.Parallel()

	initTestSchema(t, pgHandler.PGx())

	for i := range 100 {
		_ = i
		_, err := pgHandler.PGx().Exec(ctx, `INSERT INTO entity (id, name) VALUES (uuid_generate_v4(),uuid_generate_v4());`)
		assert.NoError(t, err)
	}

	repo, err := arepo.NewPostgresRepository[testdata.Entity, testdata.EntityID](pgHandler.PGx())
	assert.NoError(t, err)

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
}

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
