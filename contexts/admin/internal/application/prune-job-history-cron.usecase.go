package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/app"
	"github.com/go-arrower/arrower/contexts/admin/internal/interfaces/repository/models"
	"github.com/go-arrower/arrower/setting"
)

var ErrPruneJobHistoryCronFailed = errors.New("prune job history cron failed")

var SettingAutomaticallyPruneJobHistory = setting.NewKey("admin", "jobs", "automatic_prune_settings")

func NewPruneJobHistoryCronCommandHandler(
	logger alog.Logger,
	settings setting.Settings,
	queries *models.Queries,
) app.Command[PruneJobHistoryCronCommand] {
	return &pruneJobHistoryCronCommandHandler{
		logger:   logger,
		settings: settings,
		queries:  queries,
	}
}

type pruneJobHistoryCronCommandHandler struct {
	logger   alog.Logger
	settings setting.Settings
	queries  *models.Queries
}

type (
	PruneJobHistoryCronCommand struct{}

	// PruneJobHistoryCronSetting belongs to SettingAutomaticallyPruneJobHistory.
	// Annotations to keep it simple:
	// - json for persistence
	// - form, validate for HTML UI.
	PruneJobHistoryCronSetting struct {
		PruneHistoryEntries      bool `form:"prune_history_entries"       json:"pruneHistoryEntries"`
		PruneHistoryEntriesValue int  `form:"prune_history_entries_value" json:"pruneHistoryEntriesValue" validate:"min=1"`

		PruneHistoryAge      bool `form:"prune_history_age"       json:"pruneHistoryAge"`
		PruneHistoryAgeValue int  `form:"prune_history_age_value" json:"pruneHistoryAgeValue" validate:"min=1"`

		PruneHistorySize      bool `form:"prune_history_size"       json:"pruneHistorySize"`
		PruneHistorySizeValue int  `form:"prune_history_size_value" json:"pruneHistorySizeValue" validate:"min=1"`
	}
)

func (h *pruneJobHistoryCronCommandHandler) H(ctx context.Context, _ PruneJobHistoryCronCommand) error {
	val, err := h.settings.Setting(ctx, SettingAutomaticallyPruneJobHistory)
	if err != nil {
		return fmt.Errorf("could not get settings: %w", err)
	}

	pruneSetting := PruneJobHistoryCronSetting{}

	err = val.Unmarshal(&pruneSetting)
	if err != nil {
		return fmt.Errorf("could not unmarshal settings: %w", err)
	}

	h.logger.InfoContext(ctx, "prune job history", slog.Any("setting", pruneSetting))

	if pruneSetting.PruneHistoryEntries {
		err = errors.Join(err, h.queries.TruncateHistoryEntries(ctx, int32(pruneSetting.PruneHistoryEntriesValue)))
	}

	if pruneSetting.PruneHistoryAge {
		const timeDay = time.Hour * 24

		deleteBefore := time.Now().Add(-1 * time.Duration(pruneSetting.PruneHistoryAgeValue) * timeDay)

		err = errors.Join(err, h.queries.PruneHistory(
			ctx,
			pgtype.Timestamptz{Time: deleteBefore, Valid: true, InfinityModifier: pgtype.Finite},
		))
	}

	if pruneSetting.PruneHistorySize {
		err = errors.Join(err, h.queries.TruncateHistorySize(ctx, int32(pruneSetting.PruneHistorySizeValue*1024*1024*1024)))
	}

	if err != nil {
		return fmt.Errorf("%w: %w", ErrPruneJobHistoryCronFailed, err)
	}

	return nil
}
