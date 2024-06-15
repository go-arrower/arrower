package application

import (
	"context"
	"errors"

	"github.com/go-arrower/arrower/contexts/admin/internal/domain/jobs"
)

var ErrJobsAppFailed = errors.New("use case error")

func NewJobsSuccessApplication() *JobsSuccess { return &JobsSuccess{} }

type JobsSuccess struct{}

var _ JobsApplication = (*JobsSuccess)(nil)

func (app *JobsSuccess) Queues(_ context.Context) (jobs.QueueNames, error) { return nil, nil }

func (app *JobsSuccess) ListAllQueues(_ context.Context, _ ListAllQueuesQuery) (ListAllQueuesResponse, error) {
	return ListAllQueuesResponse{}, nil
}

func (app *JobsSuccess) GetWorkers(_ context.Context, _ GetWorkersQuery) (GetWorkersResponse, error) {
	return GetWorkersResponse{}, nil
}

func (app *JobsSuccess) ScheduleJobs(_ context.Context, _ ScheduleJobsCommand) error { return nil }

func (app *JobsSuccess) RescheduleJob(_ context.Context, _ RescheduleJobRequest) error { return nil }

func (app *JobsSuccess) JobTypesForQueue(_ context.Context, _ jobs.QueueName) ([]jobs.JobType, error) {
	return nil, nil
}

func (app *JobsSuccess) VacuumJobsTable(_ context.Context, _ string) error { return nil }

func (app *JobsSuccess) PruneHistory(_ context.Context, _ int) error { return nil }

func NewJobsFailureApplication() *JobsFailure { return &JobsFailure{} }

type JobsFailure struct{}

var _ JobsApplication = (*JobsFailure)(nil)

func (app *JobsFailure) Queues(_ context.Context) (jobs.QueueNames, error) {
	return nil, ErrJobsAppFailed
}

func (app *JobsFailure) ListAllQueues(_ context.Context, _ ListAllQueuesQuery) (ListAllQueuesResponse, error) {
	return ListAllQueuesResponse{}, ErrJobsAppFailed
}

func (app *JobsFailure) GetWorkers(_ context.Context, _ GetWorkersQuery) (GetWorkersResponse, error) {
	return GetWorkersResponse{}, ErrJobsAppFailed
}

func (app *JobsFailure) ScheduleJobs(_ context.Context, _ ScheduleJobsCommand) error {
	return ErrJobsAppFailed
}

func (app *JobsFailure) RescheduleJob(_ context.Context, _ RescheduleJobRequest) error {
	return ErrJobsAppFailed
}

func (app *JobsFailure) JobTypesForQueue(_ context.Context, _ jobs.QueueName) ([]jobs.JobType, error) {
	return nil, ErrJobsAppFailed
}

func (app *JobsFailure) VacuumJobsTable(_ context.Context, _ string) error { return ErrJobsAppFailed }

func (app *JobsFailure) PruneHistory(_ context.Context, _ int) error { return ErrJobsAppFailed }
