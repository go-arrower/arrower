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

	"github.com/go-arrower/arrower/postgres"
)

// GetDBConnectionForIntegrationTesting returns a fully connected Handler and a function to clean up
// after the integration tests are done.
// If called in a CI environment, the pipeline needs access to a docker socket.
// If run locally, if spins up and connects to a db instance in a new docker container.
// In case of an issue it panics.
func GetDBConnectionForIntegrationTesting(ctx context.Context) (*postgres.Handler, func() error) {
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
				handler, err := postgres.ConnectAndMigrate(ctx, conf)
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

	return pgHandler, func() error {
		return cleanup()
	}
}

// PrepareTestDatabase creates a new database, connects to it, and applies all migrations.
// Afterwards it loads al fixtures from files.
// Use in integration tests to create a valid database state for your test.
// If there is a file named `testdata/fixtures/_common.yml`, it's always loaded by default.
// In case of an issue it panics.
// It can be used in parallel and works around the limitations of go-testfixtures/testfixtures.
func PrepareTestDatabase(pg *postgres.Handler, files ...string) *postgres.Handler {
	pgHandler := createAndConnectToNewRandomDatabase(pg)

	const commonFixture = "testdata/fixtures/_common.yml"

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

	return pgHandler
}

func createAndConnectToNewRandomDatabase(pg *postgres.Handler) *postgres.Handler {
	newDB := randomDatabaseName()

	_, err := pg.PGx.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE %s;", newDB))
	if err != nil {
		panic(err)
	}

	c := pg.Config
	c.Database = newDB

	newHandler, err := postgres.ConnectAndMigrate(context.Background(), c)
	if err != nil {
		panic(err)
	}

	setupSeedDBSchema(newHandler.PGx)

	return newHandler
}

var validPGDatabaseLetters = []rune("abcdefghijklmnopqrstuvwxyz") //nolint:gochecknoglobals
func randomDatabaseName() string {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec // used for ids, not security

	const n = 16
	b := make([]rune, n)

	for i := range b {
		b[i] = validPGDatabaseLetters[rnd.Intn(len(validPGDatabaseLetters))]
	}

	return string(b) + "_test"
}

func setupSeedDBSchema(pgx *pgxpool.Pool) {
	_, _ = pgx.Exec(context.Background(), `CREATE SCHEMA admin;`)
	_, _ = pgx.Exec(context.Background(), `CREATE TABLE admin.setting (
        setting	TEXT PRIMARY KEY,
    	value   TEXT NOT NULL DEFAULT ''
	);`)

	_, _ = pgx.Exec(context.Background(), `CREATE TABLE "some_table" (
        name	TEXT PRIMARY KEY
	);`)

	_, _ = pgx.Exec(context.Background(), `CREATE TABLE "other_table" (
        name	TEXT PRIMARY KEY
	);`)
}
