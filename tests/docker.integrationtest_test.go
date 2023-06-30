//go:build integration

package tests_test

import (
	"context"
	"errors"
	"fmt"
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
		assert.True(t, errors.Is(err, tests.ErrDockerFailure))
		assert.Nil(t, cleanup)
	})

	t.Run("invalid docker retry func", func(t *testing.T) {
		t.Parallel()

		cleanup, err := tests.StartDockerContainer(&dockertest.RunOptions{Repository: "postgres"}, nil)
		assert.True(t, errors.Is(err, tests.ErrDockerFailure))
		assert.Nil(t, cleanup)
	})

	t.Run("start docker container", func(t *testing.T) {
		t.Parallel()

		runOptions := &dockertest.RunOptions{
			Repository: "postgres",
			Tag:        "15",
			Env: []string{
				"POSTGRES_PASSWORD=secret",
				"POSTGRES_USER=username",
				"POSTGRES_DB=dbname_test",
				"listen_addresses = '*'",
			},
		}

		retryFunc := func(resource *dockertest.Resource) func() error {
			return func() error {
				port := resource.GetPort(fmt.Sprintf("%s/tcp", "5432"))
				url := fmt.Sprintf("postgres://username:secret@localhost:%s/dbname_test", port)

				conn, err := pgx.Connect(context.Background(), url)
				if err != nil {
					return err //nolint:wrapcheck
				}

				return conn.Ping(context.Background()) //nolint:wrapcheck
			}
		}

		cleanup, err := tests.StartDockerContainer(runOptions, retryFunc)
		assert.NoError(t, err)
		assert.NotNil(t, cleanup)

		err = cleanup()
		assert.NoError(t, err)
	})
}

func TestGetDBConnectionForIntegrationTesting(t *testing.T) {
	t.Parallel()

	pgHandler, cleanup := tests.GetDBConnectionForIntegrationTesting(context.Background())
	assert.NotNil(t, pgHandler)
	assert.NotNil(t, cleanup)

	err := cleanup()
	assert.NoError(t, err)
}
