//go:build integration

package application_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/contexts/admin/internal/application"
)

func TestVacuumJobTableRequestHandler_H(t *testing.T) {
	t.Parallel()

	passingTests := map[string]struct {
		table string
		err   error
	}{
		"jobs": {
			table: "jobs",
			err:   nil,
		},
		"history": {
			table: "history",
			err:   nil,
		},
	}

	failingTests := map[string]struct {
		table string
		err   error
	}{
		"empty table": {
			table: "",
			err:   application.ErrVacuumJobTableFailed,
		},
		"non existing table": {
			table: "non-existing-table",
			err:   application.ErrVacuumJobTableFailed,
		},
	}

	// share one database for all tests, as it is about vacuum and not modifying data
	app := application.NewVacuumJobTableRequestHandler(pgHandler.PGx())

	for name, tc := range passingTests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			res, err := app.H(context.Background(), application.VacuumJobTableRequest{Table: tc.table})

			assert.ErrorIs(t, err, tc.err)
			assert.NotEmpty(t, res.Jobs)
			assert.NotEmpty(t, res.History)
		})
	}

	for name, tc := range failingTests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			res, err := app.H(context.Background(), application.VacuumJobTableRequest{Table: tc.table})
			assert.ErrorIs(t, err, tc.err)
			assert.Empty(t, res)
		})
	}
}
