package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vgarvardt/gue/v5"
	"github.com/vgarvardt/gue/v5/adapter"
	"github.com/vgarvardt/gue/v5/adapter/pgxv5"
	"github.com/vgarvardt/gueron/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"

	"github.com/go-arrower/arrower/alog"
	ctx2 "github.com/go-arrower/arrower/ctx"
	"github.com/go-arrower/arrower/jobs/models"
	"github.com/go-arrower/arrower/postgres"
)

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
	opts ...QueueOption,
) (*PostgresJobsHandler, error) {
	const (
		// defaults of gue, set it here, so it can be overwritten by QueueOption.
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
		logger:     logger,
		gueLogger:  gueLogger,
		meter:      meter,
		tracer:     tracer,
		propagator: propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}),
		queries:    models.New(pgxPool),
		gueClient:  nil, // has to be set after all opts have been applied
		gueWorkMap: gue.WorkMap{},
		queueOpt: queueOpt{
			pollInterval: defaultPollInterval,
			queue:        defaultQueue,
			poolName:     poolName,
			poolSize:     defaultPoolSize,
			pollStrategy: PriorityPollStrategy,
		},
		gitHash:            gitHash(),
		modulePath:         modulePath(),
		scheduler:          nil, // has to be set after all opts have benn applied
		shutdownWorkerPool: nil,
		groupWorkerPool:    nil,
		mu:                 sync.Mutex{},
		schedules:          []schedule{},
		hasStarted:         false,
		isStartInProgress:  false,
		startTimer:         nil,
	}

	// apply all options to the job
	for _, opt := range opts {
		opt(&handler.queueOpt)
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

	scheduler, err := gueron.NewScheduler(
		poolAdapter,
		gueron.WithQueueName(handler.queue),
		gueron.WithHorizon(time.Hour),
		gueron.WithLogger(gueLogger),
		gueron.WithMeter(meter),
		gueron.WithPollInterval(handler.pollInterval),
	)
	if err != nil {
		return nil, fmt.Errorf("could not create cron scheduler: %w", err)
	}

	handler.scheduler = scheduler

	return handler, nil
}

// PostgresJobsHandler is the main jobs' abstraction.
type PostgresJobsHandler struct { //nolint:govet // accept fieldalignment so the struct fields are grouped by meaning
	logger     alog.Logger
	gueLogger  adapter.Logger
	meter      metric.Meter
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator

	queries *models.Queries

	gueClient  *gue.Client
	gueWorkMap gue.WorkMap
	queueOpt
	gitHash    string
	modulePath string
	scheduler  *gueron.Scheduler

	shutdownWorkerPool context.CancelFunc
	groupWorkerPool    *errgroup.Group

	mu                sync.Mutex
	schedules         []schedule
	startTimer        *time.Timer
	isStartInProgress bool
	hasStarted        bool
}

type schedule struct {
	args    any
	spec    string
	jobType string
}

func gitHash() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value
			}
		}
	}

	return "unknown"
}

func modulePath() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		return info.Path
	}

	return ""
}

var _ Queue = (*PostgresJobsHandler)(nil)

