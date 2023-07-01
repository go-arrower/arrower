package alog_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/go-arrower/arrower/alog"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/slog"
)

const (
	userKey        = "user_id"
	userName       = "1337"
	userFormatted  = "user_id=1337"
	applicationMsg = "application message"
)

var errSomething = errors.New("some error")

func TestNewLogger(t *testing.T) {
	t.Parallel()

	t.Run("level info as default level", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		h0 := slog.NewTextHandler(buf, nil)

		logger := alog.New(alog.WithHandler(h0))

		logger.Log(context.Background(), alog.LevelInfo, "arrower debug")
		logger.Log(context.Background(), alog.LevelDebug, "arrower trance")
		logger.Log(context.Background(), slog.LevelDebug, "application debug msg")
		assert.Empty(t, buf.String())

		logger.Info("application info msg")
		assert.Contains(t, buf.String(), `msg="application info msg"`)
	})
}

func TestDefaultOutputFormat(t *testing.T) { //nolint:paralleltest,lll,wsl // concurrent access to os.Stderr will lead to race condition.
	// The default loggers constructed with NewArrowerHandler, NewDevelopment, New
	// log to os.Stderr. This is a good default, but for testing the HandlerOptions,
	// I need to inspect the output.
	// At the same time the API should stay as simple as possible and not take any io.Writer
	// as parameter. To make this possible this tests intercept os.Stderr.

	pipeRead, pipeWrite, _ := os.Pipe()

	stderr := os.Stderr

	os.Stderr = pipeWrite
	defer func() {
		os.Stderr = stderr
	}()

	// logger has to be initialised after os.Stderr is redirected. Otherwise, it will not work.
	// The reason is unclear to me, but because of this we can not loop over a set of loggers.
	// NewDevelopment is not tested, and assumed it just works, if New works.
	logger := alog.New()
	alog.LogHandlerFromLogger(logger).SetLevel(alog.LevelDebug)
	logger.Log(context.Background(), alog.LevelDebug, applicationMsg)

	_ = pipeWrite.Close()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, pipeRead)

	assert.Contains(t, buf.String(), `"source":`)
	assert.Contains(t, buf.String(), `"function":`)
	assert.Contains(t, buf.String(), `"file":`)
	assert.Contains(t, buf.String(), `"line":`)
	assert.Contains(t, buf.String(), `"level":"ARROWER:TRACE"`)
}

func TestNewLogHandler(t *testing.T) {
	t.Parallel()

	t.Run("default handler exists", func(t *testing.T) {
		t.Parallel()

		h := alog.NewArrowerHandler()

		assert.Len(t, h.Handlers(), 1)
		assert.IsType(t, &slog.JSONHandler{}, h.Handlers()[0])
	})

	t.Run("explicit handler replaces default handler", func(t *testing.T) {
		t.Parallel()

		h0 := slog.NewTextHandler(os.Stderr, nil)

		h := alog.NewArrowerHandler(alog.WithHandler(h0))
		assert.Len(t, h.Handlers(), 1)
		assert.IsType(t, &slog.TextHandler{}, h.Handlers()[0])
	})

	t.Run("set multiple handlers", func(t *testing.T) {
		t.Parallel()

		h0 := slog.NewTextHandler(os.Stderr, nil)
		h1 := slog.NewJSONHandler(os.Stderr, nil)

		handler := alog.NewArrowerHandler(
			alog.WithHandler(h0),
			alog.WithHandler(h1),
		)

		assert.Len(t, handler.Handlers(), 2)
		assert.IsType(t, &slog.TextHandler{}, handler.Handlers()[0])
		assert.IsType(t, &slog.JSONHandler{}, handler.Handlers()[1])
	})

	t.Run("info is default level", func(t *testing.T) {
		t.Parallel()

		h := alog.NewArrowerHandler()
		assert.Equal(t, slog.LevelInfo, h.Level())
	})

	t.Run("set level", func(t *testing.T) {
		t.Parallel()

		h := alog.NewArrowerHandler(alog.WithLevel(slog.LevelDebug))
		assert.Equal(t, slog.LevelDebug, h.Level())

		h = alog.NewArrowerHandler(alog.WithLevel(alog.LevelDebug))
		assert.Equal(t, alog.LevelDebug, h.Level())
	})
}

