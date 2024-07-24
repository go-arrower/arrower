package alog

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/afiskon/promtail-client/promtail"
)

// NewLokiHandler use this handler only for local development!
//
// Its purpose is to mimic your production setting in case you're using loki & grafana.
// It ships your logs to a local loki instance, so you can use the same setup for development.
// It does not care about performance, as in production you would log to `stdout` and the
// container-runtime's drivers (docker, kubernetes) or something will ship your logs to loki.
func NewLokiHandler(opt *LokiHandlerOptions) *LokiHandler {
	conf := getPromtailConfig(opt)
	client := getClient(conf)

	// generate json log by writing to local buffer with slog default json
	buf := &bytes.Buffer{}
	renderer := slog.NewJSONHandler(buf, &slog.HandlerOptions{
		Level:       LevelDebug, // allow all messages, as the level gets controlled by the arrowerHandler instead.
		AddSource:   false,
		ReplaceAttr: MapLogLevelsToName,
	})

	handler := &LokiHandler{
		mu:       sync.Mutex{},
		client:   client,
		renderer: renderer,
		output:   buf,
	}

	if client == nil {
		go retryLokiConnection(handler, conf)
	}

	return handler
}

func getPromtailConfig(opt *LokiHandlerOptions) promtail.ClientConfig {
	defaultOpt := &LokiHandlerOptions{
		PushURL: "http://localhost:3100/api/prom/push",
		Labels: map[string]string{
			"arrower": "application",
			"client":  "arrower-loki",
		},
	}

	if opt == nil {
		opt = defaultOpt
	}

	if opt.PushURL == "" {
		opt.PushURL = defaultOpt.PushURL
	}

	if len(opt.Labels) == 0 {
		opt.Labels = defaultOpt.Labels
	}

	label := "{"
	for k, l := range opt.Labels {
		label += fmt.Sprintf("%s=\"%s\",", k, l)
	}
	label += "}" //nolint:wsl

	return promtail.ClientConfig{
		PushURL:            opt.PushURL,
		BatchWait:          1 * time.Second,
		BatchEntriesNumber: 1,
		SendLevel:          promtail.DEBUG,
		PrintLevel:         promtail.DISABLE,
		Labels:             label,
	}
}

func retryLokiConnection(handler *LokiHandler, conf promtail.ClientConfig) {
	const lokiRetryInterval = 15 * time.Second
	t := time.NewTicker(lokiRetryInterval)

	for range t.C {
		client := getClient(conf)
		if client == nil {
			continue
		}

		handler.mu.Lock()
		defer handler.mu.Unlock()

		handler.client = client

		return
	}
}

func getClient(conf promtail.ClientConfig) promtail.Client { //nolint:ireturn,lll // promtail.NewClientX() only returns interface.
	cli := &http.Client{} //nolint:exhaustruct

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, conf.PushURL, nil)
	if err != nil {
		slog.Info("could not get request", slog.Any("err", err), slog.String("url", conf.PushURL))
		return nil
	}

	res, err := cli.Do(req)
	if err != nil {
		slog.Info("could not ping loki", slog.Any("err", err), slog.String("url", conf.PushURL))
		return nil
	}

	_ = res.Body.Close()

	client, _ := promtail.NewClientJson(conf) // Do not handle error here, because promtail always returns `nil`.

	return client
}

type (
	LokiHandlerOptions struct {
		Labels  map[string]string
		PushURL string
	}

	LokiHandler struct { //nolint:govet // fieldalignment not as important as readability.
		mu     sync.Mutex
		client promtail.Client

		renderer slog.Handler
		output   *bytes.Buffer
	}
)

var _ slog.Handler = (*LokiHandler)(nil)

func (l *LokiHandler) Handle(ctx context.Context, record slog.Record) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.client == nil { // client is empty if no loki instance is available => do not log.
		return nil
	}

	err := l.renderer.Handle(ctx, record)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

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

func (l *LokiHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (l *LokiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LokiHandler{
		mu:       sync.Mutex{},
		client:   l.client,
		renderer: l.renderer.WithAttrs(attrs),
		output:   l.output,
	}
}

func (l *LokiHandler) WithGroup(name string) slog.Handler {
	return &LokiHandler{
		mu:       sync.Mutex{},
		client:   l.client,
		renderer: l.renderer.WithGroup(name),
		output:   l.output,
	}
}
