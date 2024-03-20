// Use white box testing to ensure the gue log adapter is implemented properly
//
//nolint:testpackage
package jobs

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vgarvardt/gue/v5/adapter"

	"github.com/go-arrower/arrower/alog"
)

func TestGueLogAdapter_Debug(t *testing.T) {
	t.Parallel()

	t.Run("Debug() does not contain bad key", func(t *testing.T) {
		t.Parallel()

		buf, l := setupTestLogger()
		l.Debug("hello", adapter.F("some", "attr"))

		assert.NotEmpty(t, buf.String())
		assert.NotContains(t, buf.String(), "!BADKEY", "mapping of extra fields/attributes should not result in bad key")
		assert.Contains(t, buf.String(), "group.some=attr")
	})

	t.Run("With() does not contain bad key", func(t *testing.T) {
		t.Parallel()

		buf, l := setupTestLogger()
		l2 := l.With(adapter.F("some", "attr"))
		l2.Debug("hello")

		assert.NotContains(t, buf.String(), "!BADKEY", "mapping of extra fields/attributes should not result in bad key")
		assert.Contains(t, buf.String(), "group.some=attr")
	})
}

func setupTestLogger() (*bytes.Buffer, gueLogAdapter) {
	buf := &bytes.Buffer{}
	logger := alog.NewTest(buf)
	alog.Unwrap(logger).SetLevel(alog.LevelDebug)

	logger = logger.WithGroup("group")

	l := gueLogAdapter{
		l: logger,
	}

	return buf, l
}
