package init

import (
	"github.com/go-arrower/arrower/app"
	"github.com/go-arrower/arrower/contexts/auth/internal/application"
	"github.com/go-arrower/arrower/jobs"
)

// registerJobs initialises all jobs to be run by this Context.
func (c *AuthContext) registerJobs(queue jobs.Queue) {
	_ = queue.RegisterJobFunc(app.NewInstrumentedJob(
		c.traceProvider, c.meterProvider, c.logger,
		application.NewSendNewUserVerificationEmailJobHandler(c.logger, c.repo),
	).H)
}
