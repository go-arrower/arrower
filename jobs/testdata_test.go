//go:build integration

// Valid and invalid testdata for all tests to use for convenience.
package jobs_test

import (
	"context"
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