package arrower_test

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"

	"github.com/go-arrower/arrower"
)

const (
	userKey        = "user_id"
	userName       = "1337"
	userFormatted  = "user_id=1337"
	applicationMsg = "application message"
)

func TestNewLogHandler(t *testing.T) {
	t.Parallel()

	/*
		~~bare with nothing => default json to Stdout?~~
		~~with loggers, to output to multiple~~
		Assertions
			~~level info by default~~
			~~level can be initialised with WithLevel~~
			source is xy and the right file name & line numbers
			replace attributes is set (for arrower levels)
	*/

	t.Run("default handler exists", func(t *testing.T) {
		t.Parallel()

		h := arrower.NewLogHandler()

		assert.Len(t, h.Handlers(), 1)
		assert.IsType(t, &slog.JSONHandler{}, h.Handlers()[0])
	})

	t.Run("explicit handler replaces default handler", func(t *testing.T) {
		t.Parallel()

		h0 := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})

		h := arrower.NewLogHandler(arrower.WithHandler(h0))
		assert.Len(t, h.Handlers(), 1)
		assert.IsType(t, &slog.TextHandler{}, h.Handlers()[0])
	})

	t.Run("set multiple handlers", func(t *testing.T) {
		t.Parallel()

		h0 := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})
		h1 := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{})

		h := arrower.NewLogHandler(
			arrower.WithHandler(h0),
			arrower.WithHandler(h1),
		)

		assert.Len(t, h.Handlers(), 2)
		assert.IsType(t, &slog.TextHandler{}, h.Handlers()[0])
		assert.IsType(t, &slog.JSONHandler{}, h.Handlers()[1])
	})

	t.Run("info is default level", func(t *testing.T) {
		t.Parallel()

		h := arrower.NewLogHandler()
		assert.Equal(t, slog.LevelInfo, h.Level())
	})

	t.Run("set level", func(t *testing.T) {
		t.Parallel()

		h := arrower.NewLogHandler(arrower.WithLevel(slog.LevelDebug))
		assert.Equal(t, slog.LevelDebug, h.Level())

		h = arrower.NewLogHandler(arrower.WithLevel(arrower.LevelTrace))
		assert.Equal(t, arrower.LevelTrace, h.Level())
	})
}
func TestNewLogger(t *testing.T) {
	t.Parallel()

	t.Run("level info as default level", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		h0 := slog.NewTextHandler(buf, &slog.HandlerOptions{})

		logger := arrower.NewLogger(arrower.WithHandler(h0))

		logger.Log(context.Background(), arrower.LevelDebug, "arrower debug")
		logger.Log(context.Background(), arrower.LevelTrace, "arrower trance")
		logger.Log(context.Background(), slog.LevelDebug, "application debug msg")
		assert.Empty(t, buf.String())

		logger.Info("application info msg")
		assert.Contains(t, buf.String(), `msg="application info msg"`)
	})
}

func TestArrowerLogger_Handle(t *testing.T) {
	t.Parallel()

	t.Run("call each handler", func(t *testing.T) {
		t.Parallel()

		buf0 := &bytes.Buffer{}
		h0 := slog.NewTextHandler(buf0, &slog.HandlerOptions{})
		buf1 := &bytes.Buffer{}
		h1 := slog.NewJSONHandler(buf1, &slog.HandlerOptions{})

		logger := arrower.NewLogger(
			arrower.WithHandler(h0),
			arrower.WithHandler(h1),
		)

		logger.Info(applicationMsg)
		assert.Contains(t, buf0.String(), applicationMsg)
		assert.Contains(t, buf1.String(), applicationMsg)
	})

	// todo case: if one or multiple handlers return an error
}

func TestArrowerLogger_WithAttrs(t *testing.T) {
	t.Parallel()

	t.Run("set attrs for all handlers", func(t *testing.T) {
		t.Parallel()

		buf0 := &bytes.Buffer{}
		h0 := slog.NewTextHandler(buf0, &slog.HandlerOptions{})
		buf1 := &bytes.Buffer{}
		h1 := slog.NewTextHandler(buf1, &slog.HandlerOptions{})

		logger := arrower.NewLogger(arrower.WithHandler(h0), arrower.WithHandler(h1))
		logger.Info("hello")
		assert.NotContains(t, buf0.String(), "some")
		assert.NotContains(t, buf1.String(), "some")

		logger2 := logger.With(slog.String("some", "attr"))
		logger2.Info("hello")
		assert.Contains(t, buf0.String(), "some=attr")
		assert.Contains(t, buf1.String(), "some=attr")
	})

	// todo case: change level in orig logger, see how it does in child logger
}

func TestExtract(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	h0 := slog.NewTextHandler(buf, &slog.HandlerOptions{})

	logger := arrower.NewLogger(arrower.WithHandler(h0))
	_ = logger
	//assert.Len(t, l.Handlers(), 1)
	//assert.IsType(t, &slog.TextHandler{}, h.Handlers()[0])
}

