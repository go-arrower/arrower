package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/go-arrower/arrower/jobs/models"
	"github.com/go-arrower/arrower/postgres"
)

var ErrQueryFailed = errors.New("query failed")

// repository manages the data access to the underlying Jobs implementation.
type repository interface {
	RegisterWorkerPool(ctx context.Context, wp workerPool) error
	WorkerPools(ctx context.Context) ([]workerPool, error)
}

type workerPool struct {
	LastSeen time.Time
	ID       string
	Queue    string
	Workers  int
}

type postgresJobsRepository struct {
	db postgres.BaseRepository[*models.Queries]
}

var _ repository = (*postgresJobsRepository)(nil)

// TODO Do not export!
func NewPostgresJobsRepository(queries *models.Queries) *postgresJobsRepository {
	return &postgresJobsRepository{db: postgres.NewPostgresBaseRepository(queries)}
}

func (repo *postgresJobsRepository) RegisterWorkerPool(ctx context.Context, wp workerPool) error {
	err := repo.db.ConnOrTX(ctx).UpsertWorkerToPool(ctx, models.UpsertWorkerToPoolParams{
		ID:        wp.ID,
		Queue:     wp.Queue,
		Workers:   int16(wp.Workers),
		UpdatedAt: pgtype.Timestamptz{Time: wp.LastSeen, Valid: true, InfinityModifier: pgtype.Finite},
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrQueryFailed, err) //nolint:errorlint // prevent err in api
	}

	return nil
}

func (repo *postgresJobsRepository) WorkerPools(ctx context.Context) ([]workerPool, error) {
	w, err := repo.db.ConnOrTX(ctx).GetWorkerPools(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQueryFailed, err) //nolint:errorlint // prevent err in api
	}

	return workersToDomain(w), nil
}

func workersToDomain(w []models.GueJobsWorkerPool) []workerPool {
	workers := make([]workerPool, len(w))

	for i, w := range w {
		workers[i] = workerPool{
			ID:       w.ID,
			Queue:    w.Queue,
			Workers:  int(w.Workers),
			LastSeen: w.UpdatedAt.Time,
		}
	}

	return workers
}
