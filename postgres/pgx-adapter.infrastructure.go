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

var (
	_ pgx.QueryTracer    = (*pgxTraceAdapter)(nil)
	_ pgx.BatchTracer    = (*pgxTraceAdapter)(nil)
	_ pgx.CopyFromTracer = (*pgxTraceAdapter)(nil)
	_ pgx.PrepareTracer  = (*pgxTraceAdapter)(nil)
	_ pgx.ConnectTracer  = (*pgxTraceAdapter)(nil)
)

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

func (p pgxTraceAdapter) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	if span, ok := ctx.Value(spanKey).(trace.Span); ok {
		span.SetAttributes(attribute.Int64("sql_rows_affected", data.CommandTag.RowsAffected()))

		if data.Err != nil {
			span.SetStatus(codes.Error, data.Err.Error())
		}

		span.End()
	}
}

func (p pgxTraceAdapter) TraceBatchStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceBatchStartData,
) context.Context {
	ctx, span := p.tracer.Start(ctx, "pgx", trace.WithAttributes(
		attribute.String("db_host", conn.Config().Host),
		attribute.Int("db_port", int(conn.Config().Port)),
		attribute.String("db_database", conn.Config().Database),
		attribute.String("db_user", conn.Config().User),
		attribute.Int("batch_len", data.Batch.Len()),
	))

	return context.WithValue(ctx, spanKey, span)
}

func (p pgxTraceAdapter) TraceBatchQuery(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchQueryData) {
	if span, ok := ctx.Value(spanKey).(trace.Span); ok {
		span.SetAttributes(attribute.String("db_host", conn.Config().Host))
		span.SetAttributes(attribute.Int("db_port", int(conn.Config().Port)))
		span.SetAttributes(attribute.String("db_database", conn.Config().Database))
		span.SetAttributes(attribute.String("db_user", conn.Config().User))
		span.SetAttributes(attribute.String("batch_sql", data.SQL))
		span.SetAttributes(attribute.StringSlice("batch_args", anySliceToStrings(data.Args)))
		span.SetAttributes(attribute.Int64("sql_rows_affected", data.CommandTag.RowsAffected()))
		span.SetAttributes(attribute.Bool("sql_select", data.CommandTag.Select()))
		span.SetAttributes(attribute.Bool("sql_insert", data.CommandTag.Insert()))
		span.SetAttributes(attribute.Bool("sql_update", data.CommandTag.Update()))
		span.SetAttributes(attribute.Bool("sql_delete", data.CommandTag.Delete()))
		span.SetAttributes(attribute.String("sql_string", data.CommandTag.String()))
	}
}

func (p pgxTraceAdapter) TraceBatchEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceBatchEndData) {
	if span, ok := ctx.Value(spanKey).(trace.Span); ok {
		if data.Err != nil {
			span.SetStatus(codes.Error, data.Err.Error())
		}

		span.End()
	}
}

func (p pgxTraceAdapter) TraceCopyFromStart(ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceCopyFromStartData,
) context.Context {
	ctx, span := p.tracer.Start(ctx, "pgx", trace.WithAttributes(
		attribute.String("db_host", conn.Config().Host),
		attribute.Int("db_port", int(conn.Config().Port)),
		attribute.String("db_database", conn.Config().Database),
		attribute.String("db_user", conn.Config().User),
		attribute.String("copyfrom_table_name", data.TableName.Sanitize()),
		attribute.StringSlice("copyfrom_column_names", data.ColumnNames),
	))

	return context.WithValue(ctx, spanKey, span)
}

func (p pgxTraceAdapter) TraceCopyFromEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceCopyFromEndData) {
	if span, ok := ctx.Value(spanKey).(trace.Span); ok {
		span.SetAttributes(attribute.Int64("sql_rows_affected", data.CommandTag.RowsAffected()))
		span.SetAttributes(attribute.Bool("sql_select", data.CommandTag.Select()))
		span.SetAttributes(attribute.Bool("sql_insert", data.CommandTag.Insert()))
		span.SetAttributes(attribute.Bool("sql_update", data.CommandTag.Update()))
		span.SetAttributes(attribute.Bool("sql_delete", data.CommandTag.Delete()))
		span.SetAttributes(attribute.String("sql_string", data.CommandTag.String()))

		if data.Err != nil {
			span.SetStatus(codes.Error, data.Err.Error())
		}

		span.End()
	}
}

func (p pgxTraceAdapter) TracePrepareStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TracePrepareStartData,
) context.Context {
	ctx, span := p.tracer.Start(ctx, "pgx", trace.WithAttributes(
		attribute.String("db_host", conn.Config().Host),
		attribute.Int("db_port", int(conn.Config().Port)),
		attribute.String("db_database", conn.Config().Database),
		attribute.String("db_user", conn.Config().User),
		attribute.String("sql", data.SQL),
		attribute.String("name", data.Name),
	))

	return context.WithValue(ctx, spanKey, span)
}

func (p pgxTraceAdapter) TracePrepareEnd(ctx context.Context, _ *pgx.Conn, data pgx.TracePrepareEndData) {
	if span, ok := ctx.Value(spanKey).(trace.Span); ok {
		span.SetAttributes(attribute.Bool("already_prepared", data.AlreadyPrepared))

		if data.Err != nil {
			span.SetStatus(codes.Error, data.Err.Error())
		}

		span.End()
	}
}

func (p pgxTraceAdapter) TraceConnectStart(ctx context.Context, data pgx.TraceConnectStartData) context.Context {
	ctx, span := p.tracer.Start(ctx, "pgx", trace.WithAttributes(
		attribute.String("db_host", data.ConnConfig.Host),
		attribute.Int("db_port", int(data.ConnConfig.Port)),
		attribute.String("db_database", data.ConnConfig.Database),
		attribute.String("db_user", data.ConnConfig.User),
		attribute.Int("db_timeout", int(data.ConnConfig.ConnectTimeout)),
		attribute.StringSlice("runtime_params", mapToStringSlice(data.ConnConfig.RuntimeParams)),
	))

	return context.WithValue(ctx, spanKey, span)
}

func (p pgxTraceAdapter) TraceConnectEnd(ctx context.Context, data pgx.TraceConnectEndData) {
	if span, ok := ctx.Value(spanKey).(trace.Span); ok {
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

func mapToStringSlice(in map[string]string) []string {
	s := []string{}

	for k, v := range in {
		s = append(s, fmt.Sprintf("%s=%s", k, v))
	}

	return s
}
