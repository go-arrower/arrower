package alog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/go-arrower/arrower/contexts/auth"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-arrower/arrower/alog/models"
)

var ErrLogFailed = errors.New("could not save log")

// NewPostgresHandler use this handler in low traffic situations
// to inspect logs via the arrower admin Context.
// All logs older than 30 days are deleted when this handler is constructed,
// so the database stays pruned.
// In case pgx is nil the handler returns nil.
func NewPostgresHandler(pgx *pgxpool.Pool, opt *PostgresHandlerOptions) *PostgresHandler {
	if pgx == nil {
		return nil
	}

	// generate json log by writing to local buffer with slog default json
	buf := &bytes.Buffer{} // protect by a mutex?
	jsonLog := slog.HandlerOptions{
		Level:       LevelDebug, // allow all messages, as the level gets controlled by the arrowerHandler instead.
		AddSource:   false,
		ReplaceAttr: MapLogLevelsToName,
	}

	renderer := slog.NewJSONHandler(buf, &jsonLog)
	queries := models.New(pgx)

	const duration = -time.Hour * 24 * 30

	_, err := pgx.Exec(context.Background(), `DELETE FROM arrower.log WHERE time < $1`, time.Now().UTC().Add(duration))
	if err != nil {
		panic("could not prune logs from postgres: " + err.Error())
	}

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
		mu:       sync.Mutex{},
		renderer: renderer,
		output:   buf,
		isAsync:  isAsync,
		records:  records,
	}
}

func processBatches(queries *models.Queries, records chan logRecord, maxBatchSite int, maxTimeout time.Duration) {
	batch := []logRecord{}

	expire := time.NewTicker(maxTimeout)

	for {
		select {
		case value := <-records:
			batch = append(batch, value)
			if len(batch) >= maxBatchSite {
				_ = saveLogs(context.Background(), queries, batch)
				batch = batch[:0]
			}
		case <-expire.C:
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
		queries *models.Queries

		mu       sync.Mutex
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
	if v, ok := ctx.Value(auth.CtxUserID).(string); ok { // auth.CurrentUserID() // TODO use auth.CtxUserID
		userID = uuid.NullUUID{
			UUID:  uuid.MustParse(v),
			Valid: true,
		}
	}

	out, err := l.render(ctx, record)
	if err != nil {
		return fmt.Errorf("%w: could not render the record: %w", ErrLogFailed, err)
	}

	lRecord := logRecord{
		time:   record.Time,
		userID: userID,
		Log:    out,
	}

	if l.isAsync {
		l.records <- lRecord

		return nil
	}

	err = saveLogs(ctx, l.queries, []logRecord{lRecord})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrLogFailed, err) //nolint:errorlint // prevent err in api
	}

	return nil
}

// render is used to create a valid json of the record.
// Relying on renderer.Handle which is using a slog.JSONHandler does sometimes result in invalid json.
// And postgres returns with a 22P02 error.
// The reason is unclear to me (no visual indicators): converting from and to json again seams to work.
func (l *PostgresHandler) render(ctx context.Context, record slog.Record) ([]byte, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	err := l.renderer.Handle(ctx, record)
	if err != nil {
		return nil, fmt.Errorf("could not handle: %v", err) //nolint:errorlint,goerr113 // prevent err in api
	}

	var nmap map[string]interface{}

	err = json.Unmarshal(l.output.Bytes(), &nmap)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal: %v", err) //nolint:errorlint,goerr113 // prevent err in api
	}

	output, err := json.Marshal(nmap)
	if err != nil {
		return nil, fmt.Errorf("could not marshal: %v", err) //nolint:errorlint,goerr113 // prevent err in api
	}

	l.output.Reset()

	return output, nil
}

func saveLogs(ctx context.Context, queries *models.Queries, logs []logRecord) error {
	if len(logs) == 0 {
		return nil
	}

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
		mu:       sync.Mutex{},
		renderer: l.renderer.WithAttrs(attrs),
		output:   l.output,
		isAsync:  l.isAsync,
		records:  l.records,
	}
}

func (l *PostgresHandler) WithGroup(name string) slog.Handler {
	return &PostgresHandler{
		queries:  l.queries,
		mu:       sync.Mutex{},
		renderer: l.renderer.WithGroup(name),
		output:   l.output,
		isAsync:  l.isAsync,
		records:  l.records,
	}
}
