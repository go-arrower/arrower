package pages

import (
	"time"

	"github.com/go-arrower/skeleton/shared/domain"
)

type helloPage struct {
	domain.TeamMember
	TeamTimeFmt string
}

func PresentHello(tm domain.TeamMember) helloPage {
	if tm.Name == "" {
		tm.Name = "Welt"
	}

	return helloPage{
		TeamMember:  tm,
		TeamTimeFmt: tm.TeamTime.Format(time.TimeOnly),
	}
}

func (p helloPage) ShowTeamBanner() bool {
	return p.IsTeamMember
}
