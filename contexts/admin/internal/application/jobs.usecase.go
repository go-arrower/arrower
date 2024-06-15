package application

import (
	"context"
	"fmt"
	"time"

	"github.com/go-arrower/arrower/contexts/admin/internal/domain/jobs"
)

//go:generate go run github.com/hexdigest/gowrap/cmd/gowrap gen -p github.com/go-arrower/arrower/contexts/admin/internal/application -g -i JobsApplication -t ./templates/slog.html -o jobs.log.usecase.go
type JobsApplication interface {
	Queues(ctx context.Context) (jobs.QueueNames, error)
	RescheduleJob(ctx context.Context, in RescheduleJobRequest) error
}

func NewJobsApplication(repo jobs.Repository) *JobsUsecase {
	return &JobsUsecase{
		repo: repo,
	}
}

type JobsUsecase struct {
	repo jobs.Repository
}

var _ JobsApplication = (*JobsUsecase)(nil)

// Queues returns a list of all known Queues.
func (app *JobsUsecase) Queues(ctx context.Context) (jobs.QueueNames, error) {
	return app.repo.Queues(ctx)
}

type (
	RescheduleJobRequest struct {
		JobID string
	}
)

func (app *JobsUsecase) RescheduleJob(ctx context.Context, in RescheduleJobRequest) error {
	err := app.repo.RunJobAt(ctx, in.JobID, time.Now())

	return fmt.Errorf("%w", err)
}
