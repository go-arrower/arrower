package arrower

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/jobs"
	"github.com/go-arrower/arrower/postgres"
	"github.com/go-arrower/arrower/renderer"
	"github.com/go-arrower/arrower/setting"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	prometheus2 "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc"

	"github.com/go-arrower/skeleton/contexts/auth"
	"github.com/go-arrower/skeleton/shared/views"
)

var ErrMissingDependency = errors.New("missing dependency")

// Container holds global dependencies that can be used within each Context, to make initialisation easier.
// If the Context can operate with the shared resources.
// Otherwise, the Context is advised to initialise its own dependencies from its own configuration.
type Container struct {
	Logger        alog.Logger
	MeterProvider *metric.MeterProvider
	TraceProvider *trace.TracerProvider

	Config *Config
	PGx    *pgxpool.Pool
	db     *postgres.Handler

	WebRenderer *renderer.EchoRenderer
	WebRouter   *echo.Echo
	APIRouter   *echo.Group
	AdminRouter *echo.Group

	ArrowerQueue jobs.Queue
	DefaultQueue jobs.Queue

	Settings setting.Settings
}

func (c *Container) EnsureAllDependenciesPresent() error {
	if c.Config == nil {
		return fmt.Errorf("%w: global config not found", ErrMissingDependency)
	}

	return nil
}

