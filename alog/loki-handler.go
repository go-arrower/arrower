package alog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var (
	errSendFailed = errors.New("send to loki failed")
	errLokiFailed = errors.New("loki failed")
)

// NewLokiHandler use this handler only for local development!
//
// Its purpose is to mimic your production setting in case you're using loki & grafana.
// It ships your logs to a local loki instance, so you can use the same setup for development.
// It does not care about performance, as in production you would log to `stdout` and the
// container-runtime's drivers (docker, kubernetes) or something will ship your logs to loki.
func NewLokiHandler(opt *LokiHandlerOptions) *LokiHandler {
	opt = optsFromConfigOrDefault(opt)

	// generate JSON log by writing to local buffer with slog default JSON
	buf := &bytes.Buffer{}
	renderer := slog.NewJSONHandler(buf, &slog.HandlerOptions{
		Level:       LevelDebug, // allow all messages, as the level gets controlled by the arrowerHandler instead.
		AddSource:   false,
		ReplaceAttr: MapLogLevelsToName,
	})

	available := pingLoki(opt)
	handler := &LokiHandler{
		opt:           opt,
		mu:            &sync.Mutex{},
		lokiAvailable: &available,
		renderer:      renderer,
		output:        buf,
	}

	go retryLokiConnection(handler)

	return handler
}

func optsFromConfigOrDefault(opt *LokiHandlerOptions) *LokiHandlerOptions {
	defaultOpt := &LokiHandlerOptions{
		URL: "http://localhost:3100",
		Labels: map[string]string{
			"arrower": "application",
			"client":  "arrower-loki",
		},
	}

	if opt == nil {
		opt = defaultOpt
	}

	if opt.URL == "" {
		opt.URL = defaultOpt.URL
	}

	if len(opt.Labels) == 0 {
		opt.Labels = defaultOpt.Labels
	}

	return opt
}

func retryLokiConnection(handler *LokiHandler) {
	const lokiRetryInterval = 15 * time.Second
	t := time.NewTicker(lokiRetryInterval)

	for range t.C {
		av := pingLoki(handler.opt)

		handler.mu.Lock()
		handler.lokiAvailable = &av
		handler.mu.Unlock()
	}
}

func pingLoki(opt *LokiHandlerOptions) bool {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, opt.URL+"/ready", nil)
	if err != nil {
		return false
	}

	client := &http.Client{
		Transport:     http.DefaultTransport,
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       defaultTimeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}

	defer resp.Body.Close()

	return resp.StatusCode < http.StatusBadRequest
}

type (
	LokiHandlerOptions struct {
		Labels map[string]string
		URL    string
	}

	LokiHandler struct {
		opt           *LokiHandlerOptions
		mu            *sync.Mutex
		lokiAvailable *bool

		renderer slog.Handler
		output   *bytes.Buffer
	}
)

var _ slog.Handler = (*LokiHandler)(nil)

func (l *LokiHandler) Handle(ctx context.Context, record slog.Record) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !*l.lokiAvailable { //  no loki instance is available => do not log.
		return nil
	}

	err := l.renderer.Handle(ctx, record)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	level := record.Level

	record.Attrs(func(a slog.Attr) bool {
		if a.Key == "err" {
			level = slog.LevelError
			return false
		}

		return true
	})

	err = l.sendToLoki(ctx, l.output.String(), level)
	if err != nil {
		return fmt.Errorf("%w: could not transmit logs: %w", errSendFailed, err)
	}

	l.output.Reset()

	return nil
}

func (l *LokiHandler) sendToLoki(ctx context.Context, jsonLog string, level slog.Level) error {
	payload := map[string]interface{}{
		"streams": []map[string]interface{}{
			{
				"stream": l.getLabels(level),
				"values": [][]string{
					{strconv.FormatInt(time.Now().UnixNano(), 10), jsonLog},
				},
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("could not marshal payload: %v", err) //nolint:err113,errorlint // // prevent err in api
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		l.opt.URL+"/loki/api/v1/push",
		bytes.NewBuffer(jsonPayload),
	)
	if err != nil {
		return fmt.Errorf("could not build http request: %v", err) //nolint:err113,errorlint // // prevent err in api
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Transport:     http.DefaultTransport,
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       defaultTimeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("could not post to URL: %v", err) //nolint:err113,errorlint // // prevent err in api
	}

	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body := []byte{}
		_, _ = resp.Body.Read(body)

		return fmt.Errorf("%w: %d: %s", errLokiFailed, resp.StatusCode, string(body))
	}

	return nil
}

func (l *LokiHandler) getLabels(level slog.Level) map[string]string {
	copiedMap := make(map[string]string)
	for key, value := range l.opt.Labels {
		copiedMap[key] = value
	}

	copiedMap["level"] = level.String()

	return copiedMap
}

func (l *LokiHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (l *LokiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LokiHandler{
		opt:           l.opt,
		mu:            l.mu,
		lokiAvailable: l.lokiAvailable,
		renderer:      l.renderer.WithAttrs(attrs),
		output:        l.output,
	}
}

func (l *LokiHandler) WithGroup(name string) slog.Handler {
	return &LokiHandler{
		opt:           l.opt,
		mu:            l.mu,
		lokiAvailable: l.lokiAvailable,
		renderer:      l.renderer.WithGroup(name),
		output:        l.output,
	}
}
