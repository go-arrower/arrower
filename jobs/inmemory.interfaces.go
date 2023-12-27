package jobs

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// NewTestingJobs is an im memory implementation of the Queue, specialised on unit testing.
// Use the Assert() method in your testing.
func NewTestingJobs() *InMemoryHandler {
	return &InMemoryHandler{
		mu:   sync.Mutex{},
		jobs: []Job{},
	}
}

type InMemoryHandler struct {
	jobs []Job
	mu   sync.Mutex
}

var _ Queue = (*InMemoryHandler)(nil)

func (q *InMemoryHandler) Enqueue(_ context.Context, job Job, _ ...JobOpt) error {
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

func (q *InMemoryHandler) RegisterJobFunc(_ JobFunc) error {
	panic("implement me")
}

func (q *InMemoryHandler) Shutdown(_ context.Context) error {
	panic("implement me")
}

// Reset resets the queue to be empty.
func (q *InMemoryHandler) Reset() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.jobs = []Job{}
}

// GetFirst returns the first Job in the queue or nil if the queue is empty.
// The Job stays in the queue.
func (q *InMemoryHandler) GetFirst() Job { //nolint:ireturn // fp
	return q.Get(1)
}

// Get returns the pos'th Job in the queue or nil if the queue is empty.
// The Job stays in the queue.
func (q *InMemoryHandler) Get(pos int) Job { //nolint:ireturn // fp
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
func (q *InMemoryHandler) GetFirstOf(job Job) Job { //nolint:ireturn // fp
	return q.GetOf(job, 1)
}

// GetOf returns the pos'th Job of the same type as the given job or nil if the queue is empty.
// The Job stays in the queue.
func (q *InMemoryHandler) GetOf(job Job, pos int) Job { //nolint:ireturn // fp
	if len(q.jobs) == 0 {
		return nil
	}

	searchType, err := getJobTypeFromJobStruct(job)
	if err != nil {
		return nil
	}

	matchPos := 1

	for _, sJob := range q.jobs {
		jobType, err := getJobTypeFromJobStruct(sJob)
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
func (q *InMemoryHandler) Assert(t *testing.T) *InMemoryAssertions {
	t.Helper()

	return &InMemoryAssertions{
		q: q,
		t: t,
	}
}

// InMemoryAssertions is a helper that exposes a lot of Queue-specific assertions for the use in tests.
// The interface follows stretchr/testify as close as possible.
//
//   - Every assert func returns a bool indicating whether the assertion was successful or not,
//     this is useful for if you want to go on making further assertions under certain conditions.
type InMemoryAssertions struct {
	q *InMemoryHandler
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

	expType, err := getJobTypeFromJobStruct(job)
	if err != nil {
		return assert.Fail(a.t, fmt.Sprintf("invalid jobType of given job: %s", expType), msgAndArgs...)
	}

	jobsByType := map[string]int{}

	for _, j := range a.q.jobs {
		jobType, err := getJobTypeFromJobStruct(j)
		if err != nil {
			return assert.Fail(a.t, fmt.Sprintf("invalid jobType in queue: %s", jobType), msgAndArgs...)
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
