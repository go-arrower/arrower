package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"reflect"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vgarvardt/gue/v5"
	"github.com/vgarvardt/gue/v5/adapter"
	"github.com/vgarvardt/gue/v5/adapter/pgxv5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/jobs/models"
	"github.com/go-arrower/arrower/postgres"
)

// QueueOpt are functions that allow the PostgresJobsHandler different behaviour.
type QueueOpt func(*PostgresJobsHandler)

// WithQueue sets the name of the queue used for all Jobs.
func WithQueue(queue string) QueueOpt {
	return func(h *PostgresJobsHandler) {
		h.queue = queue
	}
}

// WithPollInterval sets the duration in which to check the database for new Jobs.
func WithPollInterval(d time.Duration) QueueOpt {
	return func(h *PostgresJobsHandler) {
		h.pollInterval = d
	}
}

// WithPoolSize sets the number of workers used to poll from the queue.
func WithPoolSize(n int) QueueOpt {
	return func(h *PostgresJobsHandler) {
		h.poolSize = n
	}
}

// WithPoolName sets the name of the worker pool.
func WithPoolName(n string) QueueOpt {
	return func(h *PostgresJobsHandler) {
		h.poolName = n
	}
}

// WithPollStrategy overrides default poll strategy with given value.
func WithPollStrategy(s PollStrategy) QueueOpt {
	return func(h *PostgresJobsHandler) {
		h.pollStrategy = s
	}
}

// NewPostgresJobs returns an initialised PostgresJobsHandler.
// Each Worker in the pool defaults to a poll interval of 5 seconds, which can be
// overridden by WithPollInterval option. The default queue is the
// nameless queue "", which can be overridden by WithQueue option.
//
//nolint:funlen // function length is alright, just initialising the PostgresJobsHandler struct takes a lot of lines.
func NewPostgresJobs(
	logger alog.Logger,
	meterProvider metric.MeterProvider,
	traceProvider trace.TracerProvider,
	pgxPool *pgxpool.Pool,
	opts ...QueueOpt,
) (*PostgresJobsHandler, error) {
	const (
		// defaults of gue, set it here, so it can be overwritten by QueueOpt.
		defaultQueue          = ""
		defaultPollInterval   = 5 * time.Second
		defaultPoolSize       = 10
		defaultPoolNameLength = 5
	)

	logger = logger.WithGroup("arrower.jobs")
	gueLogger := &gueLogAdapter{l: logger.WithGroup("gue")}

	meter := meterProvider.Meter("arrower.jobs")
	tracer := traceProvider.Tracer("arrower.jobs")

	poolName := randomPoolName(defaultPoolNameLength)
	poolAdapter := pgxv5.NewConnPool(pgxPool)

	handler := &PostgresJobsHandler{
		version:            "",
		logger:             logger,
		gueLogger:          gueLogger,
		meter:              meter,
		tracer:             tracer,
		propagator:         propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}),
		queries:            models.New(pgxPool),
		gueClient:          nil, // has to be set after all opts have been applied
		gueWorkMap:         gue.WorkMap{},
		pollInterval:       defaultPollInterval,
		queue:              defaultQueue,
		poolName:           poolName,
		poolSize:           defaultPoolSize,
		pollStrategy:       PriorityPollStrategy,
		shutdownWorkerPool: nil,
		groupWorkerPool:    nil,
		mu:                 sync.Mutex{},
		hasStarted:         false,
		isStartInProgress:  false,
		startTimer:         nil,
	}

	// apply all options to the job
	for _, opt := range opts {
		opt(handler)
	}

	gc, err := gue.NewClient(
		poolAdapter,
		gue.WithClientID(handler.poolName), // after potential overwriting from opts
		gue.WithClientLogger(gueLogger),
		gue.WithClientMeter(meter),
	)
	if err != nil {
		return nil, fmt.Errorf("could not connect gue to the database: %w", err)
	}

	handler.gueClient = gc

	return handler, nil
}

