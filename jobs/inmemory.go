package jobs

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// NewMemoryQueue is an in memory implementation of the Queue.
// No Jobs are persisted! Recommended use for local development and demos only.
func NewMemoryQueue() *MemoryQueue {
	q := newMemoryQueue()
	q.start(context.Background())

	return q
}

func newMemoryQueue() *MemoryQueue {
	return &MemoryQueue{
		modulePath: modulePath(),

		mu:        sync.Mutex{},
		jobs:      []Job{},
		workerMap: map[string]JobFunc{},

		cancel: func() {},

		cron: cron.New(cron.WithParser(
			cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor),
		)),
	}
}

type MemoryQueue struct { //nolint:govet // alignment less important than grouping of mutex
	modulePath string

	mu        sync.Mutex
	jobs      []Job
	workerMap map[string]JobFunc

	cancel context.CancelFunc

	cron *cron.Cron
}

var _ Queue = (*MemoryQueue)(nil)

func (q *MemoryQueue) Enqueue(_ context.Context, job Job, _ ...JobOption) error {
	err := ensureValidJobTypeForEnqueue(job)
	if err != nil {
		return err
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	switch reflect.ValueOf(job).Kind() { //nolint:exhaustive // other types are prevented by ensureValidJobTypeForEnqueue
	case reflect.Struct:
		q.jobs = append(q.jobs, job)
	case reflect.Slice:
		allJobs := reflect.ValueOf(job)
		for i := 0; i < allJobs.Len(); i++ {
			job := allJobs.Index(i)

			q.jobs = append(q.jobs, job.Interface())
		}
	}

	return nil
}

func (q *MemoryQueue) Schedule(spec string, job Job) error {
	err := ensureValidJobTypeForEnqueue(job)
	if err != nil {
		return err
	}

	_, err = q.cron.AddFunc(spec, func() {
		q.mu.Lock()
		defer q.mu.Unlock()

		q.jobs = append(q.jobs, job)
	})
	if err != nil {
		return fmt.Errorf("%w: could not schedule job: %v", ErrScheduleFailed, err) //nolint:errorlint,lll // prevent err in api
	}

	return nil
}

func (q *MemoryQueue) RegisterJobFunc(jf JobFunc) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	jobType, _, err := getJobTypeFromType(reflect.TypeOf(jf).In(1), q.modulePath)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	q.workerMap[jobType] = jf

	return nil
}

func (q *MemoryQueue) Shutdown(_ context.Context) error {
	q.cancel()

	wait := q.cron.Stop()
	<-wait.Done()

	return nil
}

// start processes the Jobs enqueued in this queue.
// It has to be started explicitly, so no Jobs are processed while asserting for tests.
func (q *MemoryQueue) start(ctx context.Context) {
	q.mu.Lock()
	defer q.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	q.cancel = cancel

	go q.runWorkers(ctx)
	go q.cron.Run()
}

func (q *MemoryQueue) runWorkers(ctx context.Context) {
	const tickerDuration = 10 * time.Millisecond
	interval := time.NewTicker(tickerDuration)

	for {
		select {
		case <-interval.C:
			q.processFirstJob()
		case <-ctx.Done(): // stop workers
			return
		}
	}
}

func (q *MemoryQueue) processFirstJob() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.jobs) == 0 {
		return
	}

	var job Job
	job, q.jobs = q.jobs[0], q.jobs[1:]

	jt, _, _ := getJobTypeFromType(reflect.TypeOf(job), q.modulePath)

	workerFn, exists := q.workerMap[jt]
	if !exists { // silently fail, if no JobFunc is registered
		return
	}

	// call the JobFunc
	fn := reflect.ValueOf(workerFn)
	vals := fn.Call([]reflect.Value{
		reflect.ValueOf(context.Background()),
		reflect.ValueOf(job),
	})

	// if JobFunc returned an error, put the job back on the queue.
	if len(vals) > 0 {
		if jobErr, ok := vals[0].Interface().(error); ok && jobErr != nil {
			q.jobs = append(q.jobs, job)
		}
	}
}

//// Queued asserts that of a given JobType exactly as many jobs are queued as expected.
//func (a *InMemoryAssertions) Queued(job Job, expCount int, msgAndArgs ...any) bool {
//	a.t.Helper()
//
//	expType, _, err := getJobTypeFromType(reflect.TypeOf(job), a.q.modulePath)
//	if err != nil {
//		return assert.Fail(a.t, "invalid jobType of given job: "+expType, msgAndArgs...)
//	}
//
//	jobsByType := map[string]int{}
//
//	for _, j := range a.q.jobs {
//		jobType, _, err := getJobTypeFromType(reflect.TypeOf(j), a.q.modulePath)
//		if err != nil {
//			return assert.Fail(a.t, "invalid jobType in queue: "+jobType, msgAndArgs...)
//		}
//
//		jobsByType[jobType]++
//	}
//
//	if jobsByType[expType] != expCount {
//		return assert.Fail(
//			a.t,
//			fmt.Sprintf("expected %d of type %s, got: %d", expCount, expType, jobsByType[expType]),
//			msgAndArgs...,
//		)
//	}
//
//	return true
//}
//
//// QueuedTotal asserts the total amount of Jobs in the queue, independent of their type.
//func (a *InMemoryAssertions) QueuedTotal(expCount int, msgAndArgs ...any) bool {
//	a.t.Helper()
//
//	if len(a.q.jobs) != expCount {
//		return assert.Fail(
//			a.t,
//			fmt.Sprintf("expected queue to have %d elements, but it has %d", expCount, len(a.q.jobs)),
//			msgAndArgs...,
//		)
//	}
//
//	return true
//}
