//go:build integration

package jobs_test

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/jobs"
	"github.com/go-arrower/arrower/tests"
)

type myJob struct {
	Payload int
}

type otherJob struct{}

func ExampleGueHandler_Enqueue() {
	db := tests.NewPostgresDockerForIntegrationTesting()

	jq, _ := jobs.NewPostgresJobs(alog.NewTest(nil), noop.NewMeterProvider(), trace.NewNoopTracerProvider(), db.PGx(),
		jobs.WithPollInterval(time.Millisecond), jobs.WithPoolSize(1), // options are to make example deterministic, no production values
	)

	_ = jq.RegisterJobFunc(func(ctx context.Context, j myJob) error {
		fmt.Println("myJob with payload:", j.Payload)

		return nil
	})
	_ = jq.RegisterJobFunc(func(ctx context.Context, j otherJob) error {
		fmt.Println("otherJob")

		return nil
	})

	// enqueue a single job
	_ = jq.Enqueue(context.Background(), myJob{Payload: 1})

	// enqueue multiple jobs
	_ = jq.Enqueue(context.Background(), []myJob{{Payload: 1}, {Payload: 2}})

	// enqueue multiple jobs
	_ = jq.Enqueue(context.Background(), []any{myJob{Payload: 1}, otherJob{}})

	// Wait for the workers to start and run.
	time.Sleep(time.Second)
	db.Cleanup()

	// Output: myJob with payload: 1
	// myJob with payload: 1
	// myJob with payload: 2
	// myJob with payload: 1
	// otherJob
}
