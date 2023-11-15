package alog

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-arrower/arrower"
	"github.com/go-arrower/arrower/alog/models"
)

var ErrLogFailed = errors.New("could not save log")

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

	const (
		defaultLogChanBuffer = 10
		defaultTimeout       = 2 * time.Second
		defaultBatchSize     = 100
	)

	records := make(chan logRecord, defaultLogChanBuffer)

	if opt != nil { //nolint:nestif
		isAsync = opt.MaxTimeout != 0 || opt.MaxBatchSize != 0

		if isAsync {
			maxTimeout := defaultTimeout
			if opt.MaxTimeout != 0 {
				maxTimeout = opt.MaxTimeout
			}

			maxBatchSize := defaultBatchSize
			if opt.MaxBatchSize != 0 {
				maxBatchSize = opt.MaxBatchSize
			}

			go processBatches(queries, records, maxBatchSize, maxTimeout)
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

func processBatches(queries *models.Queries, records chan logRecord, maxBatchSite int, maxTimeout time.Duration) {
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
	PostgresHandler struct { //nolint:govet // accept fieldalignment
		queries  *models.Queries
		renderer slog.Handler
		output   *bytes.Buffer

		isAsync bool
		records chan logRecord
	}

	logRecord struct { //nolint:govet // accept fieldalignment
		time   time.Time
		userID uuid.NullUUID
		Log    []byte
	}
)

var _ slog.Handler = (*PostgresHandler)(nil)

func (l *PostgresHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (l *PostgresHandler) Handle(ctx context.Context, record slog.Record) error {
	var userID uuid.NullUUID
	if v, ok := ctx.Value(arrower.CtxAuthUserID).(string); ok { // auth.CurrentUserID() // TODO use auth.CtxAuthUserID
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
		return fmt.Errorf("%w: %v", ErrLogFailed, err) //nolint:errorlint // prevent err in api
	}

	return nil
}

func saveLogs(ctx context.Context, queries *models.Queries, logs []logRecord) error {
	params := []models.LogRecordsParams{}

	for _, log := range logs {
		params = append(params, models.LogRecordsParams{
			Time:   pgtype.Timestamptz{Time: log.time, Valid: true, InfinityModifier: pgtype.Finite},
			UserID: log.userID,
			Log:    log.Log,
		})
	}

	_, err := queries.LogRecords(ctx, params)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrLogFailed, err) //nolint:errorlint // prevent err in api
	}

	return nil
}

func (l *PostgresHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &PostgresHandler{
		queries:  l.queries,
		renderer: l.renderer.WithAttrs(attrs),
		output:   l.output,
		isAsync:  l.isAsync,
		records:  l.records,
	}
}

func (l *PostgresHandler) WithGroup(name string) slog.Handler {
	return &PostgresHandler{
		queries:  l.queries,
		renderer: l.renderer.WithGroup(name),
		output:   l.output,
		isAsync:  l.isAsync,
		records:  l.records,
	}
}
