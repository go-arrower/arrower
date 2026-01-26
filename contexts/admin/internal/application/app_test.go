//go:build integration

package application_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/tests"
)

var (
	ctx       = context.Background()
	pgHandler *tests.PostgresDocker
)

func TestMain(m *testing.M) {
	pgHandler = tests.GetPostgresDockerForIntegrationTestingInstance()

	//
	// Run tests
	code := m.Run()

	pgHandler.Cleanup()
	os.Exit(code)
}

func assertTableNumberOfRows(t *testing.T, db *pgxpool.Pool, table string, num int) {
	t.Helper()

	var c int
	_ = db.QueryRow(context.Background(), fmt.Sprintf(`SELECT COUNT(*) FROM %s;`, table)).Scan(&c) //nolint:wsl_v5

	assert.Equal(t, num, c)
}
