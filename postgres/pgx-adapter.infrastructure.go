package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-arrower/arrower"
)

const spanKey arrower.CTXKey = "otel_span"

var _ pgx.QueryTracer = (*pgxTraceAdapter)(nil)

type pgxTraceAdapter struct {
	tracer trace.Tracer
}

func (p pgxTraceAdapter) TraceQueryStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	ctx, span := p.tracer.Start(ctx, "pgx", trace.WithAttributes(
		attribute.String("db_host", conn.Config().Host),
		attribute.Int("db_port", int(conn.Config().Port)),
		attribute.String("db_database", conn.Config().Database),
		attribute.String("db_user", conn.Config().User),
		attribute.String("sql", data.SQL),
		attribute.StringSlice("sql_args", anySliceToStrings(data.Args)),
	))

	return context.WithValue(ctx, spanKey, span)
}

func (p pgxTraceAdapter) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	if span, ok := ctx.Value(spanKey).(trace.Span); ok {
		span.SetAttributes(attribute.Int64("sql_rows_affected", data.CommandTag.RowsAffected()))

		if data.Err != nil {
			span.SetStatus(codes.Error, data.Err.Error())
		}

		span.End()
	}
}

func anySliceToStrings(in []any) []string {
	s := make([]string, len(in))

	for i := 0; i < len(in); i++ {
		s[i] = fmt.Sprintf("%v", in[i])
	}

	return s
}
