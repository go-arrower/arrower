//go:build integration

package tests_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/tests"
)

var pgHandler *tests.PostgresDocker

func TestMain(m *testing.M) {
	pgHandler = tests.NewPostgresDockerForIntegrationTesting()

	//
	// Run tests
	code := m.Run()

	pgHandler.Cleanup()
	os.Exit(code)
}

func TestPostgresDocker_NewTestDatabase(t *testing.T) {
	t.Parallel()

	t.Run("load common file automatically", func(t *testing.T) {
		t.Parallel()

		var pg *pgxpool.Pool

		assert.NotPanics(t, func() {
			pg = pgHandler.NewTestDatabase()
		})

		assertTableNumberOfRows(t, pg, "arrower.log", 1)
		assertTableNumberOfRows(t, pg, "1", 0)
		assertTableNumberOfRows(t, pg, "public.2", 0)
	})

	t.Run("load common file and multiple multi-table fixtures", func(t *testing.T) {
		t.Parallel()

		var pg *pgxpool.Pool

		assert.NotPanics(t, func() {
			pg = pgHandler.NewTestDatabase(
				"testdata/fixtures/test_case0.yaml",
				"testdata/fixtures/test_case1.yaml",
			)
		})

		assertTableNumberOfRows(t, pg, "arrower.log", 1)
		assertTableNumberOfRows(t, pg, "some_table", 2)
		assertTableNumberOfRows(t, pg, "public.other_table", 2)
	})

	t.Run("run multiple tests in parallel", func(t *testing.T) {
		t.Parallel()

		const testNumber = 100

		var wg sync.WaitGroup

		wg.Add(testNumber)
		for i := 0; i < testNumber; i++ { //nolint:wsl_v5
			go func() {
				var pg *pgxpool.Pool

				assert.NotPanics(t, func() {
					pg = pgHandler.NewTestDatabase()
				})

				assertTableNumberOfRows(t, pg, "arrower.log", 1)
				assertTableNumberOfRows(t, pg, "1", 0)
				assertTableNumberOfRows(t, pg, "public.2", 0)

				wg.Done()
			}()
		}

		wg.Wait()
	})
}

func TestPostgresDocker_PrepareDatabase(t *testing.T) {
	t.Parallel()

	t.Run("reset existing database", func(t *testing.T) {
		t.Parallel()

		pg := tests.NewPostgresDockerForIntegrationTesting()

		pg.PrepareDatabase("testdata/fixtures/test_arrower.yaml")
		assertTableNumberOfRows(t, pg.PGx(), "arrower.log", 2)
		assertTableNumberOfRows(t, pg.PGx(), "arrower.gue_jobs", 1)

		pg.PrepareDatabase()
		assertTableNumberOfRows(t, pg.PGx(), "arrower.log", 1)
		assertTableNumberOfRows(t, pg.PGx(), "arrower.gue_jobs", 0)
	})
}

func TestWithMigrations(t *testing.T) {
	// path with migration at the end; arrower does search in a folder that is called "migrations" // TODO
}

//nolint:wsl_v5
func assertTableNumberOfRows(t *testing.T, db *pgxpool.Pool, table string, num int) {
	t.Helper()

	var c int
	_ = db.QueryRow(context.Background(), fmt.Sprintf(`SELECT COUNT(*) FROM %s;`, table)).Scan(&c)

	assert.Equal(t, num, c)
}
