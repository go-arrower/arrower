package alog

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/go-arrower/arrower"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-arrower/arrower/alog/models"
)

// NewPostgresHandler use this handler in low traffic situations to inspect logs via the arrower admin Context.
func NewPostgresHandler(pgx *pgxpool.Pool, opt *PostgresHandlerOptions) *PostgresHandler {
	// generate json log by writing to local buffer with slog default json
	buf := &bytes.Buffer{}
	jsonLog := slog.HandlerOptions{
		Level:       LevelDebug, // allow all messages, as the level gets controlled by the ArrowerLogger instead.
		AddSource:   false,
		ReplaceAttr: MapLogLevelsToName,
	}

	renderer := slog.NewJSONHandler(buf, &jsonLog)
	queries := models.New(pgx)

	var isAsync bool
	records := make(chan logRecord, 10)
	if opt != nil {
		isAsync = opt.MaxTimeout != 0 || opt.MaxBatchSize != 0

		if isAsync {
			maxTimeout := 2 * time.Second
			if opt.MaxTimeout != 0 {
				maxTimeout = opt.MaxTimeout
			}

			maxBatchSize := 100
			if opt.MaxBatchSize != 0 {
				maxBatchSize = opt.MaxBatchSize
			}

			go func(records chan logRecord, maxBatchSite int, maxTimeout time.Duration) {
				batch := []logRecord{}
				expire := time.After(maxTimeout)

				for {
					select {
					case value := <-records:
						batch = append(batch, value)
						if len(batch) == maxBatchSite {
							_ = saveLogs(context.Background(), queries, batch)
							batch = batch[:0]
						}
					case <-expire:
						_ = saveLogs(context.Background(), queries, batch)
						batch = batch[:0]
					}
				}
			}(records, maxBatchSize, maxTimeout)
		}
	}

	return &PostgresHandler{
		queries:  queries,
		renderer: renderer,
		output:   buf,
		isAsync:  isAsync,
		records:  records,
	}
}

type (
	// PostgresHandlerOptions configures the PostgresHandler.
	PostgresHandlerOptions struct {
		// MaxBatchSize is the maximum number of logs kept before they are saved to the DB.
		// If the maximum is reached before the timeout the batch is transmitted.
		MaxBatchSize int
		// MaxTimeout is the maximum idle time before a batch is saved to the DB, even if
		// it didn't reach the maximum size yet.
		MaxTimeout time.Duration
	}
	PostgresHandler struct {
		queries  *models.Queries
		renderer slog.Handler
		output   *bytes.Buffer

		isAsync bool
		records chan logRecord
	}

	logRecord struct {
		time   time.Time
		userID uuid.NullUUID
		Log    []byte
	}
)

var _ slog.Handler = (*PostgresHandler)(nil)

func (l *PostgresHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (l *PostgresHandler) Handle(ctx context.Context, record slog.Record) error {
	var userID uuid.NullUUID
	if v, ok := ctx.Value(arrower.CtxAuthUserID).(string); ok { // auth.CurrentUserID() // TODO use this instead // ctx key auth.CtxAuthUserID
		userID = uuid.NullUUID{
			UUID:  uuid.MustParse(v),
			Valid: true,
		}
	}

	_ = l.renderer.Handle(ctx, record)
	lRecord := logRecord{
		time:   record.Time,
		userID: userID,
		Log:    l.output.Bytes(),
	}
	l.output.Reset()

	if l.isAsync {
		l.records <- lRecord

		return nil
	}

	err := saveLogs(ctx, l.queries, []logRecord{lRecord})
	if err != nil {
		return fmt.Errorf("could not save log: %v", err)
	}

	return nil
}

func saveLogs(ctx context.Context, queries *models.Queries, logs []logRecord) error {
	params := []models.LogRecordsParams{}

	for _, log := range logs {
		params = append(params, models.LogRecordsParams{
			Time:   pgtype.Timestamptz{Time: log.time, Valid: true},
			UserID: log.userID,
			Log:    log.Log,
		})
	}

	_, err := queries.LogRecords(ctx, params)
	if err != nil {
		return fmt.Errorf("could not save log: %v", err)
	}

	return nil
}

func (l *PostgresHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &PostgresHandler{
		queries:  l.queries,
		renderer: l.renderer.WithAttrs(attrs),
		output:   l.output,
	}
}

func (l *PostgresHandler) WithGroup(name string) slog.Handler {
	return &PostgresHandler{
		queries:  l.queries,
		renderer: l.renderer.WithGroup(name),
		output:   l.output,
	}
}
