package arrower

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	prometheusSDK "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/contexts/auth"
	"github.com/go-arrower/arrower/jobs"
	"github.com/go-arrower/arrower/postgres"
	"github.com/go-arrower/arrower/renderer"
	"github.com/go-arrower/arrower/secret"
	"github.com/go-arrower/arrower/setting"
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

	WebRouter   *echo.Echo
	APIRouter   *echo.Group
	AdminRouter *echo.Group
	WebRenderer *renderer.EchoRenderer

	RootCmd *cobra.Command

	ArrowerQueue jobs.Queue
	DefaultQueue jobs.Queue

	Settings setting.Settings

	// auth & admin Contexts
	AuthContext auth.API

	metricsEndpoint *http.Server
}

func New() (*Container, error) {
	//nolint:mnd  // allow direct port definitions etc.
	return InitialiseDefaultDependencies(context.Background(), &Config{
		OrganisationName: "arrower",
		ApplicationName:  "",
		InstanceName:     getOutboundIP(),
		Environment:      LocalEnv,
		Postgres: Postgres{
			User:     "arrower",
			Password: secret.New("secret"),
			Database: "arrower",
			Host:     "localhost",
			Port:     5432,
			SSLMode:  "disable",
			MaxConns: 100,
		},
		HTTP: HTTP{
			CookieSecret:          secret.New("secret"),
			Port:                  8080,
			StatusEndpointEnabled: true,
			StatusEndpointPort:    2223,
		},
		OTEL: OTEL{
			Host:     "localhost",
			Port:     4317,
			Hostname: "www.servername.tld",
		},
	}, nil, nil, nil, nil)
}

func (c *Container) EnsureAllDependenciesPresent() error {
	if c.Config == nil {
		return fmt.Errorf("%w: global config not found", ErrMissingDependency)
	}

	return nil
}

