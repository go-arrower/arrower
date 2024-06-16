package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-arrower/arrower/app"
	"github.com/go-arrower/arrower/contexts/admin/internal/domain/jobs"
)

var ErrListAllQueuesFailed = errors.New("list all queues failed")

func NewListAllQueuesQueryHandler(repo jobs.Repository) app.Query[ListAllQueuesQuery, ListAllQueuesResponse] {
	return &listAllQueuesQueryHandler{repo: repo}
}

type listAllQueuesQueryHandler struct {
	repo jobs.Repository
}

type (
	ListAllQueuesQuery struct{}

	ListAllQueuesResponse struct {
		QueueStats map[jobs.QueueName]jobs.QueueStats
	}
)

func (h *listAllQueuesQueryHandler) H(ctx context.Context, _ ListAllQueuesQuery) (ListAllQueuesResponse, error) {
	queues, err := h.repo.Queues(ctx)
	if err != nil {
		return ListAllQueuesResponse{}, fmt.Errorf("%w: could not get queues: %w", ErrListAllQueuesFailed, err)
	}

	qWithStats := make(map[jobs.QueueName]jobs.QueueStats)

	for _, queue := range queues {
		s, err := h.repo.QueueKPIs(ctx, queue)
		if err != nil {
			return ListAllQueuesResponse{},
				fmt.Errorf("%w: could not get kpis for queue: %s: %w", ErrListAllQueuesFailed, queue, err)
		}

		qWithStats[queue] = queueKpiToStats(string(queue), s)
	}

	return ListAllQueuesResponse{QueueStats: qWithStats}, nil
}

func queueKpiToStats(queue string, kpis jobs.QueueKPIs) jobs.QueueStats {
	var errorRate float64

	if kpis.FailedJobs != 0 {
		errorRate = float64(kpis.FailedJobs * 100 / kpis.PendingJobs)
	}

	var duration time.Duration
	if kpis.AvailableWorkers != 0 {
		duration = time.Duration(kpis.PendingJobs/kpis.AvailableWorkers) * kpis.AverageTimePerJob
	}

	return jobs.QueueStats{
		QueueName:            jobs.QueueName(queue),
		PendingJobs:          kpis.PendingJobs,
		PendingJobsPerType:   kpis.PendingJobsPerType,
		FailedJobs:           kpis.FailedJobs,
		ProcessedJobs:        kpis.ProcessedJobs,
		AvailableWorkers:     kpis.AvailableWorkers,
		PendingJobsErrorRate: errorRate,
		AverageTimePerJob:    kpis.AverageTimePerJob,
		EstimateUntilEmpty:   duration,
	}
}
