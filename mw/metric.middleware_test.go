package mw_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	prometheusSDK "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"

	"github.com/go-arrower/arrower/mw"
)

/*
	About the test cases and how assertions are set up:

	The testing of metrics is done against a prometheus, so that is as close to the original as possible and
	does not depend on mocks or fakes.
	Using the normal promhttp.HandlerFor registers the handler in the default mux and tests can not run in parallel.
	Using a custom mux does not work. Prometheus offers a solution that works in parallel with testutil,
	see https://github.com/open-telemetry/opentelemetry-go/blob/main/exporters/prometheus/exporter_test.go
*/

var handler = http.HandlerFunc(promhttp.HandlerFor(
	prometheusSDK.DefaultGatherer,
	promhttp.HandlerOpts{EnableOpenMetrics: true}, // to enable Examplars in the export format
).ServeHTTP)

func TestMetric(t *testing.T) {
	t.Parallel()

	t.Run("successful command", func(t *testing.T) {
		t.Parallel()

		// setup prometheus exporter for testing
		registry := prometheusSDK.NewRegistry()
		exporter, _ := prometheus.New(prometheus.WithRegisterer(registry))
		meterProvider := metric.NewMeterProvider(metric.WithReader(exporter))

		cmd := mw.Metric(meterProvider, func(context.Context, exampleCommand) (string, error) {
			return "", nil
		})

		_, _ = cmd(context.Background(), exampleCommand{})

		// call the prometheus endpoint to scrape all metrics
		rec := httptest.NewRecorder()
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "", nil)

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		err := testutil.GatherAndCompare(
			registry,
			metricsForSucceedingUseCase,
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

		cmd := mw.Metric(meterProvider, func(context.Context, exampleCommand) (string, error) {
			return "", errUseCaseFails
		})

		_, _ = cmd(context.Background(), exampleCommand{})

		// call the prometheus endpoint to scrape all metrics
		rec := httptest.NewRecorder()
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "", nil)

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		err := testutil.GatherAndCompare(
			registry,
			metricsForFailingUseCase,
			// restrict to specific metrics prefix.
			// Prevents missing boilerplate metrics and varying values of usecases_duration_sum.
			"usecases_total", "usecases_duration_seconds_bucket")
		assert.NoError(t, err)
	})
}

func TestMetricU(t *testing.T) {
	t.Parallel()

	t.Run("successful command", func(t *testing.T) {
		t.Parallel()

		// setup prometheus exporter for testing
		registry := prometheusSDK.NewRegistry()
		exporter, _ := prometheus.New(prometheus.WithRegisterer(registry))
		meterProvider := metric.NewMeterProvider(metric.WithReader(exporter))

		cmd := mw.MetricU(meterProvider, func(context.Context, exampleCommand) error {
			return nil
		})

		_ = cmd(context.Background(), exampleCommand{})

		// call the prometheus endpoint to scrape all metrics
		rec := httptest.NewRecorder()
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "", nil)

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		err := testutil.GatherAndCompare(
			registry,
			metricsForSucceedingUseCaseU,
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

		cmd := mw.MetricU(meterProvider, func(context.Context, exampleCommand) error {
			return errUseCaseFails
		})

		_ = cmd(context.Background(), exampleCommand{})

		// call the prometheus endpoint to scrape all metrics
		rec := httptest.NewRecorder()
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "", nil)

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		err := testutil.GatherAndCompare(
			registry,
			metricsForFailingUseCaseU,
			// restrict to specific metrics prefix.
			// Prevents missing boilerplate metrics and varying values of usecases_duration_sum.
			"usecases_total", "usecases_duration_seconds_bucket")
		assert.NoError(t, err)
	})
}