// PostgresJobsHandler is the main jobs' abstraction.
type PostgresJobsHandler struct { //nolint:govet // accept fieldalignment so the struct fields are grouped by meaning
	version string

	logger     alog.Logger
	gueLogger  adapter.Logger
	meter      metric.Meter
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator

	queries *models.Queries

	gueClient    *gue.Client
	gueWorkMap   gue.WorkMap
	pollInterval time.Duration
	queue        string
	poolName     string
	poolSize     int
	pollStrategy PollStrategy

	shutdownWorkerPool context.CancelFunc
	groupWorkerPool    *errgroup.Group

	mu                sync.Mutex
	startTimer        *time.Timer
	isStartInProgress bool
	hasStarted        bool
}

var _ Queue = (*PostgresJobsHandler)(nil)

type jobPayload struct {
	// Carrier contains the otel tracing information.
	Carrier propagation.MapCarrier `json:"carrier"`
	// JobData is the actual data as string instead of []byte,
	// so that it is readable more easily when assessing it via psql directly.
	JobData string `json:"jobData"`
}

func (h *PostgresJobsHandler) Enqueue(ctx context.Context, job Job, opts ...JobOpt) error {
	ctx, span := h.tracer.Start(ctx, "enqueue")
	defer span.End()

	err := ensureValidJobTypeForEnqueue(job)
	if err != nil {
		return err
	}

	gueJobs, err := h.gueJobsFromJob(ctx, h.queue, job, opts...)
	if err != nil {
		return err
	}

	// if db transaction is present in ctx use it, otherwise enqueue without transactional safety.
	tx, txOk := ctx.Value(postgres.CtxTX).(pgx.Tx)
	if txOk {
		err = h.gueClient.EnqueueBatchTx(ctx, gueJobs, pgxv5.NewTx(tx))
		if err != nil {
			return fmt.Errorf("%w: could not enqueue gue with transaction: %v", ErrEnqueueFailed, err) //nolint:errorlint,lll // prevent err in api
		}

		return nil
	}

	err = h.gueClient.EnqueueBatch(ctx, gueJobs)
	if err != nil {
		return fmt.Errorf("%w: could not enqueue gue job: %v", ErrEnqueueFailed, err) //nolint:errorlint // prevent err in api
	}

	return nil
}

func (h *PostgresJobsHandler) gueJobsFromJob(
	ctx context.Context,
	queue string,
	job Job,
	opts ...JobOpt,
) ([]*gue.Job, error) {
	gueJobs := []*gue.Job{}

	carrier := propagation.MapCarrier{}
	h.propagator.Inject(ctx, carrier)

	switch reflect.ValueOf(job).Kind() { //nolint:exhaustive // other types are prevented by ensureValidJobTypeForEnqueue
	case reflect.Struct:
		jobType, err := getJobTypeFromJobStruct(job)
		if err != nil {
			return nil, err
		}

		gueJobs, err = buildAndAppendGueJob(gueJobs, queue, jobType, job, carrier, opts...)
		if err != nil {
			return nil, err
		}
	case reflect.Slice:
		allJobs := reflect.ValueOf(job)
		for i := 0; i < allJobs.Len(); i++ {
			job := allJobs.Index(i)

			jobType, err := getJobTypeFromJobSliceElement(job)
			if err != nil {
				return nil, err
			}

			gueJobs, err = buildAndAppendGueJob(gueJobs, queue, jobType, job.Interface(), carrier, opts...)
			if err != nil {
				return nil, err
			}
		}
	}

	return gueJobs, nil
}

func buildAndAppendGueJob(
	gueJobs []*gue.Job,
	queue string,
	jobType string,
	job any,
	carrier propagation.MapCarrier,
	opts ...JobOpt,
) ([]*gue.Job, error) {
	jobByte, err := json.Marshal(job)
	if err != nil {
		return nil, fmt.Errorf("%w: could not marshal job: %v", ErrEnqueueFailed, err) //nolint:errorlint,lll // prevent err in api
	}

	args, err := json.Marshal(jobPayload{JobData: string(jobByte), Carrier: carrier})
	if err != nil {
		return nil, fmt.Errorf("%w: could not marshal job: %v", ErrEnqueueFailed, err) //nolint:errorlint,lll // prevent err in api
	}

	gueJob := &gue.Job{ //nolint:exhaustruct // only set required properties
		Queue: queue,
		Type:  jobType,
		Args:  args,
	}

	// apply all options to the job.
	for _, opt := range opts {
		err := opt(gueJob)
		if err != nil {
			return nil, fmt.Errorf("could not apply job option: %w", err)
		}
	}

	return append(gueJobs, gueJob), nil
}