func (h *PostgresJobsHandler) Enqueue(ctx context.Context, job Job, opts ...JobOption) error {
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

func (h *PostgresJobsHandler) Schedule(spec string, job Job) error {
	if reflect.TypeOf(job).Kind() != reflect.Struct {
		return ErrInvalidJobType
	}

	jobType, fullPath, err := getJobTypeFromType(reflect.TypeOf(job), h.modulePath)
	if err != nil {
		return fmt.Errorf("%w: could not get job type: %w", ErrScheduleFailed, err)
	}

	args, err := json.Marshal(PersistencePayload{
		JobStructPath:    fullPath,
		JobData:          job,
		GitHashEnqueued:  h.gitHash,
		GitHashProcessed: "",
		Ctx: PersistenceCTXPayload{
			UserID:  "",
			Carrier: nil,
		},
	})
	if err != nil {
		return fmt.Errorf("%w: could not marshal cron: %v", ErrScheduleFailed, err) //nolint:errorlint,lll // prevent err in api
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.schedules = append(h.schedules, schedule{
		spec:    spec,
		jobType: jobType,
		args:    job,
	})

	_, err = h.scheduler.Add(spec, jobType, args)
	if err != nil {
		return fmt.Errorf("%w: could not schedule cron: %v", ErrScheduleFailed, err) //nolint:errorlint,lll // prevent err in api
	}

	return nil
}

func (h *PostgresJobsHandler) gueJobsFromJob(
	ctx context.Context,
	queue string,
	job Job,
	opts ...JobOption,
) ([]*gue.Job, error) {
	gueJobs := []*gue.Job{}

	carrier := propagation.MapCarrier{}
	h.propagator.Inject(ctx, carrier)

	switch reflect.ValueOf(job).Kind() { //nolint:exhaustive // other types are prevented by ensureValidJobTypeForEnqueue
	case reflect.Struct:
		jobType, fullPath, err := getJobTypeFromType(reflect.TypeOf(job), h.modulePath)
		if err != nil {
			return nil, err
		}

		gueJobs, err = buildAndAppendGueJob(ctx, h.gitHash, gueJobs, queue,
			jobType, fullPath, job, carrier, opts...,
		)
		if err != nil {
			return nil, err
		}
	case reflect.Slice:
		allJobs := reflect.ValueOf(job)
		for i := 0; i < allJobs.Len(); i++ {
			job := allJobs.Index(i)

			jobType, fullPath, err := getJobTypeFromType(reflect.TypeOf(job.Interface()), h.modulePath)
			if err != nil {
				return nil, err
			}

			gueJobs, err = buildAndAppendGueJob(ctx, h.gitHash, gueJobs, queue,
				jobType, fullPath, job.Interface(), carrier, opts...,
			)
			if err != nil {
				return nil, err
			}
		}
	}

	return gueJobs, nil
}

func buildAndAppendGueJob(
	ctx context.Context,
	gitHash string,
	gueJobs []*gue.Job,
	queue string,
	jobType string,
	fullPath string,
	job any,
	carrier propagation.MapCarrier,
	opts ...JobOption,
) ([]*gue.Job, error) {
	var userID string
	userID, _ = ctx.Value(ctx2.CtxAuthUserID).(string)

	args, err := json.Marshal(PersistencePayload{
		JobStructPath:    fullPath,
		JobData:          job,
		GitHashEnqueued:  gitHash,
		GitHashProcessed: "",
		Ctx: PersistenceCTXPayload{
			Carrier: carrier,
			UserID:  userID,
		},
	})
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

// RegisterJobFunc registers new worker functions for a given JobType.
func (h *PostgresJobsHandler) RegisterJobFunc(jf JobFunc) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	ok := isValidJobFunc(jf)
	if !ok {
		return ErrInvalidJobFunc
	}

	jobType, _, err := getJobTypeFromType(reflect.TypeOf(jf).In(1), h.modulePath)
	if err != nil {
		return err
	}

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

func (h *PostgresJobsHandler) gueWorkerAdapter(workerFn JobFunc) gue.WorkFunc { //nolint:funlen
	handlerFuncType := reflect.TypeOf(workerFn)
	paramType := handlerFuncType.In(1)

	return func(ctx context.Context, job *gue.Job) error {
		payload, jobData, err := unmarshalArgsToJobPayload(paramType, job.Args)
		if err != nil {
			return err
		}

		// make the gue job's tx available in the context of the worker, so db can stay consistent
		txHandle, ok := pgxv5.UnwrapTx(job.Tx())
		if !ok {
			return fmt.Errorf("%w: could not unwrap gue job tx for use in the worker", ErrJobFuncFailed)
		}

		parentCtx := h.propagator.Extract(ctx, payload.Ctx.Carrier)

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
		ctx = context.WithValue(ctx, CTXJobID, job.ID.String())
		ctx = context.WithValue(ctx, postgres.CtxTX, txHandle)

		if payload.Ctx.UserID != "" {
			ctx = context.WithValue(ctx, ctx2.CtxAuthUserID, payload.Ctx.UserID)
		}

		_, err = txHandle.Exec(ctx, `SAVEPOINT before_worker;`)
		if err != nil {
			return fmt.Errorf("%w: could not create savepoint: %v", ErrJobFuncFailed, err) //nolint:errorlint,lll // prevent err in api
		}

		// call the JobFunc
		fn := reflect.ValueOf(workerFn)
		vals := fn.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(jobData).Elem().Convert(paramType),
		})

		// if JobFunc returned an error, put the job back on the queue.
		if len(vals) > 0 {
			if jobErr, ok := vals[0].Interface().(error); ok && jobErr != nil {
				childSpan.SetStatus(codes.Error, jobErr.Error())

				_, err = txHandle.Exec(ctx, `ROLLBACK TO before_worker;`)
				if err != nil {
					return fmt.Errorf("%w: could not roll back to savepoint: %v", ErrJobFuncFailed, err) //nolint:errorlint,lll // prevent err in api
				}

				return fmt.Errorf("%w: %s", ErrJobFuncFailed, jobErr) //nolint:errorlint // prevent err in api
			}
		}

		_, err = txHandle.Exec(ctx, `RELEASE SAVEPOINT before_worker;`)
		if err != nil {
			return fmt.Errorf("%w: could not release savepoint: %v", ErrJobFuncFailed, err) //nolint:errorlint,lll // prevent err in api
		}

		return nil
	}
}

func unmarshalArgsToJobPayload(paramType reflect.Type, rawArgs []byte) (PersistencePayload, any, error) {
	args := reflect.New(paramType)
	argsP := args.Interface()

	payload := PersistencePayload{} //nolint:exhaustruct

	if err := json.Unmarshal(rawArgs, &payload); err != nil {
		return PersistencePayload{},
			nil,
			fmt.Errorf("%w: could not unmarshal job args to job type: %w", ErrJobFuncFailed, err)
	}

	// encode to json to then unmarshal it into the target struct type.
	// payload.JobData is of type map[string]interface {} and cannot be cast to the target type using reflection.
	buf, err := json.Marshal(payload.JobData)
	if err != nil {
		return PersistencePayload{},
			nil,
			fmt.Errorf("%w: could not convert job data to target job type struct: %v", ErrJobFuncFailed, err) //nolint:errorlint,lll // prevent err in api
	}

	err = json.Unmarshal(buf, argsP)
	if err != nil {
		return PersistencePayload{},
			nil,
			fmt.Errorf("%w: could not convert job data to target job type struct: %v", ErrJobFuncFailed, err) //nolint:errorlint,lll // prevent err in api
	}

	return payload, argsP, nil
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
		gue.WithPoolHooksJobLocked(recordStartedJobsToHistory(h.logger, h.queries, h.gitHash)),
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
	group, gctx := errgroup.WithContext(ctx)

	go h.continuouslyRegisterInstance(gctx)

	// work jobs in goroutine
	group.Go(func() error {
		err := workers.Run(gctx)
		if err != nil {
			return fmt.Errorf("gue worker failed: %w", err)
		}

		return nil
	})

	// start scheduler in goroutine
	group.Go(func() error {
		err := h.scheduler.Run(gctx, h.gueWorkMap, 0) // zero => use pool of queue above instead
		if err != nil {
			return fmt.Errorf("gueron worker failed: %w", err)
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

// continuouslyRegisterInstance registers the worker pool and
// the schedules regularly, so it stays "active" for monitoring in the
// Arrower admin dashboard.
func (h *PostgresJobsHandler) continuouslyRegisterInstance(ctx context.Context) {
	const refreshDuration = 30 * time.Second

	h.registerInstance(ctx)

	for {
		select {
		case <-time.NewTicker(refreshDuration).C:
			h.registerInstance(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// registerInstance writes a "life probe" to the database indicating this
// instance is still active.
func (h *PostgresJobsHandler) registerInstance(ctx context.Context) {
	err := connOrTX(ctx, h.queries).UpsertWorkerToPool(ctx, models.UpsertWorkerToPoolParams{
		ID:        h.poolName,
		Queue:     h.queue,
		GitHash:   h.gitHash,
		JobTypes:  registeredJobTypes(h.gueWorkMap),
		Workers:   int16(h.poolSize),
		UpdatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true, InfinityModifier: pgtype.Finite},
	})
	if err != nil {
		h.logger.InfoContext(ctx, "could not save worker pool life probe to the database", slog.Any("err", err))
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for _, schedule := range h.schedules {
		args, err := json.Marshal(schedule.args)
		if err != nil {
			h.logger.InfoContext(ctx, "could not marshal schedule payload", slog.Any("err", err))
		}

		err = connOrTX(ctx, h.queries).UpsertSchedule(ctx, models.UpsertScheduleParams{
			Queue:     h.queue,
			Spec:      schedule.spec,
			JobType:   schedule.jobType,
			Args:      args,
			UpdatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true, InfinityModifier: pgtype.Finite},
		})
		if err != nil {
			h.logger.InfoContext(ctx, "could not save schedule life probe to the database", slog.Any("err", err))
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

func recordStartedJobsToHistory(
	logger alog.Logger,
	db *models.Queries,
	gitHash string,
) func(context.Context, *gue.Job, error) {
	return func(ctx context.Context, job *gue.Job, jobErr error) {
		// if jobErr is set, the job could not be pulled from the DB.
		if jobErr != nil {
			return
		}

		logger := logger.With(
			slog.String("job_id", job.ID.String()),
			slog.String("queue", job.Queue),
			slog.String("job_type", job.Type),
			slog.String("args", string(job.Args)),
			slog.Int("run_count", int(job.ErrorCount)),
			slog.Int("priority", int(job.Priority)),
			slog.Time("run_at", job.RunAt),
		)

		args, err := recordGitHashToArgs(job.Args, gitHash)
		if err != nil {
			logger.InfoContext(ctx, "recording job worker's git hash failed", slog.String("err", err.Error()))

			return
		}

		job.Args = args

		tx, ok := pgxv5.UnwrapTx(job.Tx())
		if !ok {
			logger.InfoContext(ctx, "could not access transaction to record job in history")

			return
		}

		queries := db.WithTx(tx)

		err = queries.InsertHistory(ctx, models.InsertHistoryParams{
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
				slog.String("run_error", job.LastError.String),
			)
		}
	}
}

// recordGitHashToArgs records the version of arrower when the job is processed for better debugging.
func recordGitHashToArgs(args []byte, gitHash string) ([]byte, error) {
	payload := PersistencePayload{} //nolint:exhaustruct

	if len(args) > 0 {
		err := json.Unmarshal(args, &payload)
		if err != nil {
			return nil, fmt.Errorf("could not unmarshal job data to record processing arrower version: %w", err)
		}
	} else {
		payload.JobData = struct{}{} // if job has no args, prevent `null` and set to `{}` in marshalled json
	}

	payload.GitHashProcessed = gitHash

	args, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("could not marshal job data to record processing arrower version: %w", err)
	}

	return args, nil
}

// recordFinishedJobsToHistory takes each job that's finished and logs it into a new table,
// so it's persisted for later analytics.
// gue does delete finished jobs from the gue_jobs table, and the information would be lost otherwise.
func recordFinishedJobsToHistory(logger alog.Logger, db *models.Queries) func(context.Context, *gue.Job, error) {
	return func(ctx context.Context, job *gue.Job, jobErr error) {
		logger := logger.With(
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
		GitHash:   h.gitHash,
		JobTypes:  registeredJobTypes(h.gueWorkMap),
		Workers:   0, // setting the number of workers to zero => indicator for the UI, that this pool has dropped out.
		UpdatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true, InfinityModifier: pgtype.Finite},
	}); err != nil {
		return fmt.Errorf("%w: could not unregister worker pool: %v", ErrRegisterJobFuncFailed, err) //nolint:errorlint,lll // prevent err in api
	}

	h.hasStarted = false

	return nil
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

// getJobTypeFromType returns the sanitised and short version as JobType
// and a full path of the Job struct in the form of PkgPath.Name.
//
// For example, github.com/go-arrower/arrower/contexts/auth/internal/application.NewUserVerificationEmail
// will become: auth/application.NewUserVerificationEmail.
func getJobTypeFromType(job reflect.Type, basePath string) (string, string, error) {
	fullPath := job.PkgPath() + "." + job.Name()
	arrowerFreefullPath := strings.TrimPrefix(fullPath, "github.com/go-arrower/arrower/")

	jobType := strings.TrimPrefix(arrowerFreefullPath, strings.TrimSuffix(basePath, "/")+"/")
	jobType = strings.TrimPrefix(jobType, "contexts/")
	jobType = strings.Replace(jobType, "internal/", "", 1)

	jobTypeInterfaceType := reflect.TypeOf((*JobType)(nil)).Elem()

	if job.Implements(jobTypeInterfaceType) {
		paramType := reflect.New(job)
		paramTypeP := paramType.Interface()

		if t, ok := paramTypeP.(JobType); ok {
			jobType = t.JobType()
		}
	}

	if jobType == "" {
		return "", "", ErrInvalidJobType
	}

	return jobType, fullPath, nil
}
