package aassert

import (
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func TableNumberOfRows(t *testing.T, db *pgxpool.Pool, table string, num int, msgAndArgs ...any) bool {
	t.Helper()

	var got int

	err := db.QueryRow(t.Context(), fmt.Sprintf(`SELECT COUNT(*) FROM %s;`, table)).Scan(&got)
	if err != nil {
		return assert.Fail(t, "query failed for table "+table+": "+err.Error())
	}

	if got != num {
		return assert.Fail(t, fmt.Sprintf("expected %d rows in table %s, got %d", num, table, got), msgAndArgs...)
	}

	return true
}
