package alog_test

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/alog/logging"
)

func Example_logOutputNestingWithTraces() {
	ctx := trace.ContextWithSpanContext(
		context.Background(),
		trace.NewSpanContext(trace.SpanContextConfig{
			// Manually create IDs for the root span to ensure deterministic values for assertion.
			TraceID: [16]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10},
			SpanID:  [8]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef},
		}),
	)

	logger := alog.New(alog.WithHandler(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: removeTime})))

	logger.InfoContext(ctx, "")
	logger.InfoContext(ctx, "", logging.Attr("key", "val"))

	logger = logger.With("attr", "val")
	logger.InfoContext(ctx, "", logging.Attr("key", "val"))

	contextA := logger.WithGroup("context_a")
	contextA.InfoContext(ctx, "", logging.Attr("key", "val"))

	contextB := logger.WithGroup("context_b")
	contextB.InfoContext(ctx, "", logging.Attr("key", "val"))

	contextBA := contextB.WithGroup("context_a")
	contextBA.InfoContext(ctx, "", logging.Attr("key", "val"))

	// Output:
	// level=INFO msg="" trace_id=0123456789abcdeffedcba9876543210 span_id=0123456789abcdef
	// level=INFO msg="" key=val trace_id=0123456789abcdeffedcba9876543210 span_id=0123456789abcdef
	// level=INFO msg="" attr=val key=val trace_id=0123456789abcdeffedcba9876543210 span_id=0123456789abcdef
	// level=INFO msg="" attr=val context_a.key=val context_a.trace_id=0123456789abcdeffedcba9876543210 context_a.span_id=0123456789abcdef
	// level=INFO msg="" attr=val context_b.key=val context_b.trace_id=0123456789abcdeffedcba9876543210 context_b.span_id=0123456789abcdef
	// level=INFO msg="" attr=val context_b.context_a.key=val context_b.context_a.trace_id=0123456789abcdeffedcba9876543210 context_b.context_a.span_id=0123456789abcdef
}

// removeTime to have deterministic output for assertion.
func removeTime(_ []string, attr slog.Attr) slog.Attr {
	if attr.Key == slog.TimeKey {
		attr = slog.Attr{}
	}

	return attr
}
