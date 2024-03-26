package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vgarvardt/gue/v5"
	"go.opentelemetry.io/otel/propagation"

	"github.com/go-arrower/arrower"
)

// CtxJobID contains the current job ID.
const CtxJobID arrower.CTXKey = "arrower.jobs"

var (
	ErrRegisterJobFuncFailed = errors.New("register JobFunc failed")
	ErrInvalidJobFunc        = fmt.Errorf("%w: invalid JobFunc func signature", ErrRegisterJobFuncFailed)
	ErrEnqueueFailed         = errors.New("enqueue failed")
	ErrInvalidJobType        = fmt.Errorf("%w: invalid job type", ErrEnqueueFailed)
	ErrInvalidJobOpt         = fmt.Errorf("%w: invalid job option", ErrEnqueueFailed)
	ErrInvalidQueueOpt       = errors.New("todo")
	ErrJobFuncFailed         = errors.New("arrower: job failed")
)

// Enqueuer is an interface that allows new Jobs to be enqueued.
type Enqueuer interface {
	// Enqueue schedules new Jobs. Use the JobOpts to configure the Jobs scheduled.
	// You can schedule and individual or multiple jobs at the same time.
	// If ctx has a postgres.CtxTX present, that transaction is used to persist the new job(s).
	Enqueue(ctx context.Context, job Job, jobOptions ...JobOption) error
}

type Queue interface {
	Enqueuer

	// RegisterJobFunc registers a new JobFunc in the Queue. The name of the Job struct of JobFunc is used
	// as the job type, except Job implements the JobType interface. Then that is used as a job type.
	//
	// The queue starts processing Jobs automatically after the given poll interval via WithPollInterval (default 5 sec),
	// as a waiting time for more JobFuncs to be registered. Consecutive calls to RegisterJobFunc reset the interval.
	// Subsequent calls to RegisterJobFunc will restart the queue, as the underlying library gue
	// requires all workers to be known before start.
	RegisterJobFunc(jobFunc JobFunc) error

	// Shutdown blocks and waits until all started jobs are finished.
	// Timeout does not work currently.
	Shutdown(vtc context.Context) error
}

type (
	// Job has two purposes:
	// 1. Carry the payload passed between job creator and worker.
	//    The type of Job has to be a named struct that optionally implements JobType.
	// 2. Is a placeholder for any concrete implementation of the JobType interface. Its purpose is that it can be
	//    used as a type for the JobType functions.
	// Job can be a single struct, a slice of structs, or an arbitrary slice of structs, each element representing a
	// job to be scheduled.
	Job any

	// JobType returns the Job's type. It is optional and does not have to be
	// implemented by each Job. If it's not implemented the struct type is used as JobType instead.
	JobType interface {
		JobType() string
	}

	// JobFunc is the subscriber's handler and must have the signature:
	// func(ctx context.Context, job Job) error {}.
	//
	// If using a signature for Register like: Register(f func(ctx context.Context, job any) error) error, the compiler throws:
	// `cannot use func(ctx context.Context, data []byte) error {…} (value of type func(ctx context.Context, data []byte) error) as func(ctx context.Context, job any) error value in argument to Register.`
	//nolint:lll // allow full compiler message in one line
	JobFunc any // func(ctx context.Context, job Job) error
)

type (
	// QueueOption are functions that allow different behaviour of a Queue.
	QueueOption func(*queueOpt)

	// JobOption are functions which allow specific changes in the behaviour of a Job, e.g.
	// set a priority or a time at which the job should run at.
	JobOption func(p Job) error
)

type PollStrategy int

const (
	// PriorityPollStrategy cares about the priority first to lock top priority Jobs first even if there are available
	// ones that should be executed earlier but with lower priority.
	PriorityPollStrategy PollStrategy = iota
	// RunAtPollStrategy cares about the scheduled time first to lock earliest to execute jobs first even if there
	// are ones with a higher priority scheduled to a later time but already eligible for execution.
	RunAtPollStrategy
)

type queueOpt struct {
	queue        string
	poolName     string
	poolSize     int
	pollInterval time.Duration
	pollStrategy PollStrategy
}

// WithQueue sets the name of the queue used for all Jobs.
func WithQueue(queue string) QueueOption {
	return func(h *queueOpt) {
		h.queue = queue
	}
}

// WithPollInterval sets the duration in which to check the database for new Jobs.
func WithPollInterval(d time.Duration) QueueOption {
	return func(h *queueOpt) {
		h.pollInterval = d
	}
}

// WithPoolSize sets the number of workers used to poll from the queue.
func WithPoolSize(n int) QueueOption {
	return func(h *queueOpt) {
		h.poolSize = n
	}
}

// WithPoolName sets the name of the worker pool.
func WithPoolName(n string) QueueOption {
	return func(h *queueOpt) {
		h.poolName = n
	}
}

// WithPollStrategy overrides default poll strategy with given value.
func WithPollStrategy(s PollStrategy) QueueOption {
	return func(h *queueOpt) {
		h.pollStrategy = s
	}
}

// WithPriority changes the priority of a Job.
// The default priority is 0, and a lower number means a higher priority.
func WithPriority(priority int16) JobOption {
	return func(j Job) error {
		if j, ok := (j).(*gue.Job); ok {
			j.Priority = gue.JobPriority(priority)

			return nil
		}

		return ErrInvalidJobOpt
	}
}

// WithRunAt defines the time when a Job should be run at. Use it to schedule the Job into the future.
// This is not a guarantee, that the Job will be executed at the exact runAt time, just that it will not
// be processed earlier. If your queue is full, or you have to few workers, it might be picked up later.
func WithRunAt(runAt time.Time) JobOption {
	return func(j Job) error {
		if j, ok := (j).(*gue.Job); ok {
			j.RunAt = runAt

			return nil
		}

		return ErrInvalidJobOpt
	}
}

type (
	// PersistencePayload is the structure of how a Job is saved by each Queue implementation.
	//
	// The JSON of this struct is shown in the admin UI => keep the order if fields (especially JobData at the top).
	//nolint:govet // fieldalignment not important compared to sort order of fields when serialising.
	PersistencePayload struct {
		// JobData is the actual data as string instead of []byte,
		// so that it is readable more easily when assessing it via psql directly.
		JobData interface{} `json:"jobData"`

		// JobStructPath is the full path of the struct / Job payload.
		// It will be the type's PkgPath.Name with the prefix of the executing module (your app) removed.
		JobStructPath string `json:"jobStructPath"`

		// GitHashEnqueued is the version of the source code used that got the Job enqueued.
		GitHashEnqueued string `json:"gitHashEnqueued"`
		// GitHashProcessed is the version the Job got processed with by a JobFunc.
		// It is only set when the PersistencePayload is stored in the history.
		GitHashProcessed string `json:"gitHashProcessed"`

		// Ctx persists some NOT ALL information stored in the context.
		Ctx PersistenceCtxPayload `json:"ctx"`
	}

	// PersistenceCtxPayload contains a selected subset of the values stored in the request ctx.
	// These values are partially handed down to the job worker's ctx.
	//
	// Note: slog.Attr saves the value as a pointer and does not persist when marshalled to json.
	//nolint:govet // fieldalignment not important compared to sort order of fields when serialising.
	PersistenceCtxPayload struct {
		UserID string `json:"userId"`

		// Carrier contains the otel tracing information.
		Carrier propagation.MapCarrier `json:"carrier"`
	}
)

func FromContext(ctx context.Context) (string, bool) {
	jobID, ok := ctx.Value(CtxJobID).(string)

	return jobID, ok
}
