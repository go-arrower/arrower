package init

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-arrower/arrower"
	"github.com/go-arrower/arrower/app"
	"github.com/go-arrower/arrower/contexts"
	"github.com/go-arrower/arrower/contexts/auth"
	"github.com/go-arrower/arrower/contexts/auth/internal/application"
	"github.com/go-arrower/arrower/contexts/auth/internal/domain"
	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/repository"
	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/repository/models"
	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/web"
	"github.com/go-arrower/arrower/contexts/auth/internal/views"
	"github.com/go-arrower/arrower/mw"
	"github.com/go-arrower/arrower/setting"
)

const contextName = "auth"

func NewAuthContext(di *arrower.Container) (*AuthContext, error) {
	if di == nil {
		return nil, fmt.Errorf("%w: missing dependency container", contexts.ErrInitialisationFailed)
	}

	if err := di.EnsureAllDependenciesPresent(); err != nil {
		return nil, fmt.Errorf("could not initialise auth context: %w", err)
	}

	logger := di.Logger.WithGroup(contextName)
	meter := di.MeterProvider.Meter(fmt.Sprintf("%s/%s", di.Config.ApplicationName, contextName))
	tracer := di.TraceProvider.Tracer(fmt.Sprintf("%s/%s", di.Config.ApplicationName, contextName))
	_ = meter
	_ = tracer

	if di.Config.Debug {
		// ../arrower/ is to access the views from the skeleton project for local development
		_ = di.WebRenderer.AddContext(contextName, os.DirFS("../arrower/contexts/auth/internal/views"))
	} else {
		err := di.WebRenderer.AddContext(contextName, views.AuthViews)
		if err != nil {
			return nil, fmt.Errorf("could not load auth views: %w", err)
		}
	}

	err := di.WebRenderer.AddLayoutData(contextName, "default", func(ctx context.Context) (map[string]any, error) {
		return map[string]any{
			"Title": strings.Title(contextName),
		}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not add layout data: %w", err)
	}

	{ // register default auth settings
		_ = di.Settings.Save(context.Background(), auth.SettingAllowRegistration, setting.NewValue(true))
		_ = di.Settings.Save(context.Background(), auth.SettingAllowLogin, setting.NewValue(true))

		//_ = di.Settings.Add(context.Background(), admin.Setting{
		//	Key:   admin.SettingAllowRegistration,
		//	Value: admin.NewSettingValue(true),
		//	UIOptions: admin.Options{
		//		Type:         admin.Checkbox,
		//		Label:        "Enable Registration",
		//		Info:         "Allows new Users to register themselves",
		//		DefaultValue: admin.NewSettingValue(true),
		//		ReadOnly:     false,
		//		Danger:       false,
		//	},
		//})
		//di.Settings.Add(context.Background(), admin.Setting{
		//	Key:   admin.SettingAllowLogin,
		//	Value: admin.NewSettingValue(true),
		//	UIOptions: admin.Options{
		//		Type:         admin.Checkbox,
		//		Label:        "Enable Login",
		//		Info:         "Allows Users to login to the application",
		//		DefaultValue: admin.NewSettingValue(true),
		//		ReadOnly:     false,
		//		Danger:       false,
		//	},
		//})
	}

	queries := models.New(di.PGx)
	repo, _ := repository.NewPostgresRepository(di.PGx)
	registrator := domain.NewRegistrationService(di.Settings, repo)

	webRoutes := di.WebRouter.Group(fmt.Sprintf("/%s", contextName))
	adminRouter := di.AdminRouter.Group(fmt.Sprintf("/%s", contextName))

	uc := application.UserApplication{
		RegisterUser: app.NewInstrumentedRequest(
			di.TraceProvider, di.MeterProvider, logger,
			app.NewValidatedRequest(nil,
				application.NewRegisterUserRequestHandler(logger, repo, registrator, di.ArrowerQueue),
			)),
		LoginUser: app.NewInstrumentedRequest(
			di.TraceProvider, di.MeterProvider, logger,
			app.NewValidatedRequest(
				nil,
				application.NewLoginUserRequestHandler(logger, repo, di.ArrowerQueue, domain.NewAuthenticationService(di.Settings)),
			)),
		ListUsers: app.NewInstrumentedQuery(di.TraceProvider, di.MeterProvider, logger, application.NewListUsersQueryHandler(repo)),
		ShowUser:  app.NewInstrumentedQuery(di.TraceProvider, di.MeterProvider, logger, application.NewShowUserQueryHandler(repo)),
	}

	userController := web.NewUserController(uc, webRoutes, []byte("secret"), di.Settings)
	userController.Queries = queries
	userController.CmdNewUser = mw.TracedU(di.TraceProvider,
		mw.MetricU(di.MeterProvider,
			mw.LoggedU(logger,
				mw.ValidateU(nil,
					application.NewUser(repo, registrator),
				),
			),
		),
	)
	userController.CmdVerifyUser = mw.TracedU(di.TraceProvider,
		mw.MetricU(di.MeterProvider,
			mw.LoggedU(logger,
				mw.ValidateU(nil,
					application.VerifyUser(repo),
				),
			),
		),
	)
	userController.CmdBlockUser = mw.Traced(di.TraceProvider,
		mw.Metric(di.MeterProvider,
			mw.Logged(logger,
				mw.Validate(nil,
					application.BlockUser(repo),
				),
			),
		),
	)
	userController.CmdUnBlockUser = mw.Traced(di.TraceProvider,
		mw.Metric(di.MeterProvider,
			mw.Logged(logger,
				mw.Validate(nil,
					application.UnblockUser(repo),
				),
			),
		),
	)

	authContext := AuthContext{
		settingsController: web.NewSettingsController(queries),
		userController:     userController,
		logger:             logger,
		traceProvider:      di.TraceProvider,
		meterProvider:      di.MeterProvider,
		queries:            queries,
		repo:               repo,
	}

	authContext.registerWebRoutes(webRoutes)
	authContext.registerAPIRoutes(di.APIRouter)
	authContext.registerAdminRoutes(adminRouter, localDI{queries: queries}) // todo only, if admin context is present

	authContext.registerJobs(di.ArrowerQueue)

	return &authContext, nil
}

type AuthContext struct {
	settingsController *web.SettingsController
	userController     web.UserController

	logger        *slog.Logger
	traceProvider trace.TracerProvider
	meterProvider metric.MeterProvider
	queries       *models.Queries
	repo          domain.Repository
}

func (c *AuthContext) Shutdown(ctx context.Context) error {
	return nil
}

type localDI struct {
	queries *models.Queries
}
