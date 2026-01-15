package pages

import (
	"encoding/json"

	"github.com/labstack/echo/v4"

	"github.com/go-arrower/arrower/contexts/admin/internal/application"
	"github.com/go-arrower/arrower/contexts/admin/internal/domain/jobs"
)

func PresentJobsExamplePayloads(queue, jobType string, payloads [][]byte) echo.Map {
	prettyPayloads := make([]string, len(payloads))

	for i, p := range payloads {
		var jobPayload application.JobPayload
		_ = json.Unmarshal(p, &jobPayload) //nolint:wsl_v5

		prettyPayloads[i] = prettyJobPayloadDataAsFormattedJSON(jobPayload)
	}

	if queue == "" {
		queue = string(jobs.DefaultQueueName)
	}

	return echo.Map{
		"Queue":    queue,
		"JobType":  jobType,
		"Payloads": prettyPayloads,
	}
}
