package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	ctx2 "github.com/go-arrower/arrower/ctx"
)

const spanKey ctx2.CTXKey = "otel_span"

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

//nolint:spancheck // span is closed in TraceQueryEnd. No memory leak
func (p pgxTraceAdapter) TraceQueryStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	ctx, span := p.tracer.Start(ctx, "db.query", trace.WithAttributes(
		attribute.String("db.system.name", "postgresql"),
		attribute.String("server.address", conn.Config().Host),
		attribute.Int("server.port", int(conn.Config().Port)),
		attribute.String("db.namespace", conn.Config().Database),
		attribute.String("db.user", conn.Config().User),
		attribute.String("db.query.text", data.SQL),
		attribute.StringSlice("db.query.parameters", anySliceToStrings(data.Args)),
	))

	return context.WithValue(ctx, spanKey, span)
}

func (p pgxTraceAdapter) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	if span, ok := ctx.Value(spanKey).(trace.Span); ok {
		span.SetAttributes(attribute.Int64("db.response.returned_rows", data.CommandTag.RowsAffected()))

		if data.Err != nil {
			span.SetStatus(codes.Error, data.Err.Error())
		}

		span.End()
	}
}

//nolint:spancheck // span is closed in TraceBatchEnd. No memory leak
func (p pgxTraceAdapter) TraceBatchStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceBatchStartData,
) context.Context {
	ctx, span := p.tracer.Start(ctx, "db.batch", trace.WithAttributes(
		attribute.String("db.system.name", "postgresql"),
		attribute.String("server.address", conn.Config().Host),
		attribute.Int("server.port", int(conn.Config().Port)),
		attribute.String("db.namespace", conn.Config().Database),
		attribute.String("db.user", conn.Config().User),
		attribute.String("db.operation.name", "batch"),
		attribute.Int("db.operation.batch.size", data.Batch.Len()),
	))

	return context.WithValue(ctx, spanKey, span)
}

func (p pgxTraceAdapter) TraceBatchQuery(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchQueryData) {
	if span, ok := ctx.Value(spanKey).(trace.Span); ok {
		span.SetAttributes(
			attribute.String("db.query.text", data.SQL),
			attribute.StringSlice("db.query.parameters", anySliceToStrings(data.Args)),
			attribute.Int64("db.response.returned_rows", data.CommandTag.RowsAffected()),
			attribute.Bool("sql_select", data.CommandTag.Select()),
			attribute.Bool("sql_insert", data.CommandTag.Insert()),
			attribute.Bool("sql_update", data.CommandTag.Update()),
			attribute.Bool("sql_delete", data.CommandTag.Delete()),
			attribute.String("sql_string", data.CommandTag.String()),
		)
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

//nolint:spancheck // span is closed in TraceCopyFromEnd. No memory leak
func (p pgxTraceAdapter) TraceCopyFromStart(ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceCopyFromStartData,
) context.Context {
	ctx, span := p.tracer.Start(ctx, "db.copy", trace.WithAttributes(
		attribute.String("db.system.name", "postgresql"),
		attribute.String("server.address", conn.Config().Host),
		attribute.Int("server.port", int(conn.Config().Port)),
		attribute.String("db.namespace", conn.Config().Database),
		attribute.String("db.user", conn.Config().User),
		attribute.String("copyfrom_table_name", data.TableName.Sanitize()), //nolint:misspell
		attribute.StringSlice("copyfrom_column_names", data.ColumnNames),
	))

	return context.WithValue(ctx, spanKey, span)
}

func (p pgxTraceAdapter) TraceCopyFromEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceCopyFromEndData) {
	if span, ok := ctx.Value(spanKey).(trace.Span); ok {
		span.SetAttributes(
			attribute.Int64("db.response.returned_rows", data.CommandTag.RowsAffected()),
			attribute.Bool("sql_select", data.CommandTag.Select()),
			attribute.Bool("sql_insert", data.CommandTag.Insert()),
			attribute.Bool("sql_update", data.CommandTag.Update()),
			attribute.Bool("sql_delete", data.CommandTag.Delete()),
			attribute.String("sql_string", data.CommandTag.String()),
		)

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
	if span, ok := ctx.Value(spanKey).(trace.Span); ok && span.IsRecording() {
		span.AddEvent("db.prepare.start", trace.WithAttributes(
			attribute.String("db.stored_procedure.name", data.Name),
		))
	}

	return ctx
}

func (p pgxTraceAdapter) TracePrepareEnd(ctx context.Context, _ *pgx.Conn, data pgx.TracePrepareEndData) {
	if span, ok := ctx.Value(spanKey).(trace.Span); ok && span.IsRecording() {
		span.AddEvent("db.prepare.end", trace.WithAttributes(
			attribute.Bool("db.prepared_statement.already_prepared", data.AlreadyPrepared),
		))

		if data.Err != nil {
			span.SetStatus(codes.Error, data.Err.Error())
			span.AddEvent("db.prepare.error", trace.WithAttributes(
				attribute.String("error.message", data.Err.Error()),
			))
		}
	}

	return
}

//nolint:spancheck // span is closed in TraceConnectEnd. No memory leak
func (p pgxTraceAdapter) TraceConnectStart(ctx context.Context, data pgx.TraceConnectStartData) context.Context {
	ctx, span := p.tracer.Start(ctx, "db.connect", trace.WithAttributes(
		attribute.String("db.system.name", "postgresql"),
		attribute.String("server.address", data.ConnConfig.Host),
		attribute.Int("server.port", int(data.ConnConfig.Port)),
		attribute.String("db.namespace", data.ConnConfig.Database),
		attribute.String("db.user", data.ConnConfig.User),
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

	for i := range in {
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
