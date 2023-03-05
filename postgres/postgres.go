package postgres

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrConnectionFailed = errors.New("connection failed")

// Config holds all values used to configure and connect to a postgres database.
type Config struct {
	User          string
	Password      string
	Database      string
	MigrationPath string
	Host          string
	Port          int
	MaxConns      int
}

func (c Config) toURL() string {
	if c.MaxConns == 0 { // prevent pool_max_conns too small error
		c.MaxConns = 10
	}

	return fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable&pool_max_conns=%d",
		c.User, c.Password, net.JoinHostPort(c.Host, strconv.Itoa(c.Port)), c.Database, c.MaxConns)
}

// Connect connects to a PostgreSQL database.
func Connect(conf Config) (*Handler, error) {
	ctx := context.Background()

	c, err := pgxpool.ParseConfig(conf.toURL())
	if err != nil {
		return nil, fmt.Errorf("%w: could not parse config: %v", ErrConnectionFailed, err)
	}

	dbpool, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("%w: could not connect: %v", ErrConnectionFailed, err)
	}

	err = dbpool.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("%w: could not ping db: %v", ErrConnectionFailed, err)
	}

	return &Handler{PGx: dbpool}, nil
}

type Handler struct {
	PGx *pgxpool.Pool
}

func (h Handler) Shutdown(ctx context.Context) error {
	h.PGx.Close()

	return nil
}
