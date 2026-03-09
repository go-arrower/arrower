//go:build integration

package application_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/contexts/admin/internal/application"
	"github.com/go-arrower/arrower/contexts/admin/internal/interfaces/repository/models"
	"github.com/go-arrower/arrower/setting"
)

func TestPruneJobHistoryCronCommandHandler_H(t *testing.T) {
	t.Parallel()

	t.Run("too many jobs", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase("testdata/fixtures/prune_entries.yaml")
		settings := setting.NewInMemorySettings()
		settings.Save(t.Context(), application.SettingAutomaticallyPruneJobHistory, setting.NewValue(
			application.PruneJobHistoryCronSetting{
				PruneHistoryEntries:      true,
				PruneHistoryEntriesValue: 1,
				PruneHistoryAge:          false,
				PruneHistoryAgeValue:     0,
				PruneHistorySize:         false,
				PruneHistorySizeValue:    0,
			}))
		app := application.NewPruneJobHistoryCronCommandHandler(alog.Test(t), settings, models.New(pg))

		err := app.H(ctx, application.PruneJobHistoryCronCommand{})
		assert.NoError(t, err)

		assertTableNumberOfRows(t, pg, "arrower.gue_jobs_history", 2)
	})

	t.Run("older than n days", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase("testdata/fixtures/prune_history.yaml")
		settings := setting.NewInMemorySettings()
		settings.Save(t.Context(), application.SettingAutomaticallyPruneJobHistory, setting.NewValue(
			application.PruneJobHistoryCronSetting{
				PruneHistoryEntries:      false,
				PruneHistoryEntriesValue: 0,
				PruneHistoryAge:          true,
				PruneHistoryAgeValue:     7,
				PruneHistorySize:         false,
				PruneHistorySizeValue:    0,
			}))
		app := application.NewPruneJobHistoryCronCommandHandler(alog.Test(t), settings, models.New(pg))

		err := app.H(ctx, application.PruneJobHistoryCronCommand{})
		assert.NoError(t, err)

		assertTableNumberOfRows(t, pg, "arrower.gue_jobs_history", 1)
	})

	t.Run("age setting not active", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase("testdata/fixtures/prune_history.yaml")
		settings := setting.NewInMemorySettings()
		settings.Save(t.Context(), application.SettingAutomaticallyPruneJobHistory, setting.NewValue(
			application.PruneJobHistoryCronSetting{
				PruneHistoryEntries:      false,
				PruneHistoryEntriesValue: 0,
				PruneHistoryAge:          false,
				PruneHistoryAgeValue:     7,
				PruneHistorySize:         false,
				PruneHistorySizeValue:    0,
			}))
		app := application.NewPruneJobHistoryCronCommandHandler(alog.Test(t), settings, models.New(pg))

		err := app.H(ctx, application.PruneJobHistoryCronCommand{})
		assert.NoError(t, err)

		assertTableNumberOfRows(t, pg, "arrower.gue_jobs_history", 2)
	})

	t.Run("too large table", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase("testdata/fixtures/prune_entries.yaml")
		settings := setting.NewInMemorySettings()
		settings.Save(t.Context(), application.SettingAutomaticallyPruneJobHistory, setting.NewValue(
			application.PruneJobHistoryCronSetting{
				PruneHistoryEntries:      false,
				PruneHistoryEntriesValue: 0,
				PruneHistoryAge:          false,
				PruneHistoryAgeValue:     0,
				PruneHistorySize:         true,
				PruneHistorySizeValue:    0,
			}))
		app := application.NewPruneJobHistoryCronCommandHandler(alog.Test(t), settings, models.New(pg))

		err := app.H(ctx, application.PruneJobHistoryCronCommand{})
		assert.NoError(t, err)

		assertTableNumberOfRows(t, pg, "arrower.gue_jobs_history", 1)
	})
}
