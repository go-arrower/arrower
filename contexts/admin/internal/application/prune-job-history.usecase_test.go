//go:build integration

package application_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/contexts/admin/internal/application"
	"github.com/go-arrower/arrower/contexts/admin/internal/interfaces/repository/models"
)

func TestPruneJobHistoryRequestHandler_H(t *testing.T) {
	t.Parallel()

	t.Run("older than n days", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase("testdata/fixtures/prune_history.yaml")
		app := application.NewPruneJobHistoryRequestHandler(models.New(pg))

		res, err := app.H(ctx, application.PruneJobHistoryRequest{Days: 7})
		assert.NoError(t, err)

		assert.NotEmpty(t, res.Jobs)
		assert.NotEmpty(t, res.History)
		assertTableNumberOfRows(t, pg, "arrower.gue_jobs_history", 1)
	})

	t.Run("all", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase("testdata/fixtures/prune_history.yaml")
		app := application.NewPruneJobHistoryRequestHandler(models.New(pg))

		res, err := app.H(ctx, application.PruneJobHistoryRequest{Days: 0})
		assert.NoError(t, err)

		assert.NotEmpty(t, res.Jobs)
		assert.NotEmpty(t, res.History)
		assertTableNumberOfRows(t, pg, "arrower.gue_jobs_history", 0)
	})
}