func TestArrowerLogger_SetLevel(t *testing.T) {
	t.Parallel()

	t.Run("log level is changed for all handlers", func(t *testing.T) {
		t.Parallel()

		buf0 := &bytes.Buffer{}
		h0 := slog.NewTextHandler(buf0, nil)
		buf1 := &bytes.Buffer{}
		h1 := slog.NewTextHandler(buf1, nil)

		logger := alog.New(
			alog.WithHandler(h0),
			alog.WithHandler(h1),
		)

		logger.Debug(applicationMsg)
		assert.NotContains(t, buf0.String(), applicationMsg)
		assert.NotContains(t, buf1.String(), applicationMsg)

		alog.LogHandlerFromLogger(logger).SetLevel(slog.LevelDebug)
		logger.Debug(applicationMsg)
		assert.Contains(t, buf0.String(), applicationMsg)
		assert.Contains(t, buf1.String(), applicationMsg)
	})

	t.Run("log level changes for all groups", func(t *testing.T) {
		t.Parallel()

		buf0 := &bytes.Buffer{}
		h0 := slog.NewTextHandler(buf0, nil)
		buf1 := &bytes.Buffer{}
		h1 := slog.NewTextHandler(buf1, nil)

		logger1 := alog.New(alog.WithHandler(h0), alog.WithHandler(h1))
		logger2 := logger1.WithGroup("componentA")

		logger1.Debug("hello0", slog.String("some", "attr"))
		assert.NotContains(t, buf0.String(), "hello0")

		logger2.Debug("hello1", slog.String("some", "attr"))
		assert.NotContains(t, buf1.String(), "hello1")
		assert.NotContains(t, buf1.String(), "componentA.some=attr")

		// change level of logger1 and expect it to change for logger2 as well
		alog.LogHandlerFromLogger(logger1).SetLevel(slog.LevelDebug)
		logger1.Debug("hello0 debug")
		assert.Contains(t, buf0.String(), "hello0 debug")

		logger2.Debug("hello1 debug")
		assert.Contains(t, buf1.String(), "hello1 debug")

		// change level on logger2 and expect it to change for logger1 as well
		alog.LogHandlerFromLogger(logger2).SetLevel(alog.LevelDebug)
		logger1.Debug("hello0 trace")
		assert.Contains(t, buf0.String(), "hello0 trace")

		logger2.Debug("hello1 trace")
		assert.Contains(t, buf1.String(), "hello1 trace")
	})

	t.Run("change levels for loggers with attr", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		h := slog.NewTextHandler(buf, nil)

		logger1 := alog.New(alog.WithHandler(h))
		logger2 := logger1.With(slog.String("some", "attr"))

		logger1.Debug("hello")
		logger2.Debug("hello")
		assert.NotContains(t, buf.String(), "hello")
		assert.NotContains(t, buf.String(), "some")

		alog.LogHandlerFromLogger(logger1).SetLevel(slog.LevelDebug)
		logger1.Debug("hello1")
		logger2.Debug("hello2")

		assert.Contains(t, buf.String(), "hello1")
		assert.Contains(t, buf.String(), "hello2")
		assert.Contains(t, buf.String(), "some=attr")
	})

	t.Run("ignore handler specific level settings", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		h := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelError})

		logger := alog.New(
			alog.WithHandler(h),
		)

		logger.Info(applicationMsg)
		assert.Contains(t, buf.String(), applicationMsg)
	})
}

func TestArrowerLogger_Handle(t *testing.T) {
	t.Parallel()

	t.Run("call each handler", func(t *testing.T) {
		t.Parallel()

		buf0 := &bytes.Buffer{}
		h0 := slog.NewTextHandler(buf0, nil)
		buf1 := &bytes.Buffer{}
		h1 := slog.NewJSONHandler(buf1, nil)

		logger := alog.New(
			alog.WithHandler(h0),
			alog.WithHandler(h1),
		)

		logger.Info(applicationMsg)
		assert.Contains(t, buf0.String(), applicationMsg)
		assert.Contains(t, buf1.String(), applicationMsg)
	})

	t.Run("handler fails", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		h := slog.NewTextHandler(buf, nil)

		logger := alog.New(
			alog.WithHandler(h),
			alog.WithHandler(failingHandler{}),
		)

		logger.Info(applicationMsg)
		assert.Contains(t, buf.String(), applicationMsg)
	})

	t.Run("observability", func(t *testing.T) {
		t.Parallel()

		t.Run("add trace and span IDs", func(t *testing.T) {
			t.Parallel()

			buf := &bytes.Buffer{}
			logger := alog.NewTest(buf)

			logger.Info(applicationMsg)
			assert.NotContains(t, buf.String(), "traceID")

			buf.Reset()
			ctx := trace.ContextWithSpan(context.Background(), &fakeSpan{ID: 1})
			logger.InfoCtx(ctx, applicationMsg)
			assert.Contains(t, buf.String(), "traceID=")
			assert.Contains(t, buf.String(), "spanID=")
		})

		t.Run("add logs to span as event", func(t *testing.T) {
			t.Parallel()

			buf := &bytes.Buffer{}
			logger := alog.NewTest(buf)

			span := fakeSpan{ID: 1}
			ctx := trace.ContextWithSpan(context.Background(), &span)
			logger.ErrorCtx(ctx, applicationMsg)

			assert.Equal(t, "log", span.eventName)
			assert.NotEmpty(t, span.eventOptions)
			assert.Equal(t, codes.Error, span.statusErrorCode)
			assert.Equal(t, applicationMsg, span.statusErrorMsg)
		})
	})
}

