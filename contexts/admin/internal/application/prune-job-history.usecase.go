package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/go-arrower/arrower/app"
	"github.com/go-arrower/arrower/contexts/admin/internal/interfaces/repository/models"
)

var ErrPruneJobHistoryFailed = errors.New("prune job history failed")

type PruneJobHistoryRequestHandler app.Request[PruneJobHistoryRequest, PruneJobHistoryResponse]

func NewPruneJobHistoryRequestHandler(
	queries *models.Queries,
) app.Request[PruneJobHistoryRequest, PruneJobHistoryResponse] {
	return &pruneJobHistoryRequestHandler{queries: queries}
}

type pruneJobHistoryRequestHandler struct {
	queries *models.Queries
}

type (
	PruneJobHistoryRequest struct {
		Days int
	}

	PruneJobHistoryResponse struct {
		Jobs    string
		History string
	}
)

func (h *pruneJobHistoryRequestHandler) H(
	ctx context.Context,
	req PruneJobHistoryRequest,
) (PruneJobHistoryResponse, error) {
	const timeDay = time.Hour * 24

	deleteBefore := time.Now().Add(-1 * time.Duration(req.Days) * timeDay)

	err := h.queries.PruneHistory(
		ctx,
		pgtype.Timestamptz{Time: deleteBefore, Valid: true, InfinityModifier: pgtype.Finite},
	)
	if err != nil {
		return PruneJobHistoryResponse{}, fmt.Errorf("%w: could not delete old history: %v", ErrPruneJobHistoryFailed, err)
	}

	size, err := h.queries.JobTableSize(ctx)
	if err != nil {
		return PruneJobHistoryResponse{}, fmt.Errorf(
			"%w: could not get new jobs table size: %v",
			ErrPruneJobHistoryFailed,
			err,
		)
	}

	return PruneJobHistoryResponse{
		Jobs:    size.Jobs,
		History: size.History,
	}, nil
}
