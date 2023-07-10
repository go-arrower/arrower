//go:build integration

//nolint:govet,lll // the purpose is to showcase examples, not production code
package jobs_test

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/jobs"
	"github.com/go-arrower/arrower/postgres"
	"github.com/go-arrower/arrower/tests"
)

type myJob struct {
	Payload int
}

type otherJob struct{}

func ExampleGueHandler_Enqueue() {
	db, teardown := setup()

	jq, _ := jobs.NewGueJobs(alog.NewTest(os.Stderr), noop.NewMeterProvider(), trace.NewNoopTracerProvider(), db.PGx,
		jobs.WithPollInterval(time.Second), jobs.WithPoolSize(1), // options are to make example deterministic, no production values
	)

	_ = jq.RegisterJobFunc(func(ctx context.Context, j myJob) error {
		fmt.Println("myJob with payload:", j.Payload)

		return nil
	})

	// enqueue a single job
	_ = jq.Enqueue(context.Background(), myJob{Payload: 1})

	// enqueue multiple jobs
	_ = jq.Enqueue(context.Background(), []myJob{{Payload: 1}, {Payload: 2}})

	// enqueue multiple jobs
	_ = jq.Enqueue(context.Background(), []any{myJob{Payload: 1}, otherJob{}})

	teardown()
	// Output: myJob with payload: 1
	// myJob with payload: 1
	// myJob with payload: 2
	// myJob with payload: 1
}

func setup() (*postgres.Handler, func()) {
	ctx := context.Background()
	handler, cleanup := tests.GetDBConnectionForIntegrationTesting(ctx)

	return handler, func() {
		// Wait for the workers to start and run.
		// Hide this here, so the example above looks cleaner to read.
		time.Sleep(2 * time.Second)

		_ = handler.Shutdown(ctx)
		_ = cleanup()
	}
}
