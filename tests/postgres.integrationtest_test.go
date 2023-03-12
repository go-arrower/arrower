//go:build integration

package tests_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/postgres"
	"github.com/go-arrower/arrower/tests"
)

var pgHandler *postgres.Handler

func TestMain(m *testing.M) {
	handler, cleanup := tests.GetDBConnectionForIntegrationTesting(context.Background())
	pgHandler = handler

	//
	// Run tests
	code := m.Run()

	//
	// Cleanup
	_ = handler.Shutdown(context.Background())
	_ = cleanup()

	os.Exit(code)
}

func TestPrepareTestDatabase(t *testing.T) {
	t.Parallel()

	t.Run("load common file automatically", func(t *testing.T) {
		t.Parallel()

		var pg *postgres.Handler
		assert.NotPanics(t, func() {
			pg = tests.PrepareTestDatabase(pgHandler)
		})

		assertTableNumberOfRows(t, pg.DB, "admin.setting", 1)
		assertTableNumberOfRows(t, pg.DB, "1", 0)
		assertTableNumberOfRows(t, pg.DB, "public.2", 0)
	})

	t.Run("load common file and multiple multi-table fixtures", func(t *testing.T) {
		t.Parallel()

		var pg *postgres.Handler
		assert.NotPanics(t, func() {
			pg = tests.PrepareTestDatabase(pgHandler,
				"testdata/fixtures/test_case0.yml",
				"testdata/fixtures/test_case1.yml",
			)
		})

		assertTableNumberOfRows(t, pg.DB, "admin.setting", 1)
		assertTableNumberOfRows(t, pg.DB, "some_table", 2)
		assertTableNumberOfRows(t, pg.DB, "public.other_table", 2)
	})

	t.Run("run multiple tests in parallel", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup
		const testNumber = 100

		wg.Add(testNumber)
		for i := 0; i < testNumber; i++ {
			go func() {
				var pg *postgres.Handler
				assert.NotPanics(t, func() {
					pg = tests.PrepareTestDatabase(pgHandler)
				})

				assertTableNumberOfRows(t, pg.DB, "admin.setting", 1)
				assertTableNumberOfRows(t, pg.DB, "1", 0)
				assertTableNumberOfRows(t, pg.DB, "public.2", 0)

				wg.Done()
			}()
		}

		wg.Wait()
	})
}

func assertTableNumberOfRows(t *testing.T, db *sql.DB, table string, num int) {
	t.Helper()

	var c int
	_ = db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s;`, table)).Scan(&c)

	assert.Equal(t, num, c)
}
