package jobs

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test returns a TestQueue tuned for unit testing.
func Test(t *testing.T) *TestQueue {
	if t == nil {
		panic("t is nil")
	}

	queue := newMemoryQueue()

	return &TestQueue{
		MemoryQueue: queue,
		TestAssertions: &TestAssertions{
			q: queue,
			t: t,
		},
	}
}

// TestQueue is a special Queue for unit testing.
// It exposes all methods of Queue and can be injected as a dependency
// in any application.
// Additionally, TestQueue exposes a set of assertions TestAssertions
// on all the jobs stored in the Queue.
type TestQueue struct {
	*MemoryQueue
	*TestAssertions
}

func (q *TestQueue) Jobs() []Job {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.jobs
}

// Clear resets the queue to be empty.
// todo test case or remove
func (q *TestQueue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.jobs = []Job{}
}

// GetFirst returns the first Job in the queue or nil if the queue is empty.
// The Job stays in the queue.
func (q *TestQueue) GetFirst() Job { //nolint:ireturn // fp
	return q.Get(1)
}

// Get returns the pos'th Job in the queue or nil if the queue is empty.
// The Job stays in the queue.
func (q *TestQueue) Get(pos int) Job { //nolint:ireturn // fp
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
func (q *TestQueue) GetFirstOf(job Job) Job { //nolint:ireturn // fp
	return q.GetOf(job, 1)
}

// GetOf returns the pos'th Job of the same type as the given job or nil if the queue is empty.
// The Job stays in the queue.
func (q *TestQueue) GetOf(job Job, pos int) Job { //nolint:ireturn // fp
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

// TestAssertions are assertions that work on a Queue, to make
// testing easier and convenient.
// The interface follows stretchr/testify as close as possible.
//
//   - Every assert func returns a bool indicating whether the assertion was successful or not,
//     this is useful for if you want to go on making further assertions under certain conditions.
type TestAssertions struct {
	q *MemoryQueue
	t *testing.T
}

// Empty asserts that the queue has no pending Jobs.
func (a *TestAssertions) Empty(msgAndArgs ...any) bool {
	a.t.Helper()

	if len(a.q.jobs) > 0 {
		return assert.Fail(a.t, fmt.Sprintf("queue is not empty, it has: %d jobs", len(a.q.jobs)), msgAndArgs...)
	}

	return true
}

// NotEmpty asserts that the queue has at least one pending Job.
func (a *TestAssertions) NotEmpty(msgAndArgs ...any) bool {
	a.t.Helper()

	if len(a.q.jobs) == 0 {
		return assert.Fail(a.t, "queue is empty, should not be", msgAndArgs...)
	}

	return true
}

// Total asserts that the queue has exactly total number of jobs pending.
func (a *TestAssertions) Total(total int, msgAndArgs ...any) bool {
	a.t.Helper()

	if len(a.q.jobs) != total {
		return assert.Fail(a.t, fmt.Sprintf("queue does not have %d jobs, it has: %d", total, len(a.q.jobs)), msgAndArgs...)
	}

	return true
}
