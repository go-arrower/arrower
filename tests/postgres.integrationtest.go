//go:build integration

package tests

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/go-testfixtures/testfixtures/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/dockertest/v3"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-arrower/arrower/postgres"
)

// NewPostgresDockerForIntegrationTesting returns Handler that is fully connected and has a method to clean up
// after the integration tests are done.
// It spins up and connects to a db instance in a new docker container.
// If called in a CI environment, the pipeline needs access to a docker socket.
// In case of an issue it panics.
func NewPostgresDockerForIntegrationTesting() *PostgresDocker {
	var (
		pgHandler *postgres.Handler

		runOptions = &dockertest.RunOptions{ //nolint:exhaustruct // only set required configuration
			Repository: "postgres",
			Tag:        "15",
			Env: []string{
				"POSTGRES_PASSWORD=secret",
				"POSTGRES_USER=username",
				"POSTGRES_DB=dbname_test",
				"listen_addresses = '*'",
				"MaxConnections=1000",
			},
			Cmd: []string{"-c", "max_connections=1000"},
		}

		retryFunc = func(resource *dockertest.Resource) func() error {
			port, _ := strconv.Atoi(resource.GetPort(fmt.Sprintf("%s/tcp", "5432")))
			conf := postgres.Config{
				User:       "username",
				Password:   "secret",
				Database:   "dbname_test",
				Host:       "localhost",
				Port:       port,
				MaxConns:   10, //nolint:gomnd
				Migrations: postgres.ArrowerDefaultMigrations,
			}

			return func() error {
				handler, err := postgres.ConnectAndMigrate(context.Background(), conf, trace.NewNoopTracerProvider())
				if err != nil {
					return err //nolint:wrapcheck
				}

				pgHandler = handler

				return nil
			}
		}
	)

	cleanup, err := StartDockerContainer(runOptions, retryFunc)
	if err != nil {
		panic(err)
	}

	return &PostgresDocker{
		pg:            pgHandler,
		cleanupDocker: cleanup,
	}
}

type PostgresDocker struct {
	pg            *postgres.Handler
	cleanupDocker func() error
}

// NewTestDatabase creates a new database, connects to it, and applies all migrations.
// Afterwards it loads all fixtures from files.
// Use it in integration tests to create a valid database state for your test.
// If there is a file named `testdata/fixtures/_common.yaml`, it's always loaded by default.
// In case of an issue it panics.
// It can be used in parallel and works around the limitations of go-testfixtures/testfixtures.
// If there is a folder `testdata/migrations` it is used to migrate the database up on.
func (pd *PostgresDocker) NewTestDatabase(files ...string) *pgxpool.Pool {
	pgHandler := createAndConnectToNewRandomDatabase(pd.pg)

	const commonFixture = "testdata/fixtures/_common.yaml"

	if _, err := os.Stat(commonFixture); errors.Is(err, nil) { // file exists
		files = append([]string{commonFixture}, files...)
	}

	fixtures, err := testfixtures.New(
		testfixtures.Database(pgHandler.DB),
		testfixtures.Dialect("postgres"),
		testfixtures.FilesMultiTables(files...),
	)
	if err != nil {
		panic(err)
	}

	if err := fixtures.Load(); err != nil {
		panic(err)
	}

	return pgHandler.PGx
}

// Cleanup does shutdown the database connection, stops, and removes the docker image.
// In case of an issue it panics.
func (pd *PostgresDocker) Cleanup() {
	err := pd.pg.Shutdown(context.Background())
	if err != nil {
		panic(err)
	}

	err = pd.cleanupDocker()
	if err != nil {
		panic(err)
	}
}

// PGx returns the pgx connection, if you need to access the database directly.
func (pd *PostgresDocker) PGx() *pgxpool.Pool {
	return pd.pg.PGx
}

func createAndConnectToNewRandomDatabase(pg *postgres.Handler) *postgres.Handler {
	newDB := randomDatabaseName()

	_, err := pg.PGx.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE %s;", newDB))
	if err != nil {
		panic(err)
	}

	newConfig := pg.Config
	newConfig.Database = newDB

	if _, err = os.Stat("testdata/migrations/"); errors.Is(err, nil) { // custom migrations exist
		newConfig.Migrations = os.DirFS("testdata/")
	}

	newHandler, err := postgres.ConnectAndMigrate(context.Background(), newConfig, trace.NewNoopTracerProvider())
	if err != nil {
		panic(err)
	}

	return newHandler
}

func randomDatabaseName() string {
	validPGDatabaseLetters := []rune("abcdefghijklmnopqrstuvwxyz")

	rnd := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec // used for name, not security

	const n = 16
	b := make([]rune, n)

	for i := range b {
		b[i] = validPGDatabaseLetters[rnd.Intn(len(validPGDatabaseLetters))]
	}

	return string(b) + "_test"
}
