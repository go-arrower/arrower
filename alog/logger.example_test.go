package alog_test

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"

	"github.com/go-arrower/arrower/alog"
)

func Example_logOutputNestingWithTraces() {
	// Manually create a custom TraceID and SpanID for the root span to ensure deterministic IDs for assertion.
	traceID := trace.TraceID([16]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10})
	spanID := trace.SpanID([8]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef})

	newCtx := trace.ContextWithSpanContext(
		context.Background(),
		trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		}),
	)

	logger := alog.New(alog.WithHandler(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: removeTime,
	})))

	logger.InfoContext(newCtx, "")
	logger.InfoContext(newCtx, "", slog.String("key", "val"))

	logger = logger.With("attr", "val")
	logger.InfoContext(newCtx, "", slog.String("key", "val"))

	contextA := logger.WithGroup("context_a")
	contextA.InfoContext(newCtx, "", slog.String("key", "val"))

	contextB := logger.WithGroup("context_b")
	contextB.InfoContext(newCtx, "", slog.String("key", "val"))

	contextBA := contextB.WithGroup("context_a")
	contextBA.InfoContext(newCtx, "", slog.String("key", "val"))

	// Output:
	// level=INFO msg="" traceID=0123456789abcdeffedcba9876543210 spanID=0123456789abcdef
	// level=INFO msg="" key=val traceID=0123456789abcdeffedcba9876543210 spanID=0123456789abcdef
	// level=INFO msg="" attr=val key=val traceID=0123456789abcdeffedcba9876543210 spanID=0123456789abcdef
	// level=INFO msg="" attr=val context_a.key=val context_a.traceID=0123456789abcdeffedcba9876543210 context_a.spanID=0123456789abcdef
	// level=INFO msg="" attr=val context_b.key=val context_b.traceID=0123456789abcdeffedcba9876543210 context_b.spanID=0123456789abcdef
	// level=INFO msg="" attr=val context_b.context_a.key=val context_b.context_a.traceID=0123456789abcdeffedcba9876543210 context_b.context_a.spanID=0123456789abcdef
}

// removeTime to have deterministic output for assertion.
func removeTime(_ []string, attr slog.Attr) slog.Attr {
	if attr.Key == slog.TimeKey {
		attr = slog.Attr{}
	}

	return attr
}
