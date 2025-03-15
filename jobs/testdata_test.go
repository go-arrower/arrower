// Valid and invalid testdata for all tests to use for convenience.
package jobs_test

import (
	"context"
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
	return "arrower/jobs_test.simpleJob"
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