func InitialiseDefaultDependencies(
	ctx context.Context,
	conf *Config,
	migrations fs.FS,
	publicAssets fs.FS,
	sharedViews fs.FS,
	funcs template.FuncMap,
) (*Container, error) {
	dc := &Container{
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

		{ // traces
			opts := []otlptracegrpc.Option{
				otlptracegrpc.WithEndpoint(fmt.Sprintf("%s:%d", conf.OTEL.Host, conf.OTEL.Port)),
				otlptracegrpc.WithInsecure(),
			}

			if conf.Environment == TestEnv {
				// while unit testing no otel endpoint is running. This means some operations, e.g.
				// traceprovider shutdown will block until the shutdown ctx expires,
				// which is too long for testing.
				opts = append(opts, otlptracegrpc.WithTimeout(10*time.Millisecond))
			}

			traceExporter, err := otlptracegrpc.New(ctx, opts...)
			if err != nil {
				return nil, fmt.Errorf("could not connect to trace exporter: %w", err)
			}

			traceProvider := trace.NewTracerProvider(
				trace.WithBatcher(traceExporter), // prod
				trace.WithResource(resource),
				// set the sampling rate based on the parent span to 60%
				trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(0.6))),
			)
			if conf.Environment == LocalEnv {
				traceProvider = trace.NewTracerProvider(
					// trace.WithSyncer(traceExporter), // if tempo is not running, this blocks each http request to the app
					trace.WithBatcher(traceExporter, trace.WithBlocking()),
					trace.WithResource(resource),
					trace.WithSampler(trace.AlwaysSample()),
				)
			}

			dc.TraceProvider = traceProvider
			otel.SetTracerProvider(traceProvider)
		}

		{ // metrics
			exporter, err := prometheus.New()
			if err != nil {
				return nil, fmt.Errorf("could not create to prometheus exporter: %w", err)
			}

			meterProvider := metric.NewMeterProvider(
				metric.WithResource(resource),
				metric.WithReader(exporter),
			)

			dc.MeterProvider = meterProvider
			otel.SetMeterProvider(meterProvider)
		}
	}

	{ // postgres
		if conf.Postgres != (Postgres{}) {
			mf := migrations
			if mf == nil {
				mf = postgres.ArrowerDefaultMigrations
			}

			pg, err := postgres.ConnectAndMigrate(ctx, postgres.Config{
				User:       conf.Postgres.User,
				Password:   conf.Postgres.Password.Secret(),
				Database:   conf.Postgres.Database,
				Host:       conf.Postgres.Host,
				Port:       conf.Postgres.Port,
				SSLMode:    conf.Postgres.SSLMode,
				MaxConns:   conf.Postgres.MaxConns,
				Migrations: mf,
			}, dc.TraceProvider)
			if err != nil {
				return nil, fmt.Errorf("could not connect to postgres: %w", err)
			}

			dc.PGx = pg.PGx

			dc.Settings = setting.NewPostgresSettings(dc.PGx)
		}
	}

	{ // logger
		logger := alog.New()
		logger = logger.With(
			slog.String("organisation_name", conf.OrganisationName),
			slog.String("application_name", conf.ApplicationName),
			slog.String("instance_name", conf.InstanceName),
			slog.String("git_hash", gitHash()),
			slog.String("environment", string(conf.Environment)),
		)

		if conf.Environment == LocalEnv {
			logger = alog.NewDevelopment(dc.PGx) // todo work if loki is down
		}

		dc.Logger = logger
		slog.SetDefault(dc.Logger.(*slog.Logger))
	}

	{ // web routers
		var r *renderer.EchoRenderer

		router := echo.New()
		router.HideBanner = true
		router.Logger.SetOutput(io.Discard)
		router.Validator = &CustomValidator{validator: validator.New()}
		router.IPExtractor = echo.ExtractIPFromXFFHeader() // see: https://echo.labstack.com/docs/ip-address

		router.Use(otelecho.Middleware(conf.OTEL.Hostname, otelecho.WithTracerProvider(dc.TraceProvider)))
		router.Use(echoprometheus.NewMiddleware(conf.ApplicationName))
		router.Use(middleware.RequestIDWithConfig(middleware.RequestIDConfig{
			TargetHeader: "Request-Id",
			RequestIDHandler: func(c echo.Context, rid string) {
				c.SetRequest(c.Request().WithContext(alog.AddAttr(
					c.Request().Context(),
					slog.String("request_id", rid)),
				))
			},
		}))

		if conf.Environment == LocalEnv {
			router.Debug = true
			r, _ = renderer.NewEchoRenderer(dc.Logger, dc.TraceProvider, router, os.DirFS("shared/views"), funcs, true)

			router.StaticFS("/static/", os.DirFS("public"))

			router.Use(injectMW)
		} else {
			r, _ = renderer.NewEchoRenderer(dc.Logger, dc.TraceProvider, router, sharedViews, funcs, false)

			router.StaticFS("/static/", publicAssets)
		}
		// err := r.AddBaseData("default", views.NewDefaultBaseDataFunc(dc.Settings))
		//if err != nil {
		//	return nil, fmt.Errorf("could not add default base data: %w", err)
		//	// todo return shutdown, as some services like postgres are already started
		//}

		router.Renderer = r
		dc.WebRenderer = r

		// router.Use(session.Middleware())
		dc.WebRouter = router
		if dc.PGx != nil {
			ss, _ := auth.NewPGSessionStore(dc.PGx, []byte(conf.HTTP.CookieSecret.Secret()))
			dc.WebRouter.Use(session.Middleware(ss))
			// di.WebRouter.Use(middleware.CSRF())
			dc.WebRouter.Use(auth.EnrichCtxWithUserInfoMiddleware)
		}

		dc.AdminRouter = dc.WebRouter.Group("/admin")
		// dc.AdminRouter.Use(auth.EnsureUserIsSuperuserMiddleware)
		// todo enable adain. Is disabled because pgx could be null or session not present...

		dc.APIRouter = router.Group("/api") // todo add api middleware
	}

	{ // jobs
		queue, err := jobs.NewPostgresJobs(dc.Logger, dc.MeterProvider, dc.TraceProvider, dc.PGx,
			jobs.WithPoolName(conf.InstanceName),
		)
		if err != nil {
			return nil, fmt.Errorf("could not start default job queue: %w", err)
		}

		arrowerQueue, err := jobs.NewPostgresJobs(
			dc.Logger,
			dc.MeterProvider,
			dc.TraceProvider,
			dc.PGx,
			jobs.WithQueue("Arrower"),
			jobs.WithPoolName(conf.InstanceName),
		)
		if err != nil {
			return nil, fmt.Errorf("could not start arrower job queue: %w", err)
		}

		dc.DefaultQueue = queue
		dc.ArrowerQueue = arrowerQueue
	}

	return dc, nil
}

