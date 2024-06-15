package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-arrower/arrower/app"

	"github.com/go-arrower/arrower/contexts/admin/internal/domain/jobs"
	"github.com/go-arrower/arrower/contexts/admin/internal/interfaces/repository/models"
)

var ErrJobTypesForQueueFailed = errors.New("job types for queue failed")

func NewJobTypesForQueueQueryHandler(queries *models.Queries) app.Query[JobTypesForQueueQuery, []jobs.JobType] {
	return &jobTypesForQueueQueryHandler{queries: queries}
}

type jobTypesForQueueQueryHandler struct {
	queries *models.Queries
}

type (
	JobTypesForQueueQuery struct {
		Queue jobs.QueueName
	}
)

func (h jobTypesForQueueQueryHandler) H(ctx context.Context, query JobTypesForQueueQuery) ([]jobs.JobType, error) {
	queue := query.Queue
	if query.Queue == jobs.DefaultQueueName { // todo move check to repo
		queue = ""
	}

	types, err := h.queries.JobTypes(ctx, string(queue))
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrJobTypesForQueueFailed, queue, err) //nolint:errorlint,lll // prevent err in api
	}

	jobTypes := make([]jobs.JobType, len(types))
	for i, jt := range types {
		jobTypes[i] = jobs.JobType(jt)
	}

	return jobTypes, nil
}
