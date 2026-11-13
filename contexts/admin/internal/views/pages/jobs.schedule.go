package pages

import (
	"encoding/json"

	"github.com/go-arrower/arrower/contexts/admin/internal/application"
	"github.com/go-arrower/arrower/contexts/admin/internal/domain/jobs"
)

func PresentJobsExamplePayloads(queue, jobType string, payloads [][]byte) map[string]any {
	prettyPayloads := make([]string, len(payloads))

	for i, p := range payloads {
		var jobPayload application.JobPayload
		_ = json.Unmarshal(p, &jobPayload) //nolint:wsl_v5

		prettyPayloads[i] = prettyJobPayloadDataAsFormattedJSON(jobPayload)
	}

	if queue == "" {
		queue = string(jobs.DefaultQueueName)
	}

	return map[string]any{
		"Queue":    queue,
		"JobType":  jobType,
		"Payloads": prettyPayloads,
	}
}
