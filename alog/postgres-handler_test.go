//go:build integration

package alog_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/tests"
)

var (
	ctx       = context.Background()
	pgHandler *tests.PostgresDocker
)

func TestMain(m *testing.M) {
	pgHandler = tests.GetPostgresDockerForIntegrationTestingInstance()

	//
	// Run tests
	code := m.Run()

	pgHandler.Cleanup()
	os.Exit(code)
}

func TestPostgresHandler_Handle(t *testing.T) {
	t.Parallel()

	t.Run("log to postgres", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		logger := alog.New(
			alog.WithHandler(alog.NewPostgresHandler(pg, nil)),
		)
		ensureLogTableRows(t, pg, 0)

		logger.InfoContext(ctx, "logged msg")
		logger.InfoContext(ctx, "logged msg")
		logger.InfoContext(ctx, "logged msg")

		ensureLogTableRows(t, pg, 3)
	})

	t.Run("async batch", func(t *testing.T) {
		t.Parallel()

		t.Run("log batch of one to postgres", func(t *testing.T) {
			pg := pgHandler.NewTestDatabase()
			logger := alog.New(
				alog.WithHandler(alog.NewPostgresHandler(pg, &alog.PostgresHandlerOptions{
					MaxBatchSize: 1,
				})),
			)

			logger.InfoContext(ctx, "logged msg")

			time.Sleep(10 * time.Millisecond)
			ensureLogTableRows(t, pg, 1)
		})

		t.Run("log batch after timeout", func(t *testing.T) {
			pg := pgHandler.NewTestDatabase()
			logger := alog.New(
				alog.WithHandler(alog.NewPostgresHandler(pg, &alog.PostgresHandlerOptions{
					MaxBatchSize: 10,
					MaxTimeout:   100 * time.Millisecond,
				})),
			)

			logger.InfoContext(ctx, "logged msg")
			logger.InfoContext(ctx, "logged msg")
			logger.InfoContext(ctx, "logged msg")
			ensureLogTableRows(t, pg, 0)

			time.Sleep(150 * time.Millisecond)
			ensureLogTableRows(t, pg, 3)
		})
	})
}

func TestPostgresHandler_WithAttrs(t *testing.T) {
	t.Parallel()

	t.Run("with attrs", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		logger := alog.New(
			alog.WithHandler(alog.NewPostgresHandler(pg, nil)),
		)

		logger = logger.With(slog.String("some", "attr"))
		logger.InfoContext(ctx, "logged msg")

		ensureLogTableRows(t, pg, 1)
		log := getMostRecentLog(pg)

		assert.Equal(t, "logged msg", log["msg"])
		assert.Equal(t, "attr", log["some"])
		assert.Len(t, log, 4, "expected attributes for the log record")
	})
}

func TestPostgresHandler_WithGroup(t *testing.T) {
	t.Parallel()

	t.Run("with group", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		logger := alog.New(
			alog.WithHandler(alog.NewPostgresHandler(pg, nil)),
		)

		logger = logger.WithGroup("someGroup")
		logger.With("some", "attr").InfoContext(ctx, "logged msg")

		ensureLogTableRows(t, pg, 1)
		log := getMostRecentLog(pg)

		assert.Equal(t, "logged msg", log["msg"])
		assert.Equal(t, map[string]any{"some": "attr"}, log["someGroup"])
	})
}

func ensureLogTableRows(t *testing.T, db *pgxpool.Pool, num int) {
	t.Helper()

	var c int
	_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM public.log;`).Scan(&c)

	assert.Equal(t, num, c)
}

func getMostRecentLog(db *pgxpool.Pool) map[string]any {
	var rawLog json.RawMessage

	row := db.QueryRow(ctx, `SELECT log FROM public.log ORDER BY time DESC LIMIT 1;`)
	_ = row.Scan(&rawLog)

	log, _ := rawLog.MarshalJSON()

	var logMap map[string]any
	_ = json.Unmarshal(log, &logMap)

	return logMap
}
