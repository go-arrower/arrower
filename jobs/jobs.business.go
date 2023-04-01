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

type (
	// JobFunc is the subscriber's handler and must have the signature:
	// func(ctx context.Context, payload Payload) error {}.
	JobFunc any // func(ctx context.Context, payload Payload) error

	// Payload is a payload passed between job creator and worker.
	// The type of Payload has to be a named struct that optionally implements JobType.
	Payload any

	// JobType returns the job's type. It is optional and does not have to be
	// implemented by each Payload. If it's not implemented the struct type is used as JobType instead.
	JobType interface {
		JobType() string
	}

	// Job is a placeholder for any concrete implementation of this interface. It's main purpose is that it can be
	// used as a type for the JobOpt functions.
	Job any

	// JobOpt are functions which allow Queue implementation specific changes in behaviour of a Job, e.g.
	// set a priority or a time at which the job should run at.
	JobOpt func(p Job) error
)

// Queuer is an interface that allows new jobs to be enqueued.
type Queuer interface {
	Enqueue(ctx context.Context, payload Payload, opts ...JobOpt) error
}

type Queue interface {
	Queuer

	// RegisterWorker registers a new JobFunc in the Queue. The name of the Payload struct of JobFunc is used
	// as the job type, except Payload implements the JobType interface, than that is used as a job type.
	RegisterWorker(f JobFunc) error

	// StartWorkers starts the job queue and processing begins.
	StartWorkers() error

	// Shutdown blocks and wait for all started jobs are finished or for the context timed out, whichever happens first.
	Shutdown(ctx context.Context) error
}
