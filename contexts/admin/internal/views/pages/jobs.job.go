package pages

import (
	"github.com/go-arrower/arrower/contexts/admin/internal/interfaces/repository/models"
)

type HistoricJob struct {
	models.ArrowerGueJobsHistory
	PrettyPayload string
	CreatedAt     string
	EnqueuedAgo   string
	FinishedAgo   string
}

func ConvertFinishedJobsForShow(jobs []models.ArrowerGueJobsHistory) []HistoricJob {
	fjobs := make([]HistoricJob, len(jobs))

	for i, j := range jobs {
		fjobs[i] = HistoricJob{
			ArrowerGueJobsHistory: j,
			PrettyPayload:         prettyJobPayloadAsFormattedJSON(j.Args),
			CreatedAt:             formatAsDateOrTimeToday(j.CreatedAt.Time),
			EnqueuedAgo:           TimeAgo(j.CreatedAt.Time),
			FinishedAgo:           TimeAgo(j.FinishedAt.Time),
		}
	}

	return fjobs
}
