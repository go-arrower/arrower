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

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

var (
	ErrConnectionFailed = errors.New("connection failed")
	ErrMigrationFailed  = errors.New("migration failed")
)

// Config holds all values used to configure and connect to a postgres database.
type Config struct {
	User     string
	Password string
	Database string
	Host     string
	Port     int
	MaxConns int
}

func (c Config) toURL() string {
	if c.MaxConns == 0 { // prevent error: pool_max_conns too small
		c.MaxConns = 10
	}

	return fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable&pool_max_conns=%d",
		c.User, c.Password, net.JoinHostPort(c.Host, strconv.Itoa(c.Port)), c.Database, c.MaxConns)
}

// Connect connects to a PostgreSQL database.
func Connect(ctx context.Context, c Config) (*Handler, error) {
	config, err := pgxpool.ParseConfig(c.toURL())
	if err != nil {
		return nil, fmt.Errorf("%w: could not parse config: %v", ErrConnectionFailed, err)
	}

	dbpool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("%w: could not connect: %v", ErrConnectionFailed, err)
	}

	err = dbpool.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: could not ping db: %v", ErrConnectionFailed, err)
	}

	connStr := stdlib.RegisterConnConfig(config.ConnConfig)

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("%w: could not connect via the std lib registration: %v", ErrConnectionFailed, err)
	}

	return &Handler{PGx: dbpool, DB: db}, nil
}

// ConnectAndMigrate connects to a PostgreSQL database and
// runs all migrations to ensure that the schema is on the latest version.
func ConnectAndMigrate(ctx context.Context, conf Config) (*Handler, error) {
	handler, err := Connect(ctx, conf)
	if err != nil {
		return nil, err
	}

	err = migrateUp(handler.DB, conf.Database, migrationsFS)
	if err != nil {
		return nil, err
	}

	return handler, nil
}

func migrateUp(db *sql.DB, dbName string, migrationsFS fs.FS) error {
	fsDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("%w: could not create migration file driver: %v", ErrMigrationFailed, err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{}) //nolint:exhaustruct // use default config
	if err != nil {
		return fmt.Errorf("%w: could not get database driver: %v", ErrMigrationFailed, err)
	}

	m, err := migrate.NewWithInstance("", fsDriver, dbName, driver)
	if err != nil {
		return fmt.Errorf("%w: could not create new migration instance: %v", ErrMigrationFailed, err)
	}

	err = m.Up()
	if err != nil {
		return fmt.Errorf("%w: could not migrate up: %v", ErrMigrationFailed, err)
	}

	return nil
}

type Handler struct {
	PGx *pgxpool.Pool
	DB  *sql.DB // keep a sql.DB connection around for migration & integration test cases, e.g. setting up test fixtures.
}

// Shutdown waits & closes all connections to PostgreSQL.
func (h Handler) Shutdown(ctx context.Context) error {
	h.PGx.Close()

	return nil
}
