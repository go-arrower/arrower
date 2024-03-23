package app_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	prometheusSDK "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"

	"github.com/go-arrower/arrower/app"
)

/*
	About the test cases and how assertions are set up:

	The testing of metrics is done against a prometheus,
	so that is as close to the original as possible and does not depend on mocks or fakes.
	Using the normal promhttp.HandlerFor registers the handler in the default mux
 	and tests cannot run in parallel. Using a custom mux does not work.
	Prometheus offers a solution that works in parallel with testutil, see:
	https://github.com/open-telemetry/opentelemetry-go/blob/main/exporters/prometheus/exporter_test.go
*/

var handlerFunc = http.HandlerFunc(promhttp.HandlerFor(
	prometheusSDK.DefaultGatherer,
	promhttp.HandlerOpts{EnableOpenMetrics: true}, // to enable Examplars in the export format
).ServeHTTP)

//nolint:dupl // decorators are basically identical but need a different metric output
func TestRequestMeteringDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful request", func(t *testing.T) {
		t.Parallel()

		// setup prometheus exporter for testing
		registry := prometheusSDK.NewRegistry()
		exporter, _ := prometheus.New(prometheus.WithRegisterer(registry))
		meterProvider := metric.NewMeterProvider(metric.WithReader(exporter))

		handler := app.NewMeteredRequest[request, response](meterProvider, app.TestSuccessRequestHandler[request, response]())

		_, err := handler.H(context.Background(), request{})
		assert.NoError(t, err)

		// call the prometheus endpoint to scrape all metrics
		rec := httptest.NewRecorder()
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "", nil)

		handlerFunc.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		err = testutil.GatherAndCompare(
			registry,
			metricsForSucceedingRequest,
			// restrict to specific metrics prefix.
			// Prevents missing boilerplate metrics and varying values of usecases_duration_sum.
			"usecases_total", "usecases_duration_seconds_bucket")
		assert.NoError(t, err)
	})

	t.Run("failed request", func(t *testing.T) {
		t.Parallel()

		// setup prometheus exporter for testing
		registry := prometheusSDK.NewRegistry()
		exporter, _ := prometheus.New(prometheus.WithRegisterer(registry))
		meterProvider := metric.NewMeterProvider(metric.WithReader(exporter))

		handler := app.NewMeteredRequest[request, response](meterProvider, app.TestFailureRequestHandler[request, response]())

		_, err := handler.H(context.Background(), request{})
		assert.Error(t, err)

		// call the prometheus endpoint to scrape all metrics
		rec := httptest.NewRecorder()
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "", nil)

		handlerFunc.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		err = testutil.GatherAndCompare(
			registry,
			metricsForFailingRequest,
			// restrict to specific metrics prefix.
			// Prevents missing boilerplate metrics and varying values of usecases_duration_sum.
			"usecases_total", "usecases_duration_seconds_bucket")
		assert.NoError(t, err)
	})
}

func TestCommandMeteringDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful command", func(t *testing.T) {
		t.Parallel()

		// setup prometheus exporter for testing
		registry := prometheusSDK.NewRegistry()
		exporter, _ := prometheus.New(prometheus.WithRegisterer(registry))
		meterProvider := metric.NewMeterProvider(metric.WithReader(exporter))

		handler := app.NewMeteredCommand[request](meterProvider, app.TestSuccessCommandHandler[request]())

		err := handler.H(context.Background(), request{})
		assert.NoError(t, err)

		// call the prometheus endpoint to scrape all metrics
		rec := httptest.NewRecorder()
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "", nil)

		handlerFunc.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		err = testutil.GatherAndCompare(
			registry,
			metricsForSucceedingCommand,
			// restrict to specific metrics prefix.
			// Prevents missing boilerplate metrics and varying values of usecases_duration_sum.
			"usecases_total", "usecases_duration_seconds_bucket")
		assert.NoError(t, err)
	})

	t.Run("failed command", func(t *testing.T) {
		t.Parallel()

		// setup prometheus exporter for testing
		registry := prometheusSDK.NewRegistry()
		exporter, _ := prometheus.New(prometheus.WithRegisterer(registry))
		meterProvider := metric.NewMeterProvider(metric.WithReader(exporter))

		handler := app.NewMeteredCommand[request](meterProvider, app.TestFailureCommandHandler[request]())

		err := handler.H(context.Background(), request{})
		assert.Error(t, err)

		// call the prometheus endpoint to scrape all metrics
		rec := httptest.NewRecorder()
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "", nil)

		handlerFunc.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		err = testutil.GatherAndCompare(
			registry,
			metricsForFailingCommand,
			// restrict to specific metrics prefix.
			// Prevents missing boilerplate metrics and varying values of usecases_duration_sum.
			"usecases_total", "usecases_duration_seconds_bucket")
		assert.NoError(t, err)
	})
}