/*
	NewDevelopLogger
	NewTestLogger
	NewProdLogger
	Change level of handler
		Assertions
			for single logger
			for all handlers
			for all handlers ignoring handler specific settings
			for "derived" logger created via WithX
	Logger log
		~~call all children~~
		handle errors in on eof the children handlers
		set level (only top level is used, withLoggers levels are ignored)
		filter for User
		filter for Context
		Assertions
			is enabled for all loggers
			filters work
			traceID is set as attr
			spanID is set as attr
			log event is added to span
			Options like source are working properly
	Logger WithX
		set attr in each child logger
		Assertions
			ensure all children have new attr
			ensure ex. loggers are unchanged

	Open Questions:
		should the methods to change levels and filter be implemented in the arrower logger/handler or in an extra wrapping struct like with the DB?
			don't like the DB-Handler like approach:
				I prototyped it out and it feels bulky and cumbersome to use
				It does not respect the creation of new logger with the WITH-Methods so easily.
			=> helper to cast logger into arrower Logger, so change Level methods become available and local to the logger that uses it
			? DOES THIS WORK FOR ALL DERIVED LOGGERS ?
*/

// --- --- ---

func TestNewDevelopment(t *testing.T) {
	t.Parallel()

	t.Run("log hello", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		h := arrower.NewFilteredLogger(buf)
		h.Logger.Info("hello logger")

		assert.Contains(t, buf.String(), "hello logger")
	})

	t.Run("level debug by default", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		h := arrower.NewFilteredLogger(buf)

		h.Logger.Log(context.Background(), arrower.LevelDebug, "arrower debug")
		h.Logger.Log(context.Background(), arrower.LevelTrace, "arrower trance")
		assert.Empty(t, buf.String())

		h.Logger.Debug("application debug msg")
		assert.Contains(t, buf.String(), `msg="application debug msg"`)
	})
}

func TestSetLevel(t *testing.T) {
	t.Parallel()

	t.Run("set level arrower:debug", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		h := arrower.NewFilteredLogger(buf)

		h.SetLogLevel(arrower.LevelDebug)
		h.Logger.Log(context.Background(), arrower.LevelDebug, "arrower debug")
		assert.Contains(t, buf.String(), `msg="arrower debug"`)
		assert.Contains(t, buf.String(), `level=ARROWER:DEBUG`)
	})

	t.Run("set level arrower:trance", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		h := arrower.NewFilteredLogger(buf)

		h.SetLogLevel(arrower.LevelTrace)
		h.Logger.Log(context.Background(), arrower.LevelTrace, "arrower trace")
		assert.Contains(t, buf.String(), `msg="arrower trace"`)
		assert.Contains(t, buf.String(), `level=ARROWER:TRACE`)
	})
}

func TestFilteredLogger_Handle(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	handler := arrower.NewFilteredLogger(buf)
	logger := handler.Logger

	buf.Reset()
	logger.Debug("message")
	assert.Contains(t, buf.String(), "message")

	buf.Reset()
	logger.Debug("", userKey, "default")
	assert.Contains(t, buf.String(), "default", "not observed user")

	handler.SetLogLevel(slog.LevelError)

	buf.Reset()
	logger.Debug("", userKey, "default")
	assert.NotContains(t, buf.String(), "default", "not observed user")

	buf.Reset()
	logger.Debug("", userKey, userName)
	assert.NotContains(t, buf.String(), userFormatted, "not observed yet")

	buf.Reset()
	handler.SetUserFilter("1337")
	logger.Debug("", userKey, userName)
	assert.Contains(t, buf.String(), userFormatted, "observed user")

	buf.Reset()
	handler.RemoveUserFilter(userName)
	logger.Debug("", userKey, userName)
	assert.NotContains(t, buf.String(), userFormatted, "removed user")

	buf.Reset()
	handler.SetLogLevel(slog.LevelDebug)

	logger = logger.WithGroup("arrower")

	handler.SetUserFilter(userName)
	logger.Debug("", userKey, userName)
	assert.Contains(t, buf.String(), userFormatted, "observed user with group")

	buf.Reset()
	handler.SetLogLevel(slog.LevelDebug)

	logger = logger.With("app", "logger")
	logger.Debug("", userKey, userName)
	assert.Contains(t, buf.String(), "app=logger")
}

func TestFilteredLogger_Enabled(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	handler := arrower.NewFilteredLogger(buf)
	logger := handler.Logger

	handler.SetLogLevel(slog.LevelDebug)
	logger.Debug("message", userKey, userName)
	assert.Contains(t, buf.String(), userFormatted)

	buf.Reset()
	handler.SetLogLevel(slog.LevelError)
	logger.Debug("message", userKey, userName)
	assert.NotContains(t, buf.String(), userFormatted)

	handler.SetUserFilter("1337")
	logger.Debug("message", userKey, userName)
	assert.Contains(t, buf.String(), userFormatted)
}

func TestSomeStuff(t *testing.T) {
	t.Skip()

	buf := &bytes.Buffer{}
	handler := arrower.NewFilteredLogger(buf)
	logger := handler.Logger

	logger = logger.With(
		slog.String("some", "arg"),
		slog.String("other", "arg"),
	)

	handler.SetUserFilter("1337")
	logger.Debug("message", userKey, userName)
	logger.Debug("DONT SHOW message")

	t.Log(buf.String())

	t.Run("", func(t *testing.T) {
		slog.New(nil)
	})
}
