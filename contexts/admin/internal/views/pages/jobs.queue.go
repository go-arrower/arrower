package pages

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-arrower/arrower/contexts/admin/internal/application"
	"github.com/go-arrower/arrower/contexts/admin/internal/domain/jobs"
)

type (
	QueueStats struct {
		PendingJobsPerType   map[string]int
		QueueName            string
		PendingJobs          int
		FailedJobs           int
		ProcessedJobs        int
		AvailableWorkers     int
		PendingJobsErrorRate float64 // can be calculated: FailedJobs * 100 / PendingJobs
		AverageTimePerJob    time.Duration
		EstimateUntilEmpty   time.Duration // can be calculated
	}

	ListQueuesPage struct {
		Queues map[jobs.QueueName]jobs.QueueStats
	}

	QueuePage struct {
		Jobs      []viewJob
		QueueName string
		Stats     QueueStats
	}
)

func BuildQueuePage(queue string, jobs []jobs.Job, kpis jobs.QueueKPIs) QueuePage {
	vjobs := prettyFormatPayload(jobs)

	return QueuePage{
		QueueName: queue,
		Stats:     queueKpiToStats(queue, kpis),

		Jobs: vjobs,
	}
}

type viewJob struct {
	RunAtFmt string
	jobs.Job
}

func prettyFormatPayload(pJobs []jobs.Job) []viewJob {
	vJobs := make([]viewJob, len(pJobs))

	for i := range pJobs {
		var m application.JobPayload

		_ = json.Unmarshal([]byte(pJobs[i].Payload), &m)
		data, _ := json.Marshal(m.JobData)

		var prettyJSON bytes.Buffer

		_ = json.Indent(&prettyJSON, data, "", "  ")

		if pJobs[i].Queue == "" {
			pJobs[i].Queue = jobs.DefaultQueueName
		}

		pJobs[i].Payload = prettyJSON.String()

		vJobs[i] = viewJob{
			Job:      pJobs[i],
			RunAtFmt: fmtRunAtTime(pJobs[i].RunAt),
		}
	}

	return vJobs
}

func fmtRunAtTime(tme time.Time) string {
	now := time.Now()

	isToday := tme.Year() == now.Year() && tme.Month() == now.Month() && tme.Day() == now.Day()
	if isToday {
		return fmt.Sprintf("%02d:%02d", tme.Hour(), tme.Minute())
	}

	return tme.Format("2006.01.02 15:04")
}

func queueKpiToStats(queue string, kpis jobs.QueueKPIs) QueueStats {
	var errorRate float64

	if kpis.FailedJobs != 0 {
		errorRate = float64(kpis.FailedJobs * 100 / kpis.PendingJobs)
	}

	var duration time.Duration
	if kpis.AvailableWorkers != 0 {
		duration = time.Duration(kpis.PendingJobs/kpis.AvailableWorkers) * kpis.AverageTimePerJob
	}

	return QueueStats{
		QueueName:            queue,
		PendingJobs:          kpis.PendingJobs,
		PendingJobsPerType:   kpis.PendingJobsPerType,
		FailedJobs:           kpis.FailedJobs,
		ProcessedJobs:        kpis.ProcessedJobs,
		AvailableWorkers:     kpis.AvailableWorkers,
		PendingJobsErrorRate: errorRate,
		AverageTimePerJob:    kpis.AverageTimePerJob.Truncate(time.Second),
		EstimateUntilEmpty:   duration.Truncate(time.Second),
	}
}