//nolint:dupl // decorators are basically identical but need a different metric output
func TestQueryMeteringDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful query", func(t *testing.T) {
		t.Parallel()

		// setup prometheus exporter for testing
		registry := prometheusSDK.NewRegistry()
		exporter, _ := prometheus.New(prometheus.WithRegisterer(registry))
		meterProvider := metric.NewMeterProvider(metric.WithReader(exporter))

		handler := app.NewMeteredQuery[request, response](meterProvider, app.TestSuccessQueryHandler[request, response]())

		_, err := handler.H(context.Background(), request{})
		assert.NoError(t, err)

		// call the prometheus endpoint to scrape all metrics
		rec := httptest.NewRecorder()
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "", nil)

		handlerFunc.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		err = testutil.GatherAndCompare(
			registry,
			metricsForSucceedingQuery,
			// restrict to specific metrics prefix.
			// Prevents missing boilerplate metrics and varying values of usecases_duration_sum.
			"usecases_total", "usecases_duration_seconds_bucket")
		assert.NoError(t, err)
	})

	t.Run("failed query", func(t *testing.T) {
		t.Parallel()

		// setup prometheus exporter for testing
		registry := prometheusSDK.NewRegistry()
		exporter, _ := prometheus.New(prometheus.WithRegisterer(registry))
		meterProvider := metric.NewMeterProvider(metric.WithReader(exporter))

		handler := app.NewMeteredQuery[request, response](meterProvider, app.TestFailureQueryHandler[request, response]())

		_, err := handler.H(context.Background(), request{})
		assert.Error(t, err)

		// call the prometheus endpoint to scrape all metrics
		rec := httptest.NewRecorder()
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "", nil)

		handlerFunc.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		err = testutil.GatherAndCompare(
			registry,
			metricsForFailingQuery,
			// restrict to specific metrics prefix.
			// Prevents missing boilerplate metrics and varying values of usecases_duration_sum.
			"usecases_total", "usecases_duration_seconds_bucket")
		assert.NoError(t, err)
	})
}

func TestJobMeteringDecorator_H(t *testing.T) {
	t.Parallel()

	t.Run("successful job", func(t *testing.T) {
		t.Parallel()

		// setup prometheus exporter for testing
		registry := prometheusSDK.NewRegistry()
		exporter, _ := prometheus.New(prometheus.WithRegisterer(registry))
		meterProvider := metric.NewMeterProvider(metric.WithReader(exporter))

		handler := app.NewMeteredJob[request](meterProvider, app.TestSuccessJobHandler[request]())

		err := handler.H(context.Background(), request{})
		assert.NoError(t, err)

		// call the prometheus endpoint to scrape all metrics
		rec := httptest.NewRecorder()
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "", nil)

		handlerFunc.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		err = testutil.GatherAndCompare(
			registry,
			metricsForSucceedingJob,
			// restrict to specific metrics prefix.
			// Prevents missing boilerplate metrics and varying values of usecases_duration_sum.
			"usecases_total", "usecases_duration_seconds_bucket")
		assert.NoError(t, err)
	})

	t.Run("failed job", func(t *testing.T) {
		t.Parallel()

		// setup prometheus exporter for testing
		registry := prometheusSDK.NewRegistry()
		exporter, _ := prometheus.New(prometheus.WithRegisterer(registry))
		meterProvider := metric.NewMeterProvider(metric.WithReader(exporter))

		handler := app.NewMeteredJob[request](meterProvider, app.TestFailureJobHandler[request]())

		err := handler.H(context.Background(), request{})
		assert.Error(t, err)

		// call the prometheus endpoint to scrape all metrics
		rec := httptest.NewRecorder()
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "", nil)

		handlerFunc.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		err = testutil.GatherAndCompare(
			registry,
			metricsForFailingJob,
			// restrict to specific metrics prefix.
			// Prevents missing boilerplate metrics and varying values of usecases_duration_sum.
			"usecases_total", "usecases_duration_seconds_bucket")
		assert.NoError(t, err)
	})
}

var (
	// needs to match the exact format expected from the prometheus endpoint
	// prepare the output, so it can be read multiple times by different tests, see:https://siongui.github.io/2018/10/28/go-read-twice-from-same-io-reader/
	rawMetricsForSucceedingUseCase, _ = io.ReadAll(strings.NewReader(`
# HELP usecases_duration_seconds a simple hist
# TYPE usecases_duration_seconds histogram
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="0"} 0
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="5"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="10"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="25"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="50"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="75"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="100"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="250"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="500"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="750"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="1000"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="2500"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="5000"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="7500"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="10000"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="+Inf"} 1
usecases_duration_seconds_sum{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version=""} 2.198e-06
usecases_duration_seconds_count{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version=""} 1
# HELP usecases_total a simple counter
# TYPE usecases_total counter
usecases_total{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",status="success"} 1
`))
	metricsForSucceedingRequest = bytes.NewReader(rawMetricsForSucceedingUseCase)
	metricsForSucceedingCommand = bytes.NewReader(rawMetricsForSucceedingUseCase)
	metricsForSucceedingQuery   = bytes.NewReader(rawMetricsForSucceedingUseCase)
	metricsForSucceedingJob     = bytes.NewReader(rawMetricsForSucceedingUseCase)

	// needs to match the exact format expected from the prometheus endpoint.
	rawMetricsForFailingUseCase, _ = io.ReadAll(strings.NewReader(`
# HELP usecases_duration_seconds a simple hist
# TYPE usecases_duration_seconds histogram
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="0"} 0
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="5"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="10"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="25"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="50"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="75"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="100"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="250"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="500"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="750"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="1000"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="2500"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="5000"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="7500"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="10000"} 1
usecases_duration_seconds_bucket{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",le="+Inf"} 1
usecases_duration_seconds_sum{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version=""} 2.198e-06
usecases_duration_seconds_count{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version=""} 1
# HELP usecases_total a simple counter
# TYPE usecases_total counter
usecases_total{command="app_test.request",otel_scope_name="arrower.application",otel_scope_version="",status="failure"} 1
`))
	metricsForFailingRequest = bytes.NewReader(rawMetricsForFailingUseCase)
	metricsForFailingCommand = bytes.NewReader(rawMetricsForFailingUseCase)
	metricsForFailingQuery   = bytes.NewReader(rawMetricsForFailingUseCase)
	metricsForFailingJob     = bytes.NewReader(rawMetricsForFailingUseCase)
)
