package jobs

import (
	"fmt"
	"time"

	"github.com/go-arrower/arrower/postgres"
)

var ErrJobLockedAlready = fmt.Errorf("%w: job might be processing already", postgres.ErrQueryFailed)

type (
	JobType string

	QueueStats struct { // todo return this from repo to prevent any mapping for trivial models like this
		QueueName            QueueName
		PendingJobs          int
		PendingJobsPerType   map[string]int
		FailedJobs           int
		ProcessedJobs        int
		AvailableWorkers     int
		PendingJobsErrorRate float64 // can be calculated: FailedJobs * 100 / PendingJobs
		AverageTimePerJob    time.Duration
		EstimateUntilEmpty   time.Duration // can be calculated
	}
)

type (
	PendingJob struct {
		CreatedAt  time.Time
		UpdatedAt  time.Time
		RunAt      time.Time
		RunAtFmt   string
		ID         string
		Type       string
		Queue      string
		Payload    string
		LastError  string
		ErrorCount int32
		Priority   int16
	}

	QueueKPIs struct {
		PendingJobsPerType map[string]int
		PendingJobs        int
		FailedJobs         int
		ProcessedJobs      int
		AvailableWorkers   int
		AverageTimePerJob  time.Duration
	}

	WorkerPool struct {
		LastSeen time.Time
		ID       string
		Queue    string // todo change type
		Workers  int
		Version  string
		JobTypes []string
	}
)
