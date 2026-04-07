//go:build integration

package tests_test

import (
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/aassert"
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

		aassert.TableNumberOfRows(t, pg, "arrower.log", 1)
		aassert.TableNumberOfRows(t, pg, "public.some_table", 0, "should not have rows of other fixtures")
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

		aassert.TableNumberOfRows(t, pg, "arrower.log", 1)
		aassert.TableNumberOfRows(t, pg, "some_table", 2)
		aassert.TableNumberOfRows(t, pg, "public.other_table", 2)
	})

	t.Run("run against multiple databases in parallel", func(t *testing.T) {
		t.Parallel()

		const testNumber = 100

		var wg sync.WaitGroup

		wg.Add(testNumber)
		for range testNumber { //nolint:wsl_v5
			go func() {
				var pg *pgxpool.Pool

				assert.NotPanics(t, func() {
					pg = pgHandler.NewTestDatabase()
				})

				aassert.TableNumberOfRows(t, pg, "arrower.log", 1)

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
		aassert.TableNumberOfRows(t, pg.PGx(), "arrower.log", 2)
		aassert.TableNumberOfRows(t, pg.PGx(), "arrower.gue_jobs", 1)

		pg.PrepareDatabase()
		aassert.TableNumberOfRows(t, pg.PGx(), "arrower.log", 1)
		aassert.TableNumberOfRows(t, pg.PGx(), "arrower.gue_jobs", 0)
	})
}

func TestWithMigrations(t *testing.T) {
	// path with migration at the end; arrower does search in a folder that is called "migrations" // TODO
}
