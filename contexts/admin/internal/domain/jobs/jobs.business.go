package jobs

import (
	"time"
)

const DefaultQueueName = QueueName("Default")

type (
	QueueName  string
	QueueNames []QueueName
)

type (
	JobType string

	// Job could also have "runs" how many times it was executed with error etc.
	Job struct {
		ID         string
		CreatedAt  time.Time
		UpdatedAt  time.Time
		RunAt      time.Time
		Type       JobType
		Queue      QueueName
		Payload    string
		LastError  string
		ErrorCount int32
		Priority   int16
	}

	FinishedJob struct {
		ID         string
		CreatedAt  time.Time
		UpdatedAt  time.Time
		RunAt      time.Time
		Type       JobType
		Queue      QueueName
		Payload    string
		Priority   int16
		RunCount   int32
		RunError   string
		Success    bool
		FinishedAt time.Time
		prunedAt   time.Time
	}

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

	QueueKPIs struct {
		PendingJobsPerType map[string]int
		PendingJobs        int
		FailedJobs         int
		ProcessedJobs      int
		AvailableWorkers   int
		AverageTimePerJob  time.Duration
	}
)

type (
	WorkerPool struct {
		InstanceName string
		Version      string
		Queue        QueueName
		LastSeenAt   time.Time
		JobTypes     []JobType
		Workers      int
	}

	Schedule struct {
		Queue   QueueName
		Spec    string
		JobType JobType
		Args    any
	}
)
