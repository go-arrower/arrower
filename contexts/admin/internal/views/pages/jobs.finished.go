package pages

import (
	"encoding/json"

	"github.com/labstack/echo/v4"

	"github.com/go-arrower/arrower/contexts/admin/internal/application"
	"github.com/go-arrower/arrower/contexts/admin/internal/domain/jobs"
)

func NewFinishedJobs(jobs []jobs.Job, queues jobs.QueueNames) echo.Map {
	type finishedJob struct {
		EnqueuedAtFmt string
		FinishedAtFmt string
		ID            string
		Type          string
		Queue         string
		Payload       string
	}

	fjobs := make([]finishedJob, len(jobs))

	for i := range jobs {
		var m application.JobPayload

		_ = json.Unmarshal([]byte(jobs[i].Payload), &m)

		fjobs[i].Payload = prettyJobPayloadDataAsFormattedJSON(m)
		fjobs[i].EnqueuedAtFmt = TimeAgo(jobs[i].CreatedAt)
		fjobs[i].FinishedAtFmt = TimeAgo(jobs[i].UpdatedAt) // todo use finished at
		fjobs[i].ID = jobs[i].ID
		fjobs[i].Type = string(jobs[i].Type)
		fjobs[i].Queue = string(jobs[i].Queue)
	}

	return echo.Map{
		"Jobs":   fjobs,
		"Queues": queues,
	}
}
