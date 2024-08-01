package jobs

import (
	"testing"

	"github.com/vgarvardt/gue/v5/adapter"

	"github.com/go-arrower/arrower/alog"
)

func TestGueLogAdapter_Debug(t *testing.T) {
	t.Parallel()

	t.Run("Debug() does not contain bad key", func(t *testing.T) {
		t.Parallel()

		logger := alog.Test(t)
		alog.Unwrap(logger).SetLevel(alog.LevelDebug)
		l := gueLogAdapter{
			l: logger.WithGroup("group"),
		}
		l.Debug("hello", adapter.F("some", "attr"))

		logger.NotEmpty()
		logger.NotContains("!BADKEY", "mapping of extra fields/attributes should not result in bad key")
		logger.Contains("group.some=attr")
	})

	t.Run("With() does not contain bad key", func(t *testing.T) {
		t.Parallel()

		logger := alog.Test(t)
		alog.Unwrap(logger).SetLevel(alog.LevelDebug)
		l := gueLogAdapter{
			l: logger.WithGroup("group"),
		}

		l2 := l.With(adapter.F("some", "attr"))
		l2.Debug("hello")

		logger.NotContains("!BADKEY", "mapping of extra fields/attributes should not result in bad key")
		logger.Contains("group.some=attr")
	})
}
