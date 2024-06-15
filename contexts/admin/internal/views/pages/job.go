package pages

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/go-arrower/arrower/contexts/admin/internal/interfaces/repository/models"
)

func NewJob(jobs []models.ArrowerGueJobsHistory) Job {
	if len(jobs) == 0 {
		return Job{
			Jobs:        nil,
			ShowActions: false,
		}
	}

	showAction := false
	if !jobs[0].Success {
		showAction = true
	}

	return Job{
		Jobs:        jobs,
		ShowActions: showAction,
	}
}

type (
	Jobs []models.ArrowerGueJobsHistory
	Job  struct {
		Jobs // either way works
		// Jobs []models.ArrowerGueJobsHistory

		ShowActions bool
	}
)

// TimelineTime could reuse a shared timeFMT function, so it is coherent across pages.
func (j Job) TimelineTime(t pgtype.Timestamptz) string {
	isToday := t.Time.Format("2006.01.02") == time.Now().Format("2006.01.02")
	if isToday {
		return t.Time.Format("03:04")
	}

	return t.Time.Format("2006.01.02 03:04")
}
