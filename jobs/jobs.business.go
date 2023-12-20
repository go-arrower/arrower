package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vgarvardt/gue/v5"
)

var (
	ErrRegisterJobFuncFailed = errors.New("register JobFunc failed")
	ErrInvalidJobFunc        = fmt.Errorf("%w: invalid JobFunc func signature", ErrRegisterJobFuncFailed)
	ErrEnqueueFailed         = errors.New("enqueue failed")
	ErrInvalidJobType        = fmt.Errorf("%w: invalid job type", ErrEnqueueFailed)
	ErrInvalidJobOpt         = fmt.Errorf("%w: invalid job option", ErrEnqueueFailed)
	ErrJobFuncFailed         = errors.New("arrower: job failed")
)

// Enqueuer is an interface that allows new Jobs to be enqueued.
type Enqueuer interface {
	// Enqueue schedules new Jobs. Use the JobOpts to configure the Jobs scheduled.
	// You can schedule and individual or multiple jobs at the same time.
	// If ctx has a postgres.CtxTX present, that transaction is used to persist the new job(s).
	Enqueue(context.Context, Job, ...JobOpt) error
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
	RegisterJobFunc(JobFunc) error

	// Shutdown blocks and waits until all started jobs are finished.
	// Timeout does not work currently.
	Shutdown(context.Context) error
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

	// JobOpt are functions which allow Queue implementation specific changes in behaviour of a Job, e.g.
	// set a priority or a time at which the job should run at.
	JobOpt func(p Job) error
)

// WithPriority changes the priority of a Job. The default priority is 0, and a lower number means a higher priority.
func WithPriority(priority int16) JobOpt {
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
func WithRunAt(runAt time.Time) JobOpt {
	return func(j Job) error {
		if j, ok := (j).(*gue.Job); ok {
			j.RunAt = runAt

			return nil
		}

		return ErrInvalidJobOpt
	}
}
