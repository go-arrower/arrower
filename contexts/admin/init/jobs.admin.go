package init

import (
	"github.com/go-arrower/arrower/app"
	"github.com/go-arrower/arrower/contexts/admin/internal/application"
	"github.com/go-arrower/arrower/contexts/admin/internal/interfaces/repository/models"
)

func (c *AdminContext) registerJobs() {
	_ = c.shared.ArrowerQueue.RegisterJobFunc(app.NewInstrumentedJob(
		c.shared.TraceProvider, c.shared.MeterProvider, c.logger,
		application.NewPruneJobHistoryCronCommandHandler(c.logger, c.shared.Settings, models.New(c.shared.PGx)),
	).H)

	_ = c.shared.ArrowerQueue.Schedule("@daily", application.PruneJobHistoryCronCommand{})
}
