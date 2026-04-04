// Package logging is a dedicated package for logging attributes.
//
// It gives a broad view of what attributes are available,
// useful when setting up indexes and queries in dashboards.
// It enforces a consistent way of naming logging attributes,
// and prevents traceID, trace_id, TraceID in the logs at the same time.
//
// Although logging is a cross-cutting and not a business concern
// the package is placed in the domain layer for pragmatism.
// As it basically only depends on the standard library's slog,
// it is considered without risk.
package logging

import (
	"log/slog"

	"github.com/go-arrower/arrower/alog/logging"
)

var (
	Error   = logging.Error
	Context = logging.Context
	Attr    = logging.Attr
)

func JobHistorySetting(setting any) slog.Attr {
	return slog.Any("setting", setting)
}
