package alog

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/go-arrower/arrower"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5/pgtype"
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

	return &PostgresHandler{
		queries:  models.New(pgx),
		renderer: renderer,
		output:   buf,
	}
}

type (
	PostgresHandlerOptions struct{}
	PostgresHandler        struct {
		queries  *models.Queries
		renderer slog.Handler
		output   *bytes.Buffer
	}
)

var _ slog.Handler = (*PostgresHandler)(nil)

func (l *PostgresHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (l *PostgresHandler) Handle(ctx context.Context, record slog.Record) error {
	_ = l.renderer.Handle(ctx, record)

	var userID uuid.NullUUID
	if v, ok := ctx.Value(arrower.CtxAuthUserID).(string); ok { // auth.CurrentUserID() // TODO use this instead // ctx key auth.CtxAuthUserID
		userID = uuid.NullUUID{
			UUID:  uuid.MustParse(v),
			Valid: true,
		}
	}
	fmt.Println(userID)

	err := l.queries.LogRecord(ctx, models.LogRecordParams{
		Time:   pgtype.Timestamptz{Time: record.Time, Valid: true},
		UserID: userID,
		Log:    l.output.Bytes(),
	})
	if err != nil {
		return fmt.Errorf("could not save log: %v", err)
	}
	l.output.Reset()

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