func InitialiseDefaultArrowerDependencies(ctx context.Context, conf *Config) (*Container, func(ctx context.Context) error, error) { //nolint:funlen,gocyclo,cyclop,lll // dependency injection is long but also straight forward.
	container := &Container{ //nolint:exhaustruct
		Config: conf,
	}

	{ // observability
		resource := resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(fmt.Sprintf("%s.%s", conf.OrganisationName, conf.ApplicationName)),

			// NEEDS TO MATCH WITH THE LOGS LABEL (why? for the "Logs for this span" button in tempo?)
			attribute.String(conf.OrganisationName, conf.ApplicationName),

			// more attributes like e.g. kubernetes pod name
		)
		// FIND OUT: CAN RESOURCES BE ADDED TO LOGGER SO ALL THREE HAVE THE SAME VALUES?

		{
			opts := []otlptracegrpc.Option{
				otlptracegrpc.WithEndpoint(fmt.Sprintf("%s:%d", conf.OTEL.Host, conf.OTEL.Port)),
			}
			if conf.Debug {
				opts = append(opts,
					otlptracegrpc.WithInsecure(),
					otlptracegrpc.WithDialOption(grpc.WithBlock()),
				)
			}

			traceExporter, err := otlptracegrpc.New(ctx, opts...)
			if err != nil {
				return nil, nil, fmt.Errorf("could not connect to trace exporter: %w", err)
			}

			traceProvider := trace.NewTracerProvider(
				trace.WithBatcher(traceExporter), // prod
				trace.WithResource(resource),
				// set the sampling rate based on the parent span to 60%
				trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(0.6))),
			)
			if conf.Debug {
				traceProvider = trace.NewTracerProvider(
					trace.WithSyncer(traceExporter),
					trace.WithResource(resource),
					trace.WithSampler(trace.AlwaysSample()),
				)
			}

			container.TraceProvider = traceProvider
			// otel.SetTracerProvider(traceProvider)
		}

		{
			exporter, err := prometheus.New()
			if err != nil {
				return nil, nil, fmt.Errorf("could not create to prometheus exporter: %w", err)
			}

			meterProvider := metric.NewMeterProvider(
				metric.WithResource(resource),
				metric.WithReader(exporter),
			)

			container.MeterProvider = meterProvider
			// otel.SetMeterProvider(meterProvider)
		}
	}

	{ // postgres
		pg, err := postgres.ConnectAndMigrate(ctx, postgres.Config{
			User:       conf.Postgres.User,
			Password:   conf.Postgres.Password.Secret(),
			Database:   conf.Postgres.Database,
			Host:       conf.Postgres.Host,
			Port:       conf.Postgres.Port,
			SSLMode:    conf.Postgres.SSLMode,
			MaxConns:   conf.Postgres.MaxConns,
			Migrations: postgres.ArrowerDefaultMigrations,
		}, container.TraceProvider)
		if err != nil {
			return nil, nil, fmt.Errorf("could not connect to postgres: %w", err)
		}

		container.PGx = pg.PGx
		container.db = pg
	}

	container.Settings = setting.NewPostgresSettings(container.PGx)

	logger := alog.New()
	logger = logger.With(
		slog.String("organisation_name", conf.OrganisationName),
		slog.String("application_name", conf.ApplicationName),
		slog.String("instance_name", conf.InstanceName),
		slog.String("git_hash", gitHash()),
		slog.Bool("debug", conf.Debug),
	)
	if conf.Debug {
		logger = alog.NewDevelopment(container.PGx, container.Settings)
	}
	container.Logger = logger
	// slog.SetDefault(container.Logger.(*slog.Logger)) // todo test if this works even if the cast works

	{ // echo router
		// todo extract echo setup to main arrower repo, ones it is "ready" and can be abstracted for easier use, analog to postgres
		router := echo.New()
		router.HideBanner = true
		router.Logger.SetOutput(io.Discard)
		router.Validator = &CustomValidator{validator: validator.New()}
		router.IPExtractor = echo.ExtractIPFromXFFHeader() // see: https://echo.labstack.com/docs/ip-address
		router.Use(otelecho.Middleware(conf.Web.Hostname, otelecho.WithTracerProvider(container.TraceProvider)))
		router.Use(echoprometheus.NewMiddleware(conf.ApplicationName))
		router.Use(middleware.Static("public")) // todo if prod use fs instead

		hotReload := false
		if conf.Debug {
			router.Debug = true
			router.Use(injectMW)
			hotReload = true
		}

		r, _ := renderer.NewEchoRenderer(container.Logger, container.TraceProvider, router, os.DirFS("shared/views"), hotReload) // todo: if prod load from embed
		err := r.AddBaseData("default", views.NewDefaultBaseDataFunc(container.Settings))
		if err != nil {
			return nil, nil, fmt.Errorf("could not add default base data: %w", err) // todo return shutdown, as some services like postgres are already started
		}

		router.Renderer = r
		container.WebRenderer = r

		// router.Use(session.Middleware())
		ss, _ := auth.NewPGSessionStore(container.PGx, []byte(conf.Web.Secret.Secret()))
		container.WebRouter = router
		container.WebRouter.Use(session.Middleware(ss))
		// di.WebRouter.Use(middleware.CSRF())
		container.WebRouter.Use(auth.EnrichCtxWithUserInfoMiddleware)

		container.AdminRouter = container.WebRouter.Group("/admin")
		container.AdminRouter.Use(auth.EnsureUserIsSuperuserMiddleware)

		container.APIRouter = router.Group("/api") // todo add api middleware
	}

	{ // jobs
		queue, err := jobs.NewPostgresJobs(container.Logger, container.MeterProvider, container.TraceProvider, container.PGx,
			jobs.WithPoolName(conf.InstanceName),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("could not start default job queue: %w", err)
		}

		arrowerQueue, err := jobs.NewPostgresJobs(
			container.Logger,
			container.MeterProvider,
			container.TraceProvider,
			container.PGx,
			jobs.WithQueue("Arrower"),
			jobs.WithPoolName(conf.InstanceName),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("could not start arrower job queue: %w", err)
		}

		container.DefaultQueue = queue
		container.ArrowerQueue = arrowerQueue
	}

	//
	// Start the prometheus HTTP server and pass the exporter Collector to it
	if conf.Web.StatusEndpoint {
		go serveMetrics(ctx, container)
	}

	return container, shutdown(container), nil
}

func shutdown(di *Container) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		di.Logger.InfoContext(ctx, "shutdown...")

		_ = di.WebRouter.Shutdown(ctx)
		_ = di.DefaultQueue.Shutdown(ctx)
		_ = di.ArrowerQueue.Shutdown(ctx)
		_ = di.TraceProvider.Shutdown(ctx)
		_ = di.MeterProvider.Shutdown(ctx)
		_ = di.db.Shutdown(ctx)
		// todo shutdown meter/status endpoint

		return nil // todo error handling
	}
}

