// Valid and invalid testdata for all tests to use for convenience.
package jobs_test

import (
	"bytes"
	"context"
	"sync"
	"time"
)

var (
	ctx        = context.Background()
	argName    = "testName"
	payloadJob = jobWithArgs{Name: argName}
)

type (
	simpleJob struct{}

	jobWithSameNameAsSimpleJob struct{}

	jobWithArgs struct {
		Name string `json:"name"`
	}

	jobWithJobType struct {
		Name string
	}
)

func (j jobWithJobType) JobType() string {
	return "custom.job.type"
}

func (j jobWithSameNameAsSimpleJob) JobType() string {
	return "github.com/go-arrower/arrower/jobs_test.simpleJob"
}

type gueJobHistory struct {
	JobID      string
	JobType    string
	Queue      string
	RunError   string
	RunAt      time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
	FinishedAt *time.Time
	PrunedAt   *time.Time
	Args       []byte
	RunCount   int
	Priority   int
	Success    bool
}

// syncBuffer is a thread safe implementation of a bytes.Buffer. You can use it for a alog.Logger,
// to write to it in parallel.
type syncBuffer struct {
	b bytes.Buffer
	m sync.Mutex
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.m.Lock()
	defer b.m.Unlock()

	return b.b.Write(p) //nolint:wrapcheck // keep the original error unwrapped, as this is just a test helper.
}

func (b *syncBuffer) String() string {
	b.m.Lock()
	defer b.m.Unlock()

	return b.b.String()
}
