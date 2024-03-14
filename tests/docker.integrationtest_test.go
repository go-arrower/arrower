//go:build integration

package tests_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/tests"
)

func TestStartDockerContainer(t *testing.T) {
	t.Parallel()

	t.Run("invalid docker run options", func(t *testing.T) {
		t.Parallel()

		cleanup, err := tests.StartDockerContainer(nil, nil)
		assert.ErrorIs(t, err, tests.ErrDockerFailure)
		assert.Nil(t, cleanup)
	})

	t.Run("invalid docker retry func", func(t *testing.T) {
		t.Parallel()

		cleanup, err := tests.StartDockerContainer(&dockertest.RunOptions{Repository: "ghcr.io/go-arrower/postgres"}, nil)
		assert.ErrorIs(t, err, tests.ErrDockerFailure)
		assert.Nil(t, cleanup)
	})

	t.Run("start docker container", func(t *testing.T) {
		t.Parallel()

		cleanup, err := tests.StartDockerContainer(&runOptions, retryFunc)
		assert.NoError(t, err)
		assert.NotNil(t, cleanup)

		err = cleanup()
		assert.NoError(t, err)
	})

	t.Run("start same container again", func(t *testing.T) {
		t.Parallel()

		options := runOptions
		options.Name = "ensure-single-container"

		cleanup1, err1 := tests.StartDockerContainer(&options, retryFunc)
		assert.NoError(t, err1)
		assert.NotNil(t, cleanup1)

		cleanup2, err2 := tests.StartDockerContainer(&options, retryFunc)
		assert.NoError(t, err2)
		assert.NotNil(t, cleanup2)

		err := cleanup1()
		assert.NoError(t, err)
		err = cleanup2()
		assert.NoError(t, err)
	})

	t.Run("start in parallel", func(t *testing.T) {
		t.Parallel()

		/*
			Note on the quality of this test:
			In a real project multiple packages call `tests.StartDockerContainer`, but the go test tool does
			build each package into its own executable test. This test does not cover that scenario.
		*/

		wg := sync.WaitGroup{}

		wg.Add(10)
		for i := 0; i < 10; i++ {
			go func() {
				options := runOptions
				options.Name = "ensure-single-container"

				cleanup, err2 := tests.StartDockerContainer(&options, retryFunc)
				assert.NoError(t, err2)
				assert.NotNil(t, cleanup)

				wg.Done()
			}()
		}

		wg.Wait()
	})
}

var (
	runOptions = dockertest.RunOptions{
		Repository: "ghcr.io/go-arrower/postgres",
		Tag:        "latest",
		Env: []string{
			"POSTGRES_PASSWORD=secret",
			"POSTGRES_USER=arrower",
			"POSTGRES_DB=dbname_test",
			"listen_addresses = '*'",
		},
	}

	retryFunc = func(resource *dockertest.Resource) func() error {
		return func() error {
			port := resource.GetPort("5432/tcp")
			url := fmt.Sprintf("postgres://arrower:secret@localhost:%s/dbname_test", port)

			conn, err := pgx.Connect(context.Background(), url)
			if err != nil {
				return err //nolint:wrapcheck
			}

			return conn.Ping(context.Background()) //nolint:wrapcheck
		}
	}
)
