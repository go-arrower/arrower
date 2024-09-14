package alog

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test returns a logger tuned for unit testing.
// It exposes a lot of log-specific assertions for the use in tests.
// The interface follows stretchr/testify as close as possible.
//
//   - Every assert func returns a bool indicating whether the assertion was successful or not,
//     this is useful for if you want to go on making further assertions under certain conditions.
func Test(t *testing.T) *TestLogger {
	if t == nil {
		panic("t is nil")
	}

	buf := &testBuffer{
		mu:    sync.Mutex{},
		lines: []*bytes.Buffer{},
	}

	logger := slog.New(newArrowerHandler(
		WithLevel(LevelDebug),
		WithHandler(slog.NewTextHandler(buf, getDebugHandlerOptions())),
	))

	return &TestLogger{
		t:      t,
		Logger: logger,
		buf:    buf,
	}
}

// TestLogger is a special logger for unit testing.
// It exposes all methods of slog and alog and can be injected as a logger dependency.
//
// Additionally, TestLogger exposes a set of assertions on all the lines
// logged with this logger.
type TestLogger struct {
	*slog.Logger

	t   *testing.T
	buf *testBuffer
}

var _ Logger = (*TestLogger)(nil)

var _ ArrowerLogger = (*TestLogger)(nil)

func (l *TestLogger) SetLevel(level slog.Level) {
	Unwrap(l.Logger).SetLevel(level)
}

func (l *TestLogger) Level() slog.Level {
	return Unwrap(l.Logger).Level()
}

func (l *TestLogger) UsesSettings() bool {
	return Unwrap(l.Logger).UsesSettings()
}

// String return the complete log output of ech line logged to TestLogger.
func (l *TestLogger) String() string {
	out := ""

	for _, line := range l.buf.lines {
		out += line.String()
	}

	return out
}

func (l *TestLogger) Lines() []string {
	lines := []string{}

	for _, line := range l.buf.lines {
		lines = append(lines, line.String())
	}

	return lines
}

// Empty asserts that the logger has no lines logged.
func (l *TestLogger) Empty(msgAndArgs ...any) bool {
	l.t.Helper()

	if len(l.buf.lines) > 0 {
		s := "it has 1 line"
		if len(l.buf.lines) > 1 {
			s = fmt.Sprintf("it has %d lines", len(l.buf.lines))
		}

		return assert.Fail(l.t, "logger is not empty, "+s, msgAndArgs...)
	}

	return true
}

// NotEmpty asserts that the logger has at least one line.
func (l *TestLogger) NotEmpty(msgAndArgs ...any) bool {
	l.t.Helper()

	if len(l.buf.lines) == 0 {
		return assert.Fail(l.t, "logger is empty, should not be", msgAndArgs...)
	}

	return true
}

// Contains asserts that at least one line contains the given substring contains.
func (l *TestLogger) Contains(contains string, msgAndArgs ...any) bool {
	l.t.Helper()

	for _, line := range l.buf.lines {
		if strings.Contains(line.String(), contains) {
			return true
		}
	}

	return assert.Fail(l.t, "log output does not have a line which contains: "+contains, msgAndArgs...)
}

// NotContains asserts that no line of the log output contains the given substring notContains.
func (l *TestLogger) NotContains(notContains string, msgAndArgs ...any) bool {
	l.t.Helper()

	for _, line := range l.buf.lines {
		if strings.Contains(line.String(), notContains) {
			return assert.Fail(l.t, "log output contains: "+notContains+", should not be", msgAndArgs...)
		}
	}

	return true
}

// Total asserts that the logger has exactly total number of lines logged.
func (l *TestLogger) Total(total int, msgAndArgs ...any) bool {
	l.t.Helper()

	if len(l.buf.lines) != total {
		return assert.Fail(l.t, fmt.Sprintf("logger does not have %d lines, it has: %d", total, len(l.buf.lines)), msgAndArgs...)
	}

	return true
}

type testBuffer struct {
	mu    sync.Mutex
	lines []*bytes.Buffer
}

func (a *testBuffer) Write(p []byte) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	buf := &bytes.Buffer{}
	n, err := buf.Write(p)

	a.lines = append(a.lines, buf)

	return n, fmt.Errorf("%w", err)
}
