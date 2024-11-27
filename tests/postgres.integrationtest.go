//go:build integration

package tests

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-testfixtures/testfixtures/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/khaiql/dbcleaner"
	"github.com/khaiql/dbcleaner/engine"
	"github.com/ory/dockertest/v3"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/go-arrower/arrower/postgres"
)

//nolint:gochecknoglobals // the variables are used on purpose for a singleton pattern.
var (
	muPostgres        = &sync.Mutex{}
	singletonPostgres *PostgresDocker
)

// PostgresOpt allows to initialise a custom postgres connection.
type PostgresOpt func(logger *postgres.Config)

// WithMigrations applies all migrations in path.
func WithMigrations(dir string) PostgresOpt {
	if dir == "" {
		dir = "shared/infrastructure/postgres" // arrower default location
	}

	return func(c *postgres.Config) {
		c.Migrations = os.DirFS(path.Join(getProjectRoot(), dir))
	}
}

// GetPostgresDockerForIntegrationTestingInstance returns a fully connected handler.
// Subsequent calls return the same handler to prevent multiple docker containers to spin up,
// if you have a lot of integration tests running in parallel.
func GetPostgresDockerForIntegrationTestingInstance(opts ...PostgresOpt) *PostgresDocker {
	var (
		pgHandler *postgres.Handler

		retryFunc = func(resource *dockertest.Resource) func() error {
			port, _ := strconv.Atoi(resource.GetPort("5432/tcp"))
			conf := defaultPGConf
			conf.Port = port

			for _, opt := range opts {
				opt(&conf)
			}

			return func() error {
				handler, err := postgres.ConnectAndMigrate(context.Background(), conf, noop.NewTracerProvider())
				if err != nil {
					return err //nolint:wrapcheck
				}

				pgHandler = handler

				return nil
			}
		}
	)

	muPostgres.Lock()
	defer muPostgres.Unlock()

	if singletonPostgres != nil {
		return singletonPostgres
	}

	options := defaultPGRunOptions
	options.Name = fmt.Sprintf("arrower-testing-postgres-%d", rand.Intn(1000)) //nolint:gosec,mnd,lll // no need for secure number, just prevent collisions

	cleanup, err := StartDockerContainer(options, retryFunc)
	if err != nil {
		panic(err)
	}

	singletonPostgres = &PostgresDocker{
		pg:            pgHandler,
		cleanupDocker: cleanup,
	}

	return singletonPostgres
}

// NewPostgresDockerForIntegrationTesting returns a Handler that is fully connected and has a method to clean up
// after the integration tests are done. Consider using GetPostgresDockerForIntegrationTestingInstance.
// It spins up and connects to a db instance in a new docker container.
// If called in a CI environment, the pipeline needs access to a docker socket.
// In case of an issue, it panics.
func NewPostgresDockerForIntegrationTesting() *PostgresDocker {
	var (
		pgHandler *postgres.Handler

		retryFunc = func(resource *dockertest.Resource) func() error {
			port, _ := strconv.Atoi(resource.GetPort("5432/tcp"))
			conf := defaultPGConf
			conf.Port = port

			return func() error {
				handler, err := postgres.ConnectAndMigrate(context.Background(), conf, noop.NewTracerProvider())
				if err != nil {
					return err //nolint:wrapcheck
				}

				pgHandler = handler

				return nil
			}
		}
	)

	cleanup, err := StartDockerContainer(defaultPGRunOptions, retryFunc)
	if err != nil {
		panic(err)
	}

	return &PostgresDocker{
		pg:            pgHandler,
		cleanupDocker: cleanup,
	}
}

var (
	defaultPGConf = postgres.Config{ //nolint:gochecknoglobals,exhaustruct
		User:       "arrower",
		Password:   "secret",
		Database:   "arrower_test",
		Host:       "localhost",
		Port:       5432, //nolint:mnd
		MaxConns:   10,   //nolint:mnd
		Migrations: postgres.ArrowerDefaultMigrations,
	}

	defaultPGRunOptions = &dockertest.RunOptions{ //nolint:gochecknoglobals,exhaustruct // only set required configuration
		Repository: "ghcr.io/go-arrower/postgres",
		Tag:        "latest",
		Env: []string{
			"POSTGRES_USER=" + defaultPGConf.User,
			"POSTGRES_PASSWORD=" + defaultPGConf.Password,
			"POSTGRES_DB=" + defaultPGConf.Database,
			"listen_addresses = '*'",
			"MaxConnections=1000",
		},
		Cmd: []string{"-c", "max_connections=1000"},
	}
)

type PostgresDocker struct {
	pg            *postgres.Handler
	cleanupDocker func() error
}

const commonFixture = "testdata/fixtures/_common.yaml"

// NewTestDatabase creates a new database, connects to it, and applies all migrations.
// Afterwards, it loads all fixtures from files.
// Use it in integration tests to create a valid database state for your test.
// If there is a file named `testdata/fixtures/_common.yaml`, it's always loaded by default.
// In case of an issue, it panics.
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

// PrepareDatabase prepares the existing database for testing:
// - All tables will be truncated
// - All fixture files are allied
// - If there is a file named `testdata/fixtures/_common.yaml`, it's always loaded by default.
// TODO: automatically search for files in two locations: given one and if not exist with auto prefix: testdata/fixtures/
func (pd *PostgresDocker) PrepareDatabase(files ...string) {
	{ // truncate all tables
		c := pd.pg.Config
		dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
			c.User, c.Password, net.JoinHostPort(c.Host, strconv.Itoa(c.Port)), c.Database)

		cleaner := dbcleaner.New()
		cleaner.SetEngine(engine.NewPostgresEngine(dsn))

		var tables []string // get all tables, that will be truncated.
		_ = pgxscan.Select(context.Background(), pd.PGx(), &tables,
			`SELECT table_schema || '.' || table_name 
					FROM information_schema.tables 
					WHERE table_schema NOT IN ('pg_catalog', 'information_schema') 
					  AND table_type = 'BASE TABLE' 
					  AND table_name <> 'schema_migrations'`,
		)

		cleaner.Clean(tables...)
		_ = cleaner.Close()
	}

	if _, err := os.Stat(commonFixture); errors.Is(err, nil) { // file exists
		files = append([]string{commonFixture}, files...)
	}

	fixtures, err := testfixtures.New(
		testfixtures.Database(pd.pg.DB),
		testfixtures.Dialect("postgres"),
		testfixtures.FilesMultiTables(files...),
	)
	if err != nil {
		panic(err)
	}

	if err := fixtures.Load(); err != nil {
		panic(err)
	}
}

// Cleanup does shutdown the database connection, stops, and removes the docker image.
// It cannot be deferred in TestMain, if it exists with os.Exit(code), as that does not execute the defer stack.
// In case of an issue, it panics.
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

// PGx returns the pgx connection if you need to access the database directly.
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

	newHandler, err := postgres.ConnectAndMigrate(context.Background(), newConfig, noop.NewTracerProvider())
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

// Go tests are called from the path of the calling test's package.
// This changes the current directory of each test and will
// make it hard to find relative test data.
// This helper will find the files independent of the caller's path,
// starting from the project root.
func getProjectRoot() string {
	cmdOut, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		panic(err)
	}

	return strings.TrimSpace(string(cmdOut)) + "/"
}
