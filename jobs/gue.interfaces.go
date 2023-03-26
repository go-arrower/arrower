package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vgarvardt/gue/v5"
	"github.com/vgarvardt/gue/v5/adapter/pgxv5"
	"golang.org/x/sync/errgroup"
)

// QueueOpt are functions that allow the GueHandler different behaviour.
type QueueOpt func(*GueHandler)

func WithQueue(queue string) QueueOpt {
	return func(h *GueHandler) {
		h.queue = queue
	}
}

func WithPollInterval(d time.Duration) QueueOpt {
	return func(h *GueHandler) {
		h.pollInterval = d
	}
}

func WithPoolSize(n int) QueueOpt {
	return func(h *GueHandler) {
		h.poolSize = n
	}
}

// NewGueJobs returns an initialised GueHandler.
// Each Worker in the pool default to a poll interval of 5 seconds, which can be
// overridden by WithPollInterval option. The default queue is the
// nameless queue "", which can be overridden by WithQueue option.
func NewGueJobs(pgxPool *pgxpool.Pool, opts ...QueueOpt) (*GueHandler, error) {
	const (
		// defaults of gue, set it here, so it can be overwritten by QueueOpt.
		defaultQueue        = ""
		defaultPollInterval = 5 * time.Second
		defaultPoolSize     = 10
	)

	poolAdapter := pgxv5.NewConnPool(pgxPool)

	gc, err := gue.NewClient(poolAdapter)
	if err != nil {
		return nil, fmt.Errorf("could not connect gue to the database: %w", err)
	}

	handler := &GueHandler{
		gueClient:    gc,
		gueWorkMap:   gue.WorkMap{},
		pollInterval: defaultPollInterval,
		queue:        defaultQueue,
		poolSize:     defaultPoolSize,
		shutdown:     nil,
		group:        nil,
		mu:           sync.Mutex{},
		hasStarted:   false,
	}

	// apply all options to the job.
	for _, opt := range opts {
		opt(handler)
	}

	return handler, nil
}

// GueHandler is the main jobs' abstraction.
type GueHandler struct { //nolint:govet // accept fieldalignment so the struct fields are grouped by meaning
	gueClient    *gue.Client
	gueWorkMap   gue.WorkMap
	pollInterval time.Duration
	queue        string
	poolSize     int

	shutdown context.CancelFunc
	group    *errgroup.Group

	mu         sync.Mutex
	hasStarted bool
}

var _ Queue = (*GueHandler)(nil)

func (h *GueHandler) Enqueue(ctx context.Context, payload Payload, opts ...JobOpt) error {
	err := ensureValidPayloadForEnqueue(payload)
	if err != nil {
		return err
	}

	jobType, err := getJobTypeFromPayloadStruct(payload)
	if err != nil {
		return ErrInvalidJobType
	}

	args, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("could not marshal job payload: %w", err)
	}

	job := &gue.Job{ //nolint:exhaustruct // only set required properties
		Queue: h.queue,
		Type:  jobType,
		Args:  args,
	}

	// apply all options to the job.
	for _, opt := range opts {
		err = opt(job)
		if err != nil {
			return fmt.Errorf("could not apply job option: %w", err)
		}
	}

	// if db transaction is present in ctx use it, otherwise enqueue without transactional safety.
	tx, txOk := ctx.Value(CtxTX).(pgx.Tx)
	if txOk {
		err = h.gueClient.EnqueueTx(ctx, job, pgxv5.NewTx(tx))
		if err != nil {
			return fmt.Errorf("could not enqueue gue job with transaction: %w", err)
		}

		return nil
	}

	err = h.gueClient.Enqueue(ctx, job)
	if err != nil {
		return fmt.Errorf("could not enqueue gue job: %w", err)
	}

	return nil
}

func ensureValidPayloadForEnqueue(payload Payload) error {
	if payload == nil {
		return ErrInvalidJobType
	}

	pt := reflect.TypeOf(payload)

	// ensure primitive data types like string or int are not allowed
	if pt.Kind() != reflect.Struct {
		return ErrInvalidJobType
	}

	return nil
}

