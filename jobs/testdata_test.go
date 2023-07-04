//go:build integration

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
		Name string
	}

	jobWithJobType struct {
		Name string
	}
)

func (j jobWithJobType) JobType() string {
	return "custom.job.type"
}

func (j jobWithSameNameAsSimpleJob) JobType() string {
	return "simpleJob"
}

var emptyWorkerFunc = func(context.Context, simpleJob) error { return nil } // todo remove OR use in more test cases

type GueJobHistory struct {
	JobID      string
	JobType    string
	Queue      string
	RunError   string
	RunAt      time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
	FinishedAt *time.Time
	Args       []byte
	RunCount   int
	Priority   int
	Success    bool
}
