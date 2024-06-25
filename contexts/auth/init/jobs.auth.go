package init

import (
	"github.com/go-arrower/arrower/contexts/auth/internal/application"
	"github.com/go-arrower/arrower/jobs"
	"github.com/go-arrower/arrower/mw"
)

// registerJobs initialises all jobs to be run by this Context.
func (c *AuthContext) registerJobs(queue jobs.Queue) {
	_ = queue.RegisterJobFunc(mw.TracedU(c.traceProvider,
		mw.MetricU(c.meterProvider,
			mw.LoggedU(c.logger,
				application.SendNewUserVerificationEmail(c.logger, c.repo),
			),
		),
	))

}