// ensureValidJobTypeForEnqueue checks if a Job is of a valid type to be enqueued.
// Valid are:
// - struct
// - slice of struct.
func ensureValidJobTypeForEnqueue(job Job) error {
	if job == nil {
		return ErrInvalidJobType
	}

	pt := reflect.TypeOf(job)

	if pt.Kind() == reflect.Struct {
		return nil
	}

	if pt.Kind() == reflect.Slice {
		jv := reflect.ValueOf(job)
		if jv.Len() == 0 {
			return ErrInvalidJobType
		}

		// don't allow slice of primitives like string or int are not allowed
		for i := 0; i < jv.Len(); i++ {
			var kind reflect.Kind

			if jv.Index(i).Kind() == reflect.Interface { // slice of any => extract the underlying element first
				kind = jv.Index(i).Elem().Kind()
			} else { // typed slice with only same elements
				kind = jv.Index(i).Kind()
			}

			if kind != reflect.Struct {
				return ErrInvalidJobType
			}
		}

		return nil
	}

	return ErrInvalidJobType
}

// getJobTypeFromJobStruct returns a name for a Job.
// If the parameter implements the JobType interface, then that type is returned as the JobType.
// Otherwise, it is expected that the Job is a struct and the name of that type is returned.
func getJobTypeFromJobStruct(job Job) (string, error) {
	if jt, ok := job.(JobType); ok {
		return jt.JobType(), nil
	}

	structTypeName := reflect.TypeOf(job).Name()
	if structTypeName == "" {
		return "", ErrInvalidJobType
	}

	return structTypeName, nil
}

// getJobTypeFromJobSliceElement does the same getJobTypeFromJobStruct does for a struct but on an element of a slice.
func getJobTypeFromJobSliceElement(job reflect.Value) (string, error) {
	if job.Kind() == reflect.Interface { // slice with different types: []any => extract the underlying element first
		job = job.Elem()
	}

	jobTypeInterfaceType := reflect.TypeOf((*JobType)(nil)).Elem()
	if job.CanConvert(jobTypeInterfaceType) {
		if jv, ok := job.Convert(jobTypeInterfaceType).Interface().(JobType); ok {
			return jv.JobType(), nil
		}

		return "", ErrInvalidJobType
	}

	structTypeName := job.Type().Name()
	if structTypeName == "" {
		return "", ErrInvalidJobType
	}

	return structTypeName, nil
}

// RegisterJobFunc registers new worker functions for a given JobType.
func (h *PostgresJobsHandler) RegisterJobFunc(jf JobFunc) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	ok := isValidJobFunc(jf)
	if !ok {
		return ErrInvalidJobFunc
	}

	jobType := getJobTypeFromJobFunc(jf)
	if _, ok := h.gueWorkMap[jobType]; ok {
		return fmt.Errorf("%w: could not register worker: JobType %s already registered", ErrInvalidJobFunc, jobType)
	}

	if h.hasStarted { // all jobFuncs have to be registered before the gue workers are run
		h.logger.Log(context.Background(), alog.LevelInfo, "restart workers",
			slog.String("queue", h.queue), slog.String("pool_name", h.poolName))

		err := h.shutdown(context.Background())
		if err != nil {
			return fmt.Errorf("could not shutdown after registration of new JobFunc: %w", err)
		}
	}

	h.gueWorkMap[jobType] = h.gueWorkerAdapter(jf)

	needsToStartWorkers := !h.hasStarted
	if needsToStartWorkers {
		if h.isStartInProgress {
			if ok := h.startTimer.Stop(); !ok { // timer expired => worker already started
				return nil
			}
		}

		// wait a bit to allow other JobFuncs to register, so the restart of gue is not happening on each call.
		h.startTimer = time.AfterFunc(h.pollInterval, func() {
			err := h.startWorkers()
			if err != nil {
				h.logger.InfoContext(context.Background(), "could not start workers after registration of new JobFunc",
					slog.Any("err", err))
			}
		})
		h.isStartInProgress = true
	}

	return nil
}

