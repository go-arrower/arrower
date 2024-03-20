package jobs

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// NewTestingJobs is an im memory implementation of the Queue, specialised on unit testing.
// Use the Assert() method in your testing.
func NewTestingJobs() *InMemoryQueue {
	return &InMemoryQueue{
		modulePath: modulePath(),

		mu:        sync.Mutex{},
		jobs:      []Job{},
		workerMap: map[string]JobFunc{},

		cancel: func() {},
	}
}

// NewInMemoryJobs is an in memory implementation of the Queue.
// Use it for local development and demos only, as no Jobs are persisted.
func NewInMemoryJobs() *InMemoryQueue {
	q := NewTestingJobs()
	q.Start(context.Background())

	return q
}

type InMemoryQueue struct { //nolint:govet // alignment less important than grouping of mutex
	modulePath string

	mu        sync.Mutex
	jobs      []Job
	workerMap map[string]JobFunc

	cancel context.CancelFunc
}

var _ Queue = (*InMemoryQueue)(nil)

func (q *InMemoryQueue) Enqueue(_ context.Context, job Job, _ ...JobOpt) error {
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

func (q *InMemoryQueue) RegisterJobFunc(jf JobFunc) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	jobType, _, err := getJobTypeFromType(reflect.TypeOf(jf).In(1), q.modulePath)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	q.workerMap[jobType] = jf

	return nil
}

func (q *InMemoryQueue) Shutdown(_ context.Context) error {
	q.cancel()

	return nil
}

// Reset resets the queue to be empty.
func (q *InMemoryQueue) Reset() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.jobs = []Job{}
}

// GetFirst returns the first Job in the queue or nil if the queue is empty.
// The Job stays in the queue.
func (q *InMemoryQueue) GetFirst() Job { //nolint:ireturn // fp
	return q.Get(1)
}

// Get returns the pos'th Job in the queue or nil if the queue is empty.
// The Job stays in the queue.
func (q *InMemoryQueue) Get(pos int) Job { //nolint:ireturn // fp
	if len(q.jobs) == 0 || pos > len(q.jobs) {
		return nil
	}

	for i, j := range q.jobs {
		if i+1 == pos {
			return j
		}
	}

	return nil
}

// GetFirstOf returns the first Job of the same type as the given job or nil if the queue is empty.
// The Job stays in the queue.
func (q *InMemoryQueue) GetFirstOf(job Job) Job { //nolint:ireturn // fp
	return q.GetOf(job, 1)
}

// GetOf returns the pos'th Job of the same type as the given job or nil if the queue is empty.
// The Job stays in the queue.
func (q *InMemoryQueue) GetOf(job Job, pos int) Job { //nolint:ireturn // fp
	if len(q.jobs) == 0 {
		return nil
	}

	searchType, _, err := getJobTypeFromType(reflect.TypeOf(job), q.modulePath)
	if err != nil {
		return nil
	}

	matchPos := 1

	for _, sJob := range q.jobs {
		jobType, _, err := getJobTypeFromType(reflect.TypeOf(sJob), q.modulePath)
		if err != nil {
			return nil
		}

		if jobType == searchType {
			if matchPos == pos {
				return sJob
			}

			matchPos++
		}
	}

	return nil
}

// Assert returns a new InMemoryAssertions for the specified testing.T for your convenience.
func (q *InMemoryQueue) Assert(t *testing.T) *InMemoryAssertions {
	t.Helper()

	return &InMemoryAssertions{
		q: q,
		t: t,
	}
}

// Start processes the Jobs enqueued in this queue.
// It has to be started explicitly, so no Jobs are processed while asserting for tests.
func (q *InMemoryQueue) Start(ctx context.Context) {
	q.mu.Lock()
	defer q.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	q.cancel = cancel

	go q.runWorkers(ctx)
}

func (q *InMemoryQueue) runWorkers(ctx context.Context) {
	interval := time.NewTicker(time.Second)

	for {
		select {
		case <-interval.C:
			q.processFirstJob()
		case <-ctx.Done(): // stop workers
			return
		}
	}
}

func (q *InMemoryQueue) processFirstJob() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.jobs) == 0 {
		return
	}

	var job Job
	job, q.jobs = q.jobs[0], q.jobs[1:]

	jt, _, _ := getJobTypeFromType(reflect.TypeOf(job), q.modulePath)
	workerFn := q.workerMap[jt]

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

// InMemoryAssertions is a helper that exposes a lot of Queue-specific assertions for the use in tests.
// The interface follows stretchr/testify as close as possible.
//
//   - Every assert func returns a bool indicating whether the assertion was successful or not,
//     this is useful for if you want to go on making further assertions under certain conditions.
type InMemoryAssertions struct {
	q *InMemoryQueue
	t *testing.T
}

// Empty asserts that the queue has no pending Jobs.
func (a *InMemoryAssertions) Empty(msgAndArgs ...any) bool {
	a.t.Helper()

	if len(a.q.jobs) > 0 {
		return assert.Fail(a.t, fmt.Sprintf("queue is not empty, it has: %d jobs", len(a.q.jobs)), msgAndArgs...)
	}

	return true
}

// NotEmpty asserts that the queue has at least one pending Job.
func (a *InMemoryAssertions) NotEmpty(msgAndArgs ...any) bool {
	a.t.Helper()

	if len(a.q.jobs) == 0 {
		return assert.Fail(a.t, "queue is empty, should not be", msgAndArgs...)
	}

	return true
}

// Queued asserts that of a given JobType exactly as many jobs are queued as expected.
func (a *InMemoryAssertions) Queued(job Job, expCount int, msgAndArgs ...any) bool {
	a.t.Helper()

	expType, _, err := getJobTypeFromType(reflect.TypeOf(job), a.q.modulePath)
	if err != nil {
		return assert.Fail(a.t, "invalid jobType of given job: "+expType, msgAndArgs...)
	}

	jobsByType := map[string]int{}

	for _, j := range a.q.jobs {
		jobType, _, err := getJobTypeFromType(reflect.TypeOf(j), a.q.modulePath)
		if err != nil {
			return assert.Fail(a.t, "invalid jobType in queue: "+jobType, msgAndArgs...)
		}

		jobsByType[jobType]++
	}

	if jobsByType[expType] != expCount {
		return assert.Fail(
			a.t,
			fmt.Sprintf("expected %d of type %s, got: %d", expCount, expType, jobsByType[expType]),
			msgAndArgs...,
		)
	}

	return true
}

// QueuedTotal asserts the total amount of Jobs in the queue, independent of their type.
func (a *InMemoryAssertions) QueuedTotal(expCount int, msgAndArgs ...any) bool {
	a.t.Helper()

	if len(a.q.jobs) != expCount {
		return assert.Fail(
			a.t,
			fmt.Sprintf("expected queue to have %d elements, but it has %d", expCount, len(a.q.jobs)),
			msgAndArgs...,
		)
	}

	return true
}
