package jobs

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func NewInMemoryJobs() *InMemoryHandler {
	return &InMemoryHandler{jobs: []Job{}}
}

type InMemoryHandler struct {
	jobs []Job
}

var _ Queue = (*InMemoryHandler)(nil)

func (q *InMemoryHandler) Enqueue(ctx context.Context, job Job, opt ...JobOpt) error {
	q.jobs = append(q.jobs, job)

	return nil
}

func (q *InMemoryHandler) RegisterJobFunc(jobFunc JobFunc) error {
	//TODO implement me
	panic("implement me")
}

func (q *InMemoryHandler) Shutdown(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

// Assert returns a new InMemoryAssertions for the specified testing.T for your convenience.
func (q *InMemoryHandler) Assert(t *testing.T) *InMemoryAssertions {
	t.Helper()

	return &InMemoryAssertions{
		q: q,
		t: t,
	}
}

// InMemoryAssertions is a helper that exposes a lot of jobs specific assertions for the use in tests.
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

	if len(a.q.jobs) <= 0 {
		return assert.Fail(a.t, "queue is empty, shoudl not be", msgAndArgs...)
	}

	return true
}
