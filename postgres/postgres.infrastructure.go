package postgres

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"strconv"

	ctx2 "github.com/go-arrower/arrower/ctx"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"go.opentelemetry.io/otel/trace"
)

// CtxTX contains a database transaction, only if set by e.g. a middleware.
const CtxTX ctx2.CTXKey = "arrower.tx"

//go:embed migrations/*.sql
var ArrowerDefaultMigrations embed.FS

var (
	ErrConnectionFailed = errors.New("connection failed")
	ErrMigrationFailed  = errors.New("migration failed")
)

// Config holds all values used to configure and connect to a postgres database.
type Config struct {
	Migrations fs.FS
	User       string
	Password   string
	Database   string
	SSLMode    string
	Host       string
	Port       int
	MaxConns   int
}

func (c Config) toURL() string {
	if c.MaxConns == 0 { // prevent error: pool_max_conns too small
		c.MaxConns = 10
	}

	return fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s&pool_max_conns=%d",
		c.User, c.Password, net.JoinHostPort(c.Host, strconv.Itoa(c.Port)), c.Database, c.SSLMode, c.MaxConns)
}

// Connect connects to a PostgreSQL database.
func Connect(ctx context.Context, pgConf Config, tracerProvider trace.TracerProvider) (*Handler, error) {
	config, err := pgxpool.ParseConfig(pgConf.toURL())
	if err != nil {
		return nil, fmt.Errorf("%w: could not parse config: %v", ErrConnectionFailed, err) //nolint:errorlint,lll // prevent err in api
	}

	// to list all runtime settings: SHOW ALL;
	config.ConnConfig.RuntimeParams = map[string]string{
		"application_name": "arrower/app",

		// The first schema is where new tables will be created if CREATE TABLE command does not specify a schema name,
		// see https://www.postgresql.org/docs/16/ddl-schemas.html
		"search_path": "public,arrower",
	}
	config.ConnConfig.Tracer = &pgxTraceAdapter{
		tracer: tracerProvider.Tracer("arrower.pgx"),
	}

	dbpool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("%w: could not connect: %v", ErrConnectionFailed, err) //nolint:errorlint,lll // prevent err in api
	}

	err = dbpool.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: could not ping db: %v", ErrConnectionFailed, err) //nolint:errorlint,lll // prevent err in api
	}

	connStr := stdlib.RegisterConnConfig(config.ConnConfig) // offer std SQL if a library would need it.

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("%w: could not connect via the std lib registration: %v", ErrConnectionFailed, err) //nolint:errorlint,lll // prevent err in api
	}

	return &Handler{
		PGx:    dbpool,
		DB:     db,
		Config: pgConf,
	}, nil
}

// ConnectAndMigrate connects to a PostgreSQL database and
// runs all migrations to ensure that the schema is on the latest version.
func ConnectAndMigrate(ctx context.Context, conf Config, tracerProvider trace.TracerProvider) (*Handler, error) {
	if conf.Migrations == nil {
		return nil, fmt.Errorf("%w: no migration files given", ErrMigrationFailed)
	}

	handler, err := Connect(ctx, conf, tracerProvider)
	if err != nil {
		return nil, err
	}

	err = migrateUp(handler.DB, conf.Database, conf.Migrations)
	if err != nil {
		return nil, err
	}

	return handler, nil
}

func migrateUp(db *sql.DB, dbName string, migrationsFS fs.FS) error {
	fsDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("%w: could not create migration file driver: %v", ErrMigrationFailed, err) //nolint:errorlint,lll // prevent err in api
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{}) //nolint:exhaustruct // use default config
	if err != nil {
		return fmt.Errorf("%w: could not get database driver: %v", ErrMigrationFailed, err) //nolint:errorlint,lll // prevent err in api
	}

	m, err := migrate.NewWithInstance("", fsDriver, dbName, driver)
	if err != nil {
		return fmt.Errorf("%w: could not create new migration instance: %v", ErrMigrationFailed, err) //nolint:errorlint,lll // prevent err in api
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("%w: could not migrate up: %v", ErrMigrationFailed, err) //nolint:errorlint,lll // prevent err in api
	}

	return nil
}

type Handler struct {
	PGx    *pgxpool.Pool
	DB     *sql.DB // keep a sql.DB connection around for migration & integration tests, e.g. setting up test fixtures.
	Config Config
}

// Shutdown waits & closes all connections to PostgreSQL.
func (h Handler) Shutdown(_ context.Context) error {
	h.PGx.Close()

	return nil
}