func isValidJobFunc(f JobFunc) bool {
	if f == nil {
		return false
	}

	jobFunc := reflect.TypeOf(f)

	// ensure jobFunc is indeed a function
	if jobFunc.Kind() != reflect.Func {
		return false
	}

	// ensure jobFunc has two parameters
	if jobFunc.NumIn() != 2 { //nolint:gomnd
		return false
	}

	// ensure the first parameter is of type context.Context
	if jobFunc.In(0).Kind() != reflect.Interface {
		return false
	}

	if jobFunc.In(0).PkgPath() != "context" && jobFunc.In(0).Name() != "Context" {
		return false
	}

	// ensure second parameter is a struct
	if jobFunc.In(1).Kind() != reflect.Struct {
		return false
	}

	// ensure jobFunc returns an error
	if jobFunc.NumOut() != 1 {
		return false
	}

	if jobFunc.Out(0).Name() != "error" {
		return false
	}

	return true
}

func (h *PostgresJobsHandler) gueWorkerAdapter(workerFn JobFunc) gue.WorkFunc {
	return func(ctx context.Context, job *gue.Job) error {
		handlerFuncType := reflect.TypeOf(workerFn)
		paramType := handlerFuncType.In(1)
		args := reflect.New(paramType)
		argsP := args.Interface()

		payload := jobPayload{} //nolint:exhaustruct

		if err := json.Unmarshal(job.Args, &payload); err != nil {
			return fmt.Errorf("%w: could not unmarshal job args to job type: %w", ErrJobFuncFailed, err)
		}

		if err := json.Unmarshal([]byte(payload.JobData), argsP); err != nil {
			return fmt.Errorf("%w: could not unmarshal job args to job type: %w", ErrJobFuncFailed, err)
		}

		// make the gue job's tx available in the context of the worker, so db can stay consistent
		txHandle, ok := pgxv5.UnwrapTx(job.Tx())
		if !ok {
			return fmt.Errorf("%w: could not unwrap gue job tx for use in the worker", ErrJobFuncFailed)
		}

		parentCtx := h.propagator.Extract(ctx, payload.Carrier)

		ctx, childSpan := h.tracer.Start(parentCtx, fmt.Sprintf("job: %s run: %d", paramType.String(), job.ErrorCount))
		defer childSpan.End()

		childSpan.SetAttributes(
			attribute.String("jobID", job.ID.String()),
			attribute.String("queue", job.Queue),
			attribute.String("type", job.Type),
			attribute.Int("priority", int(job.Priority)),
			attribute.Int("run_count", int(job.ErrorCount)),
			attribute.String("run_at", job.RunAt.Format(time.RFC3339)),
		)

		ctx = alog.AddAttr(ctx, slog.String("jobID", job.ID.String()))
		ctx = context.WithValue(ctx, postgres.CtxTX, txHandle)

		// call the JobFunc
		fn := reflect.ValueOf(workerFn)
		vals := fn.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(argsP).Elem().Convert(paramType),
		})

		// if JobFunc returned an error, put the job back on the queue.
		if len(vals) > 0 {
			if jobErr, ok := vals[0].Interface().(error); ok && jobErr != nil {
				childSpan.SetStatus(codes.Error, jobErr.Error())

				return fmt.Errorf("%w: %s", ErrJobFuncFailed, jobErr) //nolint:errorlint // prevent err in api
			}
		}

		return nil
	}
}