// getJobTypeFromPayloadStruct returns a name for a job.
// If the parameter implements the JobType interface, then that type is returned as the JobType.
// Otherwise, it is expected, that the Payload is a struct and the name of that type is returned.
func getJobTypeFromPayloadStruct(payload Payload) (string, error) {
	jt, ok := payload.(JobType)
	if ok {
		return jt.JobType(), nil
	}

	structTypeName := reflect.TypeOf(payload).Name()
	if structTypeName == "" {
		return "", ErrInvalidJobType
	}

	return structTypeName, nil
}

// RegisterWorker registers new worker functions for a given jobName. They have to be registered before the gue
// workers are run.
func (h *GueHandler) RegisterWorker(jf JobFunc) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.hasStarted {
		return ErrNotAllowed
	}

	jobFunc, err := isValidJobFunc(jf)
	if err != nil {
		return err
	}

	jobType := getJobTypeFromReflectValue(jobFunc)
	if _, ok := h.gueWorkMap[jobType]; ok {
		return fmt.Errorf("%w: could not register worker: JobType %s already registered", ErrNotAllowed, jobType)
	}

	h.gueWorkMap[jobType] = gueWorkerAdapter(jf)

	return nil
}

func isValidJobFunc(f JobFunc) (reflect.Type, error) {
	if f == nil {
		return nil, ErrInvalidJobFunc
	}

	jobFunc := reflect.TypeOf(f)

	// ensure jobFunc is indeed a function
	if jobFunc.Kind() != reflect.Func {
		return nil, ErrInvalidJobFunc
	}

	// ensure jobFunc has two parameters
	if jobFunc.NumIn() != 2 { //nolint:gomnd
		return nil, ErrInvalidJobFunc
	}

	// ensure the first parameter is of type context.Context
	if jobFunc.In(0).Kind() != reflect.Interface {
		return nil, ErrInvalidJobFunc
	}

	if jobFunc.In(0).PkgPath() != "context" && jobFunc.In(0).Name() != "Context" {
		return nil, ErrInvalidJobFunc
	}

	// ensure second parameter is a struct
	if jobFunc.In(1).Kind() != reflect.Struct {
		return nil, ErrInvalidJobFunc
	}

	// ensure jobFunc returns an error
	if jobFunc.NumOut() != 1 {
		return nil, ErrInvalidJobFunc
	}

	if jobFunc.Out(0).Name() != "error" {
		return nil, ErrInvalidJobFunc
	}

	return jobFunc, nil
}

func gueWorkerAdapter(workerFn JobFunc) gue.WorkFunc {
	return func(ctx context.Context, job *gue.Job) error {
		handlerFuncType := reflect.TypeOf(workerFn)
		paramType := handlerFuncType.In(1)

		args := reflect.New(paramType)
		argsP := args.Interface()

		if err := json.Unmarshal(job.Args, argsP); err != nil {
			return fmt.Errorf("could not unmarshal job args to job payload type: %w", err)
		}

		// make the job's tx available in the context of the worker, so they are consistent
		txHandle, ok := pgxv5.UnwrapTx(job.Tx())
		if !ok {
			return fmt.Errorf("%w: could not unwrap gue job tx for use in the worker", ErrWorkerFailed)
		}

		ctx = context.WithValue(ctx, CtxTX, txHandle)

		// call the JobFunc
		fn := reflect.ValueOf(workerFn)
		vals := fn.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(argsP).Elem().Convert(paramType),
		})

		// if JobFunc returned an error, put the job back on the queue.
		if len(vals) > 0 {
			if jobErr, ok := vals[0].Interface().(error); ok && jobErr != nil {
				return fmt.Errorf("%w: %s", ErrWorkerFailed, jobErr)
			}
		}

		return nil
	}
}