func serveMetrics(ctx context.Context, di *Container) {
	const (
		metricPath = "/metrics"
		statusPath = "/status"
	)

	serverStartedAt := time.Now()

	addr := fmt.Sprintf(":%d", di.Config.Web.StatusEndpointPort)

	di.Logger.InfoContext(ctx, "serving status endpoint",
		slog.String("addr", addr),
		slog.String("metric_path", metricPath),
		slog.String("status_path", statusPath),
	)

	// http.Handle("/metrics", promhttp.Handler())
	http.Handle(metricPath, promhttp.HandlerFor(
		prometheus2.DefaultGatherer,
		promhttp.HandlerOpts{ //nolint:exhaustruct
			EnableOpenMetrics: true, // to enable Examplars in the export format
		},
	))

	http.HandleFunc(statusPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // todo in case of error: 500 code
		// TODO disable cache

		statusData := getSystemStatus(di, serverStartedAt)

		_ = json.NewEncoder(w).Encode(statusData)
	})

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		di.Logger.DebugContext(ctx, "error serving http", slog.String("err", err.Error()))

		return
	}
}

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		// Optionally, you could return the error to give each route more control over the status code
		// return echo.NewHTTPError(http.StatusBadRequest, err.Error())

		return err //nolint:wrapcheck // return the original validate error to not break the API for the caller.
	}

	return nil
}

func injectMW(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := next(c); err != nil {
			c.Error(err)
		}

		// skip htmx requests, as the code is already present on the page from a previous load
		if c.Request().Header.Get("HX-Request") == "true" {
			return nil
		}

		if strings.Contains(c.Response().Header().Get("Content-Type"), "text/html") {
			_, _ = c.Response().Write([]byte(hotReloadJSCode))
		}

		return nil
	}
}

//nolint:lll
const hotReloadJSCode = `<!-- Code injected by hot-reload middleware -->
<div id="arrower-status" style="position:absolute; bottom:0; right:0; display:flex; flex-direction:column; align-items:flex-end; margin:10px;">
	<svg style="width:75px;" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#fa4901" class="w-6 h-6">
  		<path fill-rule="evenodd" d="M12.963 2.286a.75.75 0 00-1.071-.136 9.742 9.742 0 00-3.539 6.177A7.547 7.547 0 016.648 6.61a.75.75 0 00-1.152-.082A9 9 0 1015.68 4.534a7.46 7.46 0 01-2.717-2.248zM15.75 14.25a3.75 3.75 0 11-7.313-1.172c.628.465 1.35.81 2.133 1a5.99 5.99 0 011.925-3.545 3.75 3.75 0 013.255 3.717z" clip-rule="evenodd" />
	</svg>
	<span>Arrower not active!<span>
</div>

<script>
	function refreshCSS() {
		var sheets = [].slice.call(document.getElementsByTagName("link"));
		var head = document.getElementsByTagName("head")[0];
		for (var i = 0; i < sheets.length; ++i) {
			var elem = sheets[i];
			head.removeChild(elem);
			var rel = elem.rel;
			if (elem.href && typeof rel != "string" || rel.length == 0 || rel.toLowerCase() == "stylesheet") {
				var url = elem.href.replace(/(&|\?)_cacheOverride=\d+/, '');
				elem.href = url + (url.indexOf('?') >= 0 ? '&' : '?') + '_cacheOverride=' + (new Date().valueOf());
			}
			head.appendChild(elem);
		}
	}

    var loc = window.location;
    var uri = 'ws:';

    if (loc.protocol === 'https:') {
        uri = 'wss:';
    }
    //uri += '//' + loc.host;
    uri += loc.hostname +':3030/' + 'ws';

    console.log("connect to:", uri) // => todo build uri that connects to arrower cli instead of developer's app
    ws = new WebSocket(uri)

    ws.onopen = function() {
        console.log('Connected')
		setTimeout(function() {document.getElementById('arrower-status').style.visibility='hidden'}, 200);
    }

    ws.onmessage = function(msg) {
		console.log("RECEIVED RELOAD", msg.data);
        if (msg.data === 'reload') {
			window.location.reload();
		}
		else if (msg.data == 'refreshCSS') {
			refreshCSS();
		}
    }

    ws.onclose = msg => {
        console.log("Client closed connection")

		document.getElementById('arrower-status').style.visibility='visible'
        // setTimeout(window.location.reload.bind(window.location), 400);
    }

    ws.onerror = error => {
        console.log("Socket error: ", error)
    }
</script>
`

func gitHash() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value
			}
		}
	}

	return "unknown"
}
