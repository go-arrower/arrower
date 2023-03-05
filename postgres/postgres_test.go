//go:build integration

package postgres_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/postgres"
	"github.com/go-arrower/arrower/tests"
)

var runOptions = &dockertest.RunOptions{ //nolint:exhaustruct
	Repository: "postgres",
	Tag:        "15",
	Env: []string{
		"POSTGRES_PASSWORD=secret",
		"POSTGRES_USER=username",
		"POSTGRES_DB=dbname",
		"listen_addresses = '*'",
	},
}

func TestConnect(t *testing.T) {
	t.Parallel()

	t.Run("ensure db connected", func(t *testing.T) {
		t.Parallel()

		var pgHandler *postgres.Handler
		cleanup, _ := tests.StartDockerContainer(runOptions, func(resource *dockertest.Resource) func() error {
			return func() error {
				port, _ := strconv.Atoi(resource.GetPort(fmt.Sprintf("%s/tcp", "5432")))

				handler, err := postgres.Connect(postgres.Config{ //nolint:exhaustruct
					Host:     "localhost",
					Port:     port,
					User:     "username",
					Password: "secret",
					Database: "dbname",
				})
				if err != nil {
					return err //nolint:wrapcheck
				}

				pgHandler = handler

				return nil
			}
		})

		err := pgHandler.PGx.Ping(context.Background())
		assert.NoError(t, err)

		_ = cleanup()
	})

	t.Run("ensure db connection closed", func(t *testing.T) {
		t.Parallel()

		var pgHandler *postgres.Handler
		cleanup, _ := tests.StartDockerContainer(runOptions, func(resource *dockertest.Resource) func() error {
			return func() error {
				port, _ := strconv.Atoi(resource.GetPort(fmt.Sprintf("%s/tcp", "5432")))

				handler, err := postgres.Connect(postgres.Config{ //nolint:exhaustruct
					Host:     "localhost",
					Port:     port,
					User:     "username",
					Password: "secret",
					Database: "dbname",
				})
				if err != nil {
					return err //nolint:wrapcheck
				}

				pgHandler = handler

				return nil
			}
		})

		err := pgHandler.PGx.Ping(context.Background())
		assert.NoError(t, err)

		err = pgHandler.Shutdown(context.Background())
		assert.NoError(t, err)

		err = pgHandler.PGx.Ping(context.Background())
		assert.Error(t, err)

		_ = cleanup()
	})
}