func (h *GueHandler) StartWorkers() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.hasStarted {
		return ErrNotAllowed
	}

	workers, err := gue.NewWorkerPool(h.gueClient, h.gueWorkMap, h.poolSize,
		gue.WithPoolQueue(h.queue),
		gue.WithPoolPollInterval(h.pollInterval),
		gue.WithPoolHooksJobLocked(logStartedJobs()),
		gue.WithPoolHooksJobDone(logFinishedJobs()),
	)
	if err != nil {
		return fmt.Errorf("could not create gue worker pool: %w", err)
	}

	ctx, shutdown := context.WithCancel(context.Background())

	// work jobs in goroutine
	group, gctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		err := workers.Run(gctx)
		if err != nil {
			return fmt.Errorf("gue worker failed: %w", err)
		}

		return nil
	})

	h.shutdown = shutdown
	h.group = group

	h.hasStarted = true

	return nil
}

func logStartedJobs() func(context.Context, *gue.Job, error) {
	return func(ctx context.Context, job *gue.Job, err error) {
		// if err is set, the job could not be pulled from the DB.
		if err != nil {
			return
		}

		_, _ = job.Tx().Exec(ctx, `INSERT INTO public.gue_jobs_history (job_id, priority, run_at, job_type, args, run_count, run_error, queue, created_at, updated_at, success, finished_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW(), FALSE, NULL);`, //nolint:lll,dupword
			job.ID.String(), job.Priority, job.RunAt, job.Type, job.Args, job.ErrorCount, job.LastError, job.Queue,
		)
	}
}

// logFinishedJobs takes each job that's finished and logs it into a new table,
// so it's persisted for later analytics.
// gue does delete finished jobs from the gue_jobs table and the information would be lost otherwise.
func logFinishedJobs() func(context.Context, *gue.Job, error) {
	return func(ctx context.Context, job *gue.Job, err error) {
		if err != nil { // job returned with an error and worker JobFunc failed
			_, _ = job.Tx().Exec(ctx, `UPDATE public.gue_jobs_history SET run_error = $1, finished_at = NOW() WHERE job_id = $2 AND run_count = $3 AND finished_at IS NULL;`, //nolint:lll
				err.Error(), job.ID.String(), job.ErrorCount,
			)

			return
		}

		_, _ = job.Tx().Exec(ctx, `UPDATE public.gue_jobs_history SET run_count = $1, run_error = '', success = TRUE, finished_at = NOW() WHERE job_id = $2 AND run_count = $3 AND finished_at IS NULL;`, //nolint:lll
			job.ErrorCount, job.ID.String(), job.ErrorCount,
		)
	}
}

func (h *GueHandler) Shutdown(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.hasStarted {
		return nil
	}

	// send shutdown signal to worker
	h.shutdown()

	if err := h.group.Wait(); err != nil {
		return fmt.Errorf("could not shutdown job workers: %w", err)
	}

	return nil
}

func WithPriority(priority int16) JobOpt {
	return func(j Job) error {
		if j, ok := (j).(*gue.Job); ok {
			j.Priority = gue.JobPriority(priority)

			return nil
		}

		return ErrInvalidJobOpt
	}
}

func WithRunAt(runAt time.Time) JobOpt {
	return func(j Job) error {
		if j, ok := (j).(*gue.Job); ok {
			j.RunAt = runAt

			return nil
		}

		return ErrInvalidJobOpt
	}
}

// getJobTypeFromReflectValue returns the jobType as the struct name or takes it from JobType,
// if that interface is implemented.
func getJobTypeFromReflectValue(jobFunc reflect.Type) string {
	jobType := jobFunc.In(1).Name()

	if jobFunc.In(1).Implements(reflect.TypeOf((*JobType)(nil)).Elem()) {
		paramType := reflect.New(jobFunc.In(1))
		paramTypeP := paramType.Interface()

		if t, ok := paramTypeP.(JobType); ok {
			jobType = t.JobType()
		}
	}

	return jobType
}