func TestArrowerLogger_WithAttrs(t *testing.T) {
	t.Parallel()

	t.Run("set attrs for all handlers", func(t *testing.T) {
		t.Parallel()

		buf0 := &bytes.Buffer{}
		h0 := slog.NewTextHandler(buf0, nil)
		buf1 := &bytes.Buffer{}
		h1 := slog.NewTextHandler(buf1, nil)

		logger1 := alog.New(alog.WithHandler(h0), alog.WithHandler(h1))
		logger2 := logger1.With(slog.String("some", "attr"))

		logger1.Info("hello")
		assert.NotContains(t, buf0.String(), "some")
		assert.NotContains(t, buf1.String(), "some")

		logger2.Info("hello")
		assert.Contains(t, buf0.String(), "some=attr")
		assert.Contains(t, buf1.String(), "some=attr")
	})
}

func TestArrowerLogger_WithGroup(t *testing.T) {
	t.Parallel()

	t.Run("", func(t *testing.T) {
		t.Parallel()

		buf0 := &bytes.Buffer{}
		h0 := slog.NewTextHandler(buf0, nil)
		buf1 := &bytes.Buffer{}
		h1 := slog.NewTextHandler(buf1, nil)

		logger1 := alog.New(alog.WithHandler(h0), alog.WithHandler(h1))
		logger2 := logger1.WithGroup("componentA")

		logger1.Info("hello0", slog.String("some", "attr"))
		assert.NotContains(t, buf0.String(), "componentA")

		logger2.Info("hello1", slog.String("some", "attr"))
		assert.Contains(t, buf1.String(), "componentA.some=attr")
	})
}

func TestLogHandlerFromLogger(t *testing.T) {
	t.Parallel()

	t.Run("arrower logger", func(t *testing.T) {
		t.Parallel()

		logger := alog.New()

		h := alog.LogHandlerFromLogger(logger)
		assert.IsType(t, &alog.ArrowerLogger{}, h)
	})

	t.Run("other slog", func(t *testing.T) {
		t.Parallel()

		logger := slog.Default()

		h := alog.LogHandlerFromLogger(logger)
		assert.IsType(t, &alog.ArrowerLogger{}, h)
		assert.Nil(t, h)
	})
}

// // //  ///  /// /// ///
// // // fakes /// /// ///
// // //  ///  /// /// ///

// fakeSpan is an implementation of Span that is minimal for asserting tests.
type fakeSpan struct { //nolint:govet // fieldalignment does not matter in test cases
	ID byte

	eventName    string
	eventOptions []trace.EventOption

	statusErrorCode codes.Code
	statusErrorMsg  string
}

var _ trace.Span = (*fakeSpan)(nil)

func (fakeSpan) SpanContext() trace.SpanContext {
	return trace.SpanContext{}.WithTraceID([16]byte{1}).WithSpanID([8]byte{1})
}

func (s *fakeSpan) IsRecording() bool { return false }

func (s *fakeSpan) SetStatus(code codes.Code, msg string) {
	s.statusErrorCode = code
	s.statusErrorMsg = msg
}

func (s *fakeSpan) SetError(bool) {}

func (s *fakeSpan) SetAttributes(...attribute.KeyValue) {}

func (s *fakeSpan) End(...trace.SpanEndOption) {}

func (s *fakeSpan) RecordError(error, ...trace.EventOption) {}

func (s *fakeSpan) AddEvent(name string, opts ...trace.EventOption) {
	s.eventName = name
	s.eventOptions = opts
}

func (s *fakeSpan) SetName(string) {}

func (s *fakeSpan) TracerProvider() trace.TracerProvider { return nil } //nolint:ireturn

type failingHandler struct{}

var _ slog.Handler = (*failingHandler)(nil)

func (f failingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (f failingHandler) Handle(ctx context.Context, record slog.Record) error {
	return errSomething
}

func (f failingHandler) WithAttrs(attrs []slog.Attr) slog.Handler { //nolint:ireturn
	panic("implement me")
}

func (f failingHandler) WithGroup(name string) slog.Handler { //nolint:ireturn
	panic("implement me")
}