func (c *Container) Start(ctx context.Context) error {
	c.Logger.LogAttrs(ctx, alog.LevelInfo, "starting all servers")

	// Start the prometheus HTTP server and pass the exporter Collector to it
	if c.Config.HTTP.StatusEndpointEnabled {
		c.metricsEndpoint = serveMetrics(ctx, c)
	}

	go func() {
		_ = c.WebRouter.Start(fmt.Sprintf(":%d", c.Config.HTTP.Port))
	}()

	return nil
}

func (c *Container) Shutdown(ctx context.Context) error { // todo error handling
	c.Logger.LogAttrs(ctx, alog.LevelInfo, "shutting down all servers")

	_ = c.WebRouter.Shutdown(ctx)

	if c.Config.HTTP.StatusEndpointEnabled {
		_ = c.metricsEndpoint.Shutdown(ctx)
	}

	_ = c.DefaultQueue.Shutdown(ctx)
	_ = c.ArrowerQueue.Shutdown(ctx)

	if c.PGx != nil {
		c.PGx.Close()
	}

	_ = c.TraceProvider.Shutdown(ctx)
	_ = c.MeterProvider.Shutdown(ctx)

	return nil
}

func serveMetrics(ctx context.Context, di *Container) *http.Server {
	const (
		metricPath = "/metrics"
		statusPath = "/status"
	)

	serverStartedAt := time.Now()

	srv := &http.Server{Addr: fmt.Sprintf(":%d", di.Config.HTTP.StatusEndpointPort)}

	di.Logger.InfoContext(ctx, "serving status endpoint",
		slog.String("addr", srv.Addr),
		slog.String("metric_path", metricPath),
		slog.String("status_path", statusPath),
	)

	// http.Handle("/metrics", promhttp.Handler())
	http.Handle(metricPath, promhttp.HandlerFor(
		prometheusSDK.DefaultGatherer,
		promhttp.HandlerOpts{ //nolint:exhaustruct
			EnableOpenMetrics: true, // to enable Examplars in the export format
		},
	))

	http.HandleFunc(statusPath, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // todo in case of error: 500 code
		// TODO disable cache

		statusData := getSystemStatus(di, serverStartedAt)

		_ = json.NewEncoder(w).Encode(statusData)
	})

	go func() {
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			di.Logger.DebugContext(ctx, "error serving http", slog.String("err", err.Error()))

			return
		}
	}()

	return srv
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
		if c.Request().Header.Get("Hx-Request") == "true" {
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
		var head = document.getElementsByTagName("head")[0];
		var sheets = [].slice.call(document.getElementsByTagName("link"));
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

// Get preferred outbound ip of this machine.
//
// Actually, it does not establish any connection and the destination does not need to be existed at all :)
// So, what the code does actually, is to get the local up address if it would connect to that target,
// you can change to any other IP address you want. conn.LocalAddr().String() is the local ip and port.
// https://stackoverflow.com/questions/23558425/how-do-i-get-the-local-ip-address-in-go
func getOutboundIP() string {
	conn, err := net.Dial("udp", "5.1.66.255:80")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}
