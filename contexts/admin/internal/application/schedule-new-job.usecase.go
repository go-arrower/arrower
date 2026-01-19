package application

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/propagation"

	"github.com/go-arrower/arrower/app"
	"github.com/go-arrower/arrower/contexts/admin/internal/interfaces/repository/models"
)

var ErrScheduleJobsFailed = errors.New("schedule jobs failed")

func NewScheduleJobsCommandHandler(queries *models.Queries) app.Command[ScheduleJobsCommand] {
	return &scheduleJobsCommandHandler{queries: queries}
}

type ScheduleJobsCommand struct {
	RunAt    time.Time
	Queue    string
	JobType  string
	Payload  string
	Count    int
	Priority int16
}

type scheduleJobsCommandHandler struct {
	queries *models.Queries
}

func (h *scheduleJobsCommandHandler) H(ctx context.Context, cmd ScheduleJobsCommand) error {
	carrier := propagation.MapCarrier{}
	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})

	propagator.Inject(ctx, carrier)

	jobs, err := buildJobs(cmd, carrier)
	if err != nil {
		return fmt.Errorf("%w: could not build jobs: %w", ErrScheduleJobsFailed, err)
	}

	_, err = h.queries.ScheduleJobs(ctx, jobs)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrScheduleJobsFailed, err)
	}

	return nil
}

func buildJobs(in ScheduleJobsCommand, carrier propagation.MapCarrier) ([]models.ScheduleJobsParams, error) {
	jobs := make([]models.ScheduleJobsParams, in.Count)

	entropy := &ulid.LockedMonotonicReader{
		MonotonicReader: ulid.Monotonic(rand.Reader, 0),
	}

	buf := map[string]interface{}{}

	err := json.Unmarshal([]byte(strings.TrimSpace(in.Payload)), &buf)
	if err != nil {
		return nil, fmt.Errorf("%w: could not unmarshal job: %v", ErrScheduleJobsFailed, err)
	}

	args, err := json.Marshal(JobPayload{JobData: buf, Carrier: carrier})
	if err != nil {
		return nil, fmt.Errorf("%w: could not marshal job: %v", ErrScheduleJobsFailed, err)
	}

	for i := range in.Count {
		jobID, _ := ulid.New(ulid.Now(), entropy)

		jobs[i] = models.ScheduleJobsParams{
			JobID:     jobID.String(),
			CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true, InfinityModifier: pgtype.Finite},
			UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true, InfinityModifier: pgtype.Finite},
			Queue:     in.Queue,
			JobType:   in.JobType,
			Priority:  in.Priority,
			RunAt:     pgtype.Timestamptz{Time: in.RunAt, Valid: true, InfinityModifier: pgtype.Finite},
			Args:      args,
		}
	}

	return jobs, nil
}

type JobPayload struct { // todo reuse the one in the jobs package
	// Carrier contains the otel tracing information.
	Carrier propagation.MapCarrier `json:"carrier"`
	// JobData is the actual data as string instead of []byte,
	// so that it is readable more easily when assessing it via psql directly.
	JobData interface{} `json:"jobData"`
}
