// Package logging is a dedicated package for logging attributes.
//
// It enforces a consistent way of naming logging attributes,
// and prevents traceID, trace_id, TraceID being in the logs at the same time.
// It gives a broad view of what attributes are available,
// useful when setting up indexes and queries in dashboards.
//
// Although logging is a cross-cutting and not a business concern
// the package is placed in the domain layer for pragmatism.
// As it basically only depends on the standard library's slog,
// it is considered without risk.
package logging

import (
	"log/slog"
	"time"

	"github.com/go-arrower/arrower/alog/logging"
)

var (
	Error   = logging.Error
	Context = logging.Context
	Attr    = logging.Attr
)

func Email(email string) slog.Attr {
	return slog.String("email", email)
}

func IP(ip string) slog.Attr {
	return slog.String("ip", ip)
}

func Time(t time.Time) slog.Attr {
	return slog.String("time", t.String())
}

func Device(device string) slog.Attr {
	return slog.String("device", device)
}

func Token(token string) slog.Attr {
	return slog.String("token", token)
}
