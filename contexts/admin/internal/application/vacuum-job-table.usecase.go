package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-arrower/arrower/app"
	"github.com/go-arrower/arrower/contexts/admin/internal/interfaces/repository/models"
)

var ErrVacuumJobTableFailed = errors.New("vacuum job table failed")

func NewVacuumJobTableRequestHandler(db *pgxpool.Pool) app.Request[VacuumJobTableRequest, VacuumJobTableResponse] {
	return &vacuumJobTableRequestHandler{
		db:      db,
		queries: models.New(db),
	}
}

type vacuumJobTableRequestHandler struct {
	db      *pgxpool.Pool
	queries *models.Queries
}

type (
	VacuumJobTableRequest struct {
		Table string
	}

	VacuumJobTableResponse struct {
		Jobs    string
		History string
	}
)

func (h *vacuumJobTableRequestHandler) H(
	ctx context.Context,
	req VacuumJobTableRequest,
) (VacuumJobTableResponse, error) {
	if !isValidTable(req.Table) {
		return VacuumJobTableResponse{}, fmt.Errorf("%w: invalid table: %s", ErrVacuumJobTableFailed, req.Table)
	}

	_, err := h.db.Exec(ctx, `VACUUM FULL arrower.`+validTables()[req.Table])
	if err != nil {
		return VacuumJobTableResponse{}, fmt.Errorf("%w for table: %s: %v", ErrVacuumJobTableFailed, req.Table, err)
	}

	size, err := h.queries.JobTableSize(ctx)
	if err != nil {
		return VacuumJobTableResponse{}, fmt.Errorf("%w: could not get new job table size: %v", ErrVacuumJobTableFailed, err)
	}

	return VacuumJobTableResponse{
		Jobs:    size.Jobs,
		History: size.History,
	}, nil
}

func validTables() map[string]string {
	return map[string]string{
		"jobs":    "gue_jobs",
		"history": "gue_jobs_history",
	}
}

func isValidTable(table string) bool {
	var validTable bool

	for k := range validTables() {
		if k == table {
			validTable = true
		}
	}

	return validTable
}
