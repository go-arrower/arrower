package alog

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/afiskon/promtail-client/promtail"
)

type (
	LokiHandlerOptions struct {
		PushURL string
		Label   string
	}

	// LokiHandler, see NewLokiHandler.
	LokiHandler struct {
		client   promtail.Client
		renderer slog.Handler
		output   *bytes.Buffer
	}
)

var _ slog.Handler = (*LokiHandler)(nil)

func (l LokiHandler) Handle(ctx context.Context, record slog.Record) error {
	_ = l.renderer.Handle(ctx, record)

	var attrs string

	record.Attrs(func(a slog.Attr) bool {
		// this is high cardinality and can kill loki
		// https://grafana.com/docs/loki/latest/fundamentals/labels/#cardinality
		attrs += fmt.Sprintf(",%s=%s ", a.Key, a.Value.String())

		return true // process next attr
	})

	attrs = strings.TrimPrefix(attrs, ",")
	attrs = strings.TrimSpace(attrs)

	l.client.Infof(record.Message + " " + attrs) // in grafana: green
	l.client.Infof(l.output.String())            // in grafana: green

	l.output.Reset()

	// client.Debugf(record.Message) // in grafana: blue
	// client.Errorf(record.Message) // in grafana: red
	// client.Warnf(record.Message)  // in grafana: yellow

	// !!! Query in grafana with !!!
	// {job="somejob"} | logfmt | command="GetWorkersRequest"
	//
	// https://grafana.com/docs/loki/latest/logql/log_queries/#logfmt

	return nil
}

func (l LokiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (l LokiHandler) WithAttrs(attrs []slog.Attr) slog.Handler { //nolint:ireturn // required for slog.Handler
	return &LokiHandler{
		client:   l.client,
		renderer: l.renderer.WithAttrs(attrs),
		output:   l.output,
	}
}

func (l LokiHandler) WithGroup(name string) slog.Handler { //nolint:ireturn // required for slog.Handler
	return &LokiHandler{
		client:   l.client,
		renderer: l.renderer.WithGroup(name),
		output:   l.output,
	}
}

// NewLokiHandler use this handler only for local development!
//
// Its purpose is to mimic your production setting in case you're using loki & grafana.
// It ships your logs to a local loki instance, so you can use the same setup for development.
// It does not care about performance, as in production you would log to `stdout` and the
// container-runtime's drivers (docker, kubernetes) or something will ship your logs to loki.
func NewLokiHandler(opt *LokiHandlerOptions) *LokiHandler {
	defaultOpt := &LokiHandlerOptions{
		PushURL: "http://localhost:3100/api/prom/push",
		Label:   fmt.Sprintf("{%s=\"%s\"}", "arrower", "skeleton"),
	}

	if opt == nil {
		opt = defaultOpt
	}

	if opt.PushURL != "" {
		opt.PushURL = defaultOpt.PushURL
	}

	if opt.Label != "" {
		opt.Label = defaultOpt.Label
	}

	conf := promtail.ClientConfig{
		PushURL:            opt.PushURL,
		BatchWait:          1 * time.Second,
		BatchEntriesNumber: 1,
		SendLevel:          promtail.DEBUG,
		PrintLevel:         promtail.DISABLE,
		Labels:             opt.Label,
	}

	// Do not handle error here, because promtail method always returns `nil`.
	client, _ := promtail.NewClientJson(conf)

	// generate json log by writing to local buffer with slog default json
	buf := &bytes.Buffer{}
	jsonLog := slog.HandlerOptions{
		Level:       LevelDebug, // allow all messages, as the level gets controlled by the ArrowerLogger instead.
		AddSource:   false,
		ReplaceAttr: MapLogLevelsToName,
	}
	renderer := slog.NewJSONHandler(buf, &jsonLog)

	return &LokiHandler{
		client:   client,
		renderer: renderer,
		output:   buf,
	}
}
