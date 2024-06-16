package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-arrower/arrower/app"
	"github.com/go-arrower/arrower/contexts/admin/internal/domain/jobs"
)

var ErrGetQueueFailed = errors.New("get queue failed")

func NewGetQueueQueryHandler(repo jobs.Repository) app.Query[GetQueueQuery, GetQueueResponse] {
	return &getQueueQueryHandler{repo: repo}
}

type getQueueQueryHandler struct {
	repo jobs.Repository
}

type (
	GetQueueQuery struct {
		QueueName jobs.QueueName
	}
	GetQueueResponse struct {
		Jobs []jobs.Job
		Kpis jobs.QueueKPIs
	}
)

func (h *getQueueQueryHandler) H(ctx context.Context, query GetQueueQuery) (GetQueueResponse, error) {
	kpis, err := h.repo.QueueKPIs(ctx, query.QueueName)
	if err != nil {
		return GetQueueResponse{}, fmt.Errorf("%w: could not get queue kpis: %w", ErrGetQueueFailed, err)
	}

	jobs, err := h.repo.PendingJobs(ctx, query.QueueName)
	if err != nil {
		return GetQueueResponse{}, fmt.Errorf("%w: could not get pending jobs: %w", ErrGetQueueFailed, err)
	}

	return GetQueueResponse{
		Jobs: jobs,
		Kpis: kpis,
	}, nil
}
