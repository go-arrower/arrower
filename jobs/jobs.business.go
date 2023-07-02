package jobs

import (
	"context"
	"errors"
)

var (
	ErrInvalidJobFunc = errors.New("invalid JobFunc func signature")
	ErrInvalidJobType = errors.New("invalid job type")
	ErrInvalidJobOpt  = errors.New("invalid job option")
	ErrNotAllowed     = errors.New("not allowed")
	ErrWorkerFailed   = errors.New("arrower: job failed")
)

// Queuer is an interface that allows new jobs to be enqueued.
type Queuer interface {
	Enqueue(ctx context.Context, job Job, opts ...JobOpt) error
	// todo make a version that can take batches
}

type Queue interface {
	Queuer

	// RegisterWorker registers a new JobFunc in the Queue. The name of the Job struct of JobFunc is used
	// as the job type, except Job implements the JobType interface, than that is used as a job type.
	RegisterWorker(f JobFunc) error

	// StartWorkers starts the job queue and processing begins.
	StartWorkers() error // FIXME remove method

	// Shutdown blocks and wait for all started jobs are finished or for the context timed out, whichever happens first.
	Shutdown(ctx context.Context) error
}

type (
	// Job has two purposes:
	// 1. carry the payload passed between job creator and worker.
	//    The type of Job has to be a named struct that optionally implements JobType.
	// 2. is a placeholder for any concrete implementation of the JobOpt interface. Its purpose is that it can be
	//    used as a type for the JobOpt functions.
	Job any

	// JobType returns the job's type. It is optional and does not have to be
	// implemented by each Job. If it's not implemented the struct type is used as JobType instead.
	// FIXME use method name, if JobFunc has any (or context.methodName.structName).
	JobType interface {
		JobType() string
	}

	// JobFunc is the subscriber's handler and must have the signature:
	// func(ctx context.Context, job Job) error {}.
	//
	// If using a signature for Register like: Register(f func(ctx context.Context, job any) error) error, the compiler throws:
	// cannot use func(ctx context.Context, data []byte) error {…} (value of type func(ctx context.Context, data []byte) error) as func(ctx context.Context, job any) error value in argument to Register.
	//nolint:lll // allow full compiler message in one line
	JobFunc any // func(ctx context.Context, job Job) error

	// JobOpt are functions which allow Queue implementation specific changes in behaviour of a Job, e.g.
	// set a priority or a time at which the job should run at.
	JobOpt func(p Job) error
)
