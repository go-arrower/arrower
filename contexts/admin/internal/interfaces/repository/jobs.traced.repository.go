package repository

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-arrower/arrower/contexts/admin/internal/domain/jobs"
)

func NewTracedJobsRepository(repo jobs.Repository) *TracedJobsRepository {
	return &TracedJobsRepository{repo: repo}
}

type TracedJobsRepository struct {
	repo jobs.Repository
}

var _ jobs.Repository = (*TracedJobsRepository)(nil)

// TODO add the name of the repository as an attribute JobsRepository (or even better the original name: PostgresJobsRepository)
func (repo *TracedJobsRepository) Queues(ctx context.Context) (jobs.QueueNames, error) {
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("arrower.jobs"). // todo rename to skeleton.repository
												Start(ctx, "repo", trace.WithAttributes(attribute.String("method", "Queues")))
	defer span.End()

	return repo.repo.Queues(ctx) //nolint:wrapcheck // this is decorator
}

func (repo *TracedJobsRepository) PendingJobs(ctx context.Context, queue jobs.QueueName) ([]jobs.Job, error) {
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("arrower.jobs").
		Start(ctx, "repo", trace.WithAttributes(
			attribute.String("method", "PendingJobs"),
			attribute.String("queue", string(queue)),
		))
	defer span.End()

	return repo.repo.PendingJobs(ctx, queue) //nolint:wrapcheck // this is decorator
}

func (repo *TracedJobsRepository) QueueKPIs(ctx context.Context, queue jobs.QueueName) (jobs.QueueKPIs, error) {
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("arrower.jobs").
		Start(ctx, "repo", trace.WithAttributes(
			attribute.String("method", "QueueKPIs"),
			attribute.String("queue", string(queue)),
		))
	defer span.End()

	return repo.repo.QueueKPIs(ctx, queue) //nolint:wrapcheck // this is decorator
}

func (repo *TracedJobsRepository) Delete(ctx context.Context, jobID string) error {
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("arrower.jobs").
		Start(ctx, "repo", trace.WithAttributes(
			attribute.String("method", "Delete"),
			attribute.String("jobID", jobID),
		))
	defer span.End()

	return repo.repo.Delete(ctx, jobID) //nolint:wrapcheck // this is decorator
}

func (repo *TracedJobsRepository) RunJobAt(ctx context.Context, jobID string, runAt time.Time) error {
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("arrower.jobs").
		Start(ctx, "repo", trace.WithAttributes(
			attribute.String("method", "RunJobAt"),
			attribute.String("jobID", jobID),
			attribute.String("runAt", runAt.String()),
		))
	defer span.End()

	return repo.repo.RunJobAt(ctx, jobID, runAt) //nolint:wrapcheck // this is decorator
}

func (repo *TracedJobsRepository) WorkerPools(ctx context.Context) ([]jobs.WorkerPool, error) {
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("arrower.jobs").
		Start(ctx, "repo", trace.WithAttributes(
			attribute.String("method", "WorkerPools"),
		))
	defer span.End()

	return repo.repo.WorkerPools(ctx) //nolint:wrapcheck // this is decorator
}

func (repo *TracedJobsRepository) FinishedJobs(ctx context.Context, f jobs.Filter) ([]jobs.Job, error) {
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("arrower.jobs").
		Start(ctx, "repo", trace.WithAttributes(
			attribute.String("method", "FinishedJobs"),
		))
	defer span.End()

	return repo.repo.FinishedJobs(ctx, f) //nolint:wrapcheck // this is decorator
}

func (repo *TracedJobsRepository) FinishedJobsTotal(ctx context.Context, f jobs.Filter) (int64, error) {
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("arrower.jobs").
		Start(ctx, "repo", trace.WithAttributes(
			attribute.String("method", "FinishedJobsTotal"),
		))
	defer span.End()

	return repo.repo.FinishedJobsTotal(ctx, f) //nolint:wrapcheck // this is decorator
}
