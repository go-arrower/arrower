//go:build integration

package jobs_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	mnoop "go.opentelemetry.io/otel/metric/noop"
	tnoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/jobs"
	"github.com/go-arrower/arrower/tests"
)

type myJob struct {
	Payload int
}

type otherJob struct{}

func Example_postgresJobsHandler() {
	db := tests.GetPostgresDockerForIntegrationTestingInstance()

	jq, _ := jobs.NewPostgresJobs(alog.NewNoop(), mnoop.NewMeterProvider(), tnoop.NewTracerProvider(), db.PGx(),
		jobs.WithPollInterval(time.Millisecond), jobs.WithPoolSize(1), // options are to make example deterministic, no production values
	)

	_ = jq.RegisterJobFunc(func(_ context.Context, job myJob) error {
		fmt.Println("myJob with payload:", job.Payload)

		return nil
	})
	_ = jq.RegisterJobFunc(func(_ context.Context, _ otherJob) error {
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

func Example_inMemoryAssertionsForTesting() {
	jq := jobs.Test(new(testing.T))

	_ = jq.Enqueue(ctx, myJob{})

	jq.NotEmpty()
	jq.Total(1, "queue should have one Job enqueued")
	jq.Contains(myJob{})

	// Output:
}
