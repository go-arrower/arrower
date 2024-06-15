// Package init is the context's startup API.
//
// Put all initialisations here.
// For example, load context-specific configuration, setup dependency injection,
// register routes, workers and more.
package init

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/go-arrower/arrower"
	alogmodels "github.com/go-arrower/arrower/alog/models"
	"github.com/go-arrower/arrower/app"
	"github.com/go-arrower/arrower/contexts/admin/internal/application"
	"github.com/go-arrower/arrower/contexts/admin/internal/domain/jobs"
	"github.com/go-arrower/arrower/contexts/admin/internal/interfaces/repository"
	"github.com/go-arrower/arrower/contexts/admin/internal/interfaces/repository/models"
	"github.com/go-arrower/arrower/contexts/admin/internal/interfaces/web"
	"github.com/go-arrower/arrower/contexts/admin/internal/views"
)

const contextName = "admin"

func NewAdminContext(ctx context.Context, di *arrower.Container) (*AdminContext, error) {
	err := ensureRequiredDependencies(di)
	if err != nil {
		return nil, fmt.Errorf("missing dependencies to initialise context admin: %w", err)
	}

	admin, err := setupAdminContext(di)
	if err != nil {
		return nil, fmt.Errorf("could not initialise context admin: %w", err)
	}

	di.Logger.DebugContext(ctx, "context admin initialised")

	return admin, nil
}

type AdminContext struct {
	globalContainer *arrower.Container

	jobRepository jobs.Repository

	settingsController *web.SettingsController
	jobsController     *web.JobsController
	logsController     *web.LogsController
}

func (c *AdminContext) Shutdown(_ context.Context) error {
	return nil
}

func ensureRequiredDependencies(di *arrower.Container) error {
	if di.Logger == nil {
		return fmt.Errorf("%w: logger", arrower.ErrMissingDependency)
	}

	if di.PGx == nil {
		return fmt.Errorf("%w: pgx", arrower.ErrMissingDependency)
	}

	if di.WebRouter == nil {
		return fmt.Errorf("%w: web router", arrower.ErrMissingDependency)
	}

	if di.AdminRouter == nil {
		return fmt.Errorf("%w: admin router", arrower.ErrMissingDependency)
	}

	if di.WebRenderer == nil {
		return fmt.Errorf("%w: renderer", arrower.ErrMissingDependency)
	}

	if di.Settings == nil {
		return fmt.Errorf("%w: settings", arrower.ErrMissingDependency)
	}

	return nil
}

func setupAdminContext(di *arrower.Container) (*AdminContext, error) {
	logger := di.Logger.With(slog.String("context", contextName))

	jobRepository := repository.NewTracedJobsRepository(repository.NewPostgresJobsRepository(di.PGx))

	appDI := setupApplication(di, jobRepository)

	admin := &AdminContext{
		globalContainer: di,

		jobRepository: jobRepository,

		settingsController: web.NewSettingsController(di.AdminRouter),
		jobsController: web.NewJobsController(
			logger,
			models.New(di.PGx),
			jobRepository,
			application.NewLoggedJobsApplication(
				application.NewJobsApplication(repository.NewPostgresJobsRepository(di.PGx)),
				logger,
			),
			appDI,
		),
		logsController: web.NewLogsController(
			logger,
			di.Settings,
			alogmodels.New(di.PGx),
			di.AdminRouter.Group("/logs"),
		),
	}

	{ // add context-specific web views.
		var views fs.FS = views.AdminViews
		//if di.Config.Debug { // TODO fix when running debug from other repo than arrower
		//	fmt.Println(os.Getwd())
		//	fmt.Println(os.Getwd())
		//	fmt.Println(os.Getwd())
		//	views = os.DirFS("contexts/admin/internal/views")
		//}

		err := di.WebRenderer.AddContext(contextName, views)
		if err != nil {
			return nil, fmt.Errorf("could not add context views: %w", err)
		}

		err = di.WebRenderer.AddLayoutData(contextName, "default", func(ctx context.Context) (map[string]any, error) {
			return map[string]any{
				"Title": "arrower admin",
			}, nil
		})
		if err != nil {
			return nil, fmt.Errorf("could not add layout data: %w", err)
		}
	}

	registerAdminRoutes(admin)

	return admin, nil
}

func setupApplication(di *arrower.Container, jobRepository *repository.TracedJobsRepository) application.App {
	return application.App{
		PruneJobHistory: app.NewInstrumentedRequest(di.TraceProvider, di.MeterProvider, di.Logger,
			application.NewPruneJobHistoryRequestHandler(models.New(di.PGx)),
		),
		VacuumJobTable: app.NewInstrumentedRequest(di.TraceProvider, di.MeterProvider, di.Logger,
			application.NewVacuumJobTableRequestHandler(di.PGx),
		),
		DeleteJob: app.NewInstrumentedCommand(di.TraceProvider, di.MeterProvider, di.Logger,
			application.NewDeleteJobCommandHandler(jobRepository),
		),
		GetQueue: app.NewInstrumentedQuery(di.TraceProvider, di.MeterProvider, di.Logger,
			application.NewGetQueueQueryHandler(jobRepository),
		),
		GetWorkers: app.NewInstrumentedQuery(di.TraceProvider, di.MeterProvider, di.Logger,
			application.NewGetWorkersQueryHandler(jobRepository),
		),
		JobTypesForQueue: app.NewInstrumentedQuery(di.TraceProvider, di.MeterProvider, di.Logger,
			application.NewJobTypesForQueueQueryHandler(models.New(di.PGx)),
		),
		ListAllQueues: app.NewInstrumentedQuery(di.TraceProvider, di.MeterProvider, di.Logger,
			application.NewListAllQueuesQueryHandler(jobRepository),
		),
		ScheduleJobs: app.NewInstrumentedCommand(di.TraceProvider, di.MeterProvider, di.Logger,
			application.NewScheduleJobsCommandHandler(models.New(di.PGx)),
		),
	}
}
