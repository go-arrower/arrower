//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
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
		"POSTGRES_DB=dbname_test",
		"listen_addresses = '*'",
	},
}

func TestConnect(t *testing.T) {
	t.Parallel()

	t.Run("ensure db connected", func(t *testing.T) {
		t.Parallel()

		var pgHandler *postgres.Handler
		cleanup, _ := tests.StartDockerContainer(runOptions, func(resource *dockertest.Resource) func() error {
			port, _ := strconv.Atoi(resource.GetPort(fmt.Sprintf("%s/tcp", "5432")))

			return func() error {
				handler, err := postgres.Connect(context.Background(), postgres.Config{ //nolint:exhaustruct
					Host:     "localhost",
					Port:     port,
					User:     "username",
					Password: "secret",
					Database: "dbname_test",
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
		assert.NotEmpty(t, pgHandler.PGx)
		assert.NotEmpty(t, pgHandler.DB)

		_ = cleanup()
	})

	t.Run("ensure db connection closed", func(t *testing.T) {
		t.Parallel()

		var pgHandler *postgres.Handler
		cleanup, _ := tests.StartDockerContainer(runOptions, func(resource *dockertest.Resource) func() error {
			port, _ := strconv.Atoi(resource.GetPort(fmt.Sprintf("%s/tcp", "5432")))

			return func() error {
				handler, err := postgres.Connect(context.Background(), postgres.Config{ //nolint:exhaustruct
					Host:     "localhost",
					Port:     port,
					User:     "username",
					Password: "secret",
					Database: "dbname_test",
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

func TestConnectAndMigrate(t *testing.T) {
	t.Parallel()

	t.Run("missing migration path fails", func(t *testing.T) {
		t.Parallel()

		_, _ = tests.StartDockerContainer(runOptions, func(resource *dockertest.Resource) func() error {
			port, _ := strconv.Atoi(resource.GetPort(fmt.Sprintf("%s/tcp", "5432")))

			return func() error {
				handler, err := postgres.ConnectAndMigrate(context.Background(), postgres.Config{ //nolint:exhaustruct
					Host:     "localhost",
					Port:     port,
					User:     "username",
					Password: "secret",
					Database: "dbname_test",
				})
				if err != nil && !errors.Is(err, postgres.ErrMigrationFailed) {
					return err //nolint:wrapcheck
				}

				t.Log(err)
				assert.Error(t, err)
				assert.True(t, errors.Is(err, postgres.ErrMigrationFailed), err)
				assert.Nil(t, handler)

				return nil
			}
		})
	})

	t.Run("ensure db migration run", func(t *testing.T) {
		t.Parallel()

		var pgHandler *postgres.Handler
		cleanup, _ := tests.StartDockerContainer(runOptions, func(resource *dockertest.Resource) func() error {
			port, _ := strconv.Atoi(resource.GetPort(fmt.Sprintf("%s/tcp", "5432")))

			return func() error {
				handler, err := postgres.ConnectAndMigrate(context.Background(), postgres.Config{ //nolint:exhaustruct
					Host:       "localhost",
					Port:       port,
					User:       "username",
					Password:   "secret",
					Database:   "dbname_test",
					Migrations: postgres.ArrowerDefaultMigrations,
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
		ensureMigrationsRun(t, pgHandler.PGx)
		ensureMigrationFilesLoadedAndApplied(t, pgHandler.PGx)

		_ = cleanup()
	})

	t.Run("ensure migration does not fail, if the schema is already up to date", func(t *testing.T) {
		t.Parallel()

		var config postgres.Config

		cleanup, _ := tests.StartDockerContainer(runOptions, func(resource *dockertest.Resource) func() error {
			port, _ := strconv.Atoi(resource.GetPort(fmt.Sprintf("%s/tcp", "5432")))
			config = postgres.Config{
				Host:       "localhost",
				Port:       port,
				User:       "username",
				Password:   "secret",
				Database:   "dbname_test",
				Migrations: postgres.ArrowerDefaultMigrations,
				MaxConns:   10,
			}

			return func() error {
				_, err := postgres.ConnectAndMigrate(context.Background(), config)
				if err != nil {
					return err //nolint:wrapcheck
				}

				return nil
			}
		})

		// container is up and running => connect a second time to run migrations again
		_, err := postgres.ConnectAndMigrate(context.Background(), config)
		assert.NoError(t, err)

		_ = cleanup()
	})
}

func ensureMigrationsRun(t *testing.T, pgx *pgxpool.Pool) {
	t.Helper()

	row := pgx.QueryRow(
		context.Background(),
		`SELECT EXISTS (
				SELECT FROM information_schema.tables
					WHERE  table_schema = 'public'
					AND    table_name   = 'schema_migrations'
			);`)

	var exists bool

	err := row.Scan(&exists)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func ensureMigrationFilesLoadedAndApplied(t *testing.T, pgx *pgxpool.Pool) {
	t.Helper()

	row := pgx.QueryRow(
		context.Background(),
		`select pg_get_functiondef('set_updated_at()'::regprocedure);`)

	var funcExists string

	err := row.Scan(&funcExists)
	assert.NoError(t, err)
}
