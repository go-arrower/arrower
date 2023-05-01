package log

import (
	"io"

	"golang.org/x/exp/slog"
)

const (
	// ArrowerLevelDebug is used for arrower debug messages.
	ArrowerLevelDebug = slog.Level(-8)

	// ArrowerLevelTrace is used for arrower trace information, if you really want to know what is going on.
	ArrowerLevelTrace = slog.Level(-12)
)

// SetLevel allows to set the global level for the loggers.
func SetLevel(l slog.Level) {
	arrowerLevel.Set(l)
}

// NewDevelopment provides a convenient logger for the development.
func NewDevelopment(w io.Writer) *slog.Logger {
	arrowerLevel.Set(slog.LevelDebug)

	opts := slog.HandlerOptions{
		Level:       arrowerLevel,
		AddSource:   false,
		ReplaceAttr: nameArrowerLogLevels,
	}.NewTextHandler(w)

	return slog.New(opts)
}

//nolint:gochecknoglobals // recommended by slog documentation.
var (
	// arrowerLevel controls the level of the logger globally. Info by default.
	arrowerLevel = new(slog.LevelVar)

	// levelNames maps the arrower log levels to human readable names.
	levelNames = map[slog.Leveler]string{
		ArrowerLevelDebug: "Arrower:DEBUG",
		ArrowerLevelTrace: "Arrower:TRACE",
	}
)

func nameArrowerLogLevels(groups []string, attr slog.Attr) slog.Attr {
	if attr.Key == slog.LevelKey {
		level, _ := attr.Value.Any().(slog.Level)

		levelLabel, exists := levelNames[level]
		if !exists {
			levelLabel = level.String()
		}

		attr.Value = slog.StringValue(levelLabel)
	}

	return attr
}
