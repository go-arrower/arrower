package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-arrower/arrower/app"
	"github.com/go-arrower/arrower/contexts/admin/internal/domain/jobs"
)

var ErrGetWorkersFailed = errors.New("get workers failed")

func NewGetWorkersQueryHandler(repo jobs.Repository) app.Query[GetWorkersQuery, GetWorkersResponse] {
	return &getWorkersQueryHandler{repo: repo}
}

type getWorkersQueryHandler struct {
	repo jobs.Repository
}

type (
	GetWorkersQuery    struct{}
	GetWorkersResponse struct {
		Pool      []jobs.WorkerPool
		Schedules []jobs.Schedule
	}
)

func (h *getWorkersQueryHandler) H(ctx context.Context, _ GetWorkersQuery) (GetWorkersResponse, error) {
	wp, err := h.repo.WorkerPools(ctx)
	if err != nil {
		return GetWorkersResponse{}, fmt.Errorf("%w: %w", ErrGetWorkersFailed, err)
	}

	s, err := h.repo.Schdules(ctx)
	if err != nil {
		return GetWorkersResponse{}, fmt.Errorf("%w: %w", ErrGetWorkersFailed, err)
	}

	return GetWorkersResponse{
		Pool:      wp,
		Schedules: s,
	}, nil
}