// startWorkers expects the locking of h.mu to happen at the caller!
func (h *PostgresJobsHandler) startWorkers() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.hasStarted {
		return fmt.Errorf("%w: queue already started", ErrRegisterJobFuncFailed)
	}

	const defaultPanicStackBufSize = 4 * 1024 // 2 * gue's default

	workers, err := gue.NewWorkerPool(h.gueClient, h.gueWorkMap, h.poolSize,
		gue.WithPoolQueue(h.queue), gue.WithPoolPollInterval(h.pollInterval),
		gue.WithPoolHooksJobLocked(recordStartedJobsToHistory(h.logger, h.queries)),
		gue.WithPoolHooksJobDone(recordFinishedJobsToHistory(h.logger, h.queries)),
		gue.WithPoolID(h.poolName),
		gue.WithPoolLogger(h.gueLogger), gue.WithPoolMeter(h.meter), gue.WithPoolTracer(h.tracer),
		gue.WithPoolPollStrategy(pollStrategyToGue(h.pollStrategy)),
		gue.WithPoolPanicStackBufSize(defaultPanicStackBufSize),
	)
	if err != nil {
		return fmt.Errorf("%w: could not create gue worker pool: %v", ErrRegisterJobFuncFailed, err) //nolint:errorlint,lll // prevent err in api
	}

	ctx, shutdown := context.WithCancel(context.Background())

	go h.continuouslyRegisterWorkerPool(ctx)

	// work jobs in goroutine
	group, gctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		err := workers.Run(gctx)
		if err != nil {
			return fmt.Errorf("gue worker failed: %w", err)
		}

		return nil
	})

	h.shutdownWorkerPool = shutdown
	h.groupWorkerPool = group

	h.hasStarted = true
	h.isStartInProgress = false

	return nil
}

func pollStrategyToGue(s PollStrategy) gue.PollStrategy {
	switch s {
	case RunAtPollStrategy:
		return gue.RunAtPollStrategy
	default:
		return gue.PriorityPollStrategy
	}
}

// continuouslyRegisterWorkerPool registers the worker pool regularly, so it stays "active" for monitoring.
func (h *PostgresJobsHandler) continuouslyRegisterWorkerPool(ctx context.Context) {
	const refreshDuration = 30 * time.Second

	err := connOrTX(ctx, h.queries).UpsertWorkerToPool(ctx, models.UpsertWorkerToPoolParams{
		ID:        h.poolName,
		Queue:     h.queue,
		Version:   h.version,
		JobTypes:  registeredJobTypes(h.gueWorkMap),
		Workers:   int16(h.poolSize),
		UpdatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true, InfinityModifier: pgtype.Finite},
	})
	if err != nil {
		h.logger.InfoContext(ctx, "could not save starting worker pool to the database", slog.Any("err", err))
	}

	for {
		select {
		case <-time.NewTicker(refreshDuration).C:
			err := connOrTX(ctx, h.queries).UpsertWorkerToPool(ctx, models.UpsertWorkerToPoolParams{
				ID:        h.poolName,
				Queue:     h.queue,
				Version:   h.version,
				JobTypes:  registeredJobTypes(h.gueWorkMap),
				Workers:   int16(h.poolSize),
				UpdatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true, InfinityModifier: pgtype.Finite},
			})
			if err != nil {
				h.logger.InfoContext(ctx, "could not save worker pool life probe to the database", slog.Any("err", err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func registeredJobTypes(workMap gue.WorkMap) []string {
	jobs := []string{}

	for jobType := range workMap {
		jobs = append(jobs, jobType)
	}

	return jobs
}

func recordStartedJobsToHistory(logger alog.Logger, db *models.Queries) func(context.Context, *gue.Job, error) {
	return func(ctx context.Context, job *gue.Job, jobErr error) {
		// if jobErr is set, the job could not be pulled from the DB.
		if jobErr != nil {
			return
		}

		tx, ok := pgxv5.UnwrapTx(job.Tx())
		if !ok {
			logger.InfoContext(ctx, "could not access transaction to record job in history")

			return
		}

		queries := db.WithTx(tx)

		err := queries.InsertHistory(ctx, models.InsertHistoryParams{
			JobID:    job.ID.String(),
			Priority: int16(job.Priority),
			RunAt:    pgtype.Timestamptz{Time: job.RunAt, Valid: true, InfinityModifier: pgtype.Finite},
			JobType:  job.Type,
			Args:     job.Args,
			RunCount: job.ErrorCount,
			RunError: job.LastError.String,
			Queue:    job.Queue,
		})
		if err != nil {
			logger.InfoContext(ctx, "could not add started job to gue_jobs_history table",
				slog.Any("err", err),
				slog.String("job_id", job.ID.String()),
				slog.String("queue", job.Queue),
				slog.String("job_type", job.Type),
				slog.String("args", string(job.Args)),
				slog.Int("run_count", int(job.ErrorCount)),
				slog.Int("priority", int(job.Priority)),
				slog.Time("run_at", job.RunAt),
				slog.String("run_error", job.LastError.String),
			)
		}
	}
}

// recordFinishedJobsToHistory takes each job that's finished and logs it into a new table,
// so it's persisted for later analytics.
// gue does delete finished jobs from the gue_jobs table, and the information would be lost otherwise.
func recordFinishedJobsToHistory(logger alog.Logger, db *models.Queries) func(context.Context, *gue.Job, error) {
	return func(ctx context.Context, job *gue.Job, jobErr error) {
		logger = logger.With(
			slog.String("job_id", job.ID.String()),
			slog.String("queue", job.Queue),
			slog.String("job_type", job.Type),
			slog.String("args", string(job.Args)),
			slog.Int("run_count", int(job.ErrorCount)),
			slog.Int("priority", int(job.Priority)),
			slog.Time("run_at", job.RunAt),
		)

		tx, ok := pgxv5.UnwrapTx(job.Tx())
		if !ok {
			logger.InfoContext(ctx, "could not access transaction to record job in history")

			return
		}

		queries := db.WithTx(tx)

		if jobErr != nil { // job returned with an error and worker JobFunc failed
			err := queries.UpdateHistory(ctx, models.UpdateHistoryParams{
				RunError: jobErr.Error(),
				RunCount: job.ErrorCount,
				Success:  false,
				JobID:    job.ID.String(),
			})
			if err != nil {
				logger.InfoContext(ctx, "could not add failed job to gue_jobs_history table",
					slog.Any("err", err),
					slog.String("run_error", jobErr.Error()),
				)
			}

			return
		}

		err := queries.UpdateHistory(ctx, models.UpdateHistoryParams{
			RunError: "",
			RunCount: job.ErrorCount,
			Success:  true,
			JobID:    job.ID.String(),
		})
		if err != nil {
			logger.InfoContext(ctx, "could not add succeeded job to gue_jobs_history table",
				slog.Any("err", err),
				slog.String("run_error", ""),
			)
		}
	}
}

func (h *PostgresJobsHandler) Shutdown(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.shutdown(ctx)
}

// shutdown expects the locking of h.mu to happen at the caller!
func (h *PostgresJobsHandler) shutdown(ctx context.Context) error {
	if !h.hasStarted {
		return nil
	}

	// send shutdown signal to worker
	h.shutdownWorkerPool()

	if err := h.groupWorkerPool.Wait(); err != nil {
		return fmt.Errorf("%w: could not shutdown job workers: %v", ErrRegisterJobFuncFailed, err) //nolint:errorlint,lll // prevent err in api
	}

	if err := connOrTX(ctx, h.queries).UpsertWorkerToPool(ctx, models.UpsertWorkerToPoolParams{
		ID:        h.poolName,
		Queue:     h.queue,
		Version:   h.version,
		JobTypes:  registeredJobTypes(h.gueWorkMap),
		Workers:   0, // setting the number of workers to zero => indicator for the UI, that this pool has dropped out.
		UpdatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true, InfinityModifier: pgtype.Finite},
	}); err != nil {
		return fmt.Errorf("%w: could not unregister worker pool: %v", ErrRegisterJobFuncFailed, err) //nolint:errorlint,lll // prevent err in api
	}

	h.hasStarted = false

	return nil
}

// getJobTypeFromJobFunc returns the jobType as the struct name or takes it from JobType
// if that interface is implemented.
func getJobTypeFromJobFunc(jf JobFunc) string {
	jobFunc := reflect.TypeOf(jf)
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

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") //nolint:gochecknoglobals
func randomPoolName(n int) string {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec // used for ids, not security

	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rnd.Intn(len(letters))]
	}

	return string(b)
}

func connOrTX(ctx context.Context, queries *models.Queries) *models.Queries {
	if tx, ok := ctx.Value(postgres.CtxTX).(pgx.Tx); ok {
		return queries.WithTx(tx)
	}

	// in case no transaction is present return the default DB access.
	return queries
}
