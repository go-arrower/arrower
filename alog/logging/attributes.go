// Package logging is a dedicated package for logging attributes.
//
// It gives a broad view of what attributes are available,
// useful when setting up indexes and queries in dashboards.
// It enforces a consistent way of naming logging attributes,
// and prevents traceID, trace_id, TraceID in the logs at the same time.
package logging

import (
	"log/slog"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"
)

func Error(err error) slog.Attr {
	return slog.String("error", err.Error())
}

func Trace(id trace.TraceID) slog.Attr {
	return slog.String("trace_id", id.String())
}

func Span(id trace.SpanID) slog.Attr {
	return slog.String("span_id", id.String())
}

func Context(context string) slog.Attr {
	return slog.String("context", context)
}

func Command(command string) slog.Attr {
	return slog.String("command", command)
}

// Attr is an exception to the general pattern,
// Use it sparingly and only for quick debugging
// where you don't want to introduce a dedicated function.
func Attr(key, value string) slog.Attr {
	return slog.String(key, value)
}

func Organisation(organisation string) slog.Attr {
	return slog.String("organisation_name", organisation)
}

func Application(application string) slog.Attr {
	return slog.String("application_name", application)
}

func Instance(instance string) slog.Attr {
	return slog.String("instance_name", instance)
}

func GitHash(hash string) slog.Attr {
	return slog.String("git_hash", hash)
}

func Environment(env string) slog.Attr {
	return slog.String("environment", env)
}

func Port(port int) slog.Attr {
	return slog.String("port", strconv.Itoa(port))
}

func Addr(addr string) slog.Attr {
	return slog.String("addr", addr)
}

func MetricPath(path string) slog.Attr {
	return slog.String("metric_path", path)
}

func StatusPath(path string) slog.Attr {
	return slog.String("status_path", path)
}

func Timeout(d time.Duration) slog.Attr {
	return slog.String("timeout", d.String())
}

func RequestID(id string) slog.Attr {
	return slog.String("request_id", id)
}

//
// Jobs
//

func ID(id string) slog.Attr {
	return slog.String("id", id)
}

func Queue(queue string) slog.Attr {
	return slog.String("queue", queue)
}

func Type(t string) slog.Attr {
	return slog.String("type", t)
}

func Args(args string) slog.Attr {
	return slog.String("args", args)
}

func RunCount(count int) slog.Attr {
	return slog.Int("run_count", count)
}

func RunError(err string) slog.Attr {
	return slog.String("run_error", err)
}

func Priority(priority int) slog.Attr {
	return slog.Int("priority", priority)
}

func RunAt(t time.Time) slog.Attr {
	return slog.Time("run_at", t)
}

func PoolName(name string) slog.Attr {
	return slog.String("pool_name", name)
}

//
// Renderer
//

func HotReload(hotReload bool) slog.Attr {
	return slog.Bool("hot_reload", hotReload)
}

func DefaultLayout(layout string) slog.Attr {
	return slog.String("default_layout", layout)
}

func OriginalName(name string) slog.Attr {
	return slog.String("original_name", name)
}

func CacheKey(key string) slog.Attr {
	return slog.String("cache_key", key)
}

func Templates(templates ...string) slog.Attr {
	return slog.String("templates", strings.Join(templates, ","))
}

func ComponentCount(count int) slog.Attr {
	return slog.String("component_count", strconv.Itoa(count))
}

func ComponentTemplates(templates ...string) slog.Attr {
	return slog.String("component_templates", strings.Join(templates, ","))
}

func PageCount(count int) slog.Attr {
	return slog.String("page_count", strconv.Itoa(count))
}

func PageTemplates(templates ...string) slog.Attr {
	return slog.String("page_templates", strings.Join(templates, ","))
}

func LayoutCount(count int) slog.Attr {
	return slog.String("layout_count", strconv.Itoa(count))
}

func LayoutTemplates(templates ...string) slog.Attr {
	return slog.String("layout_templates", strings.Join(templates, ","))
}

//
// Settings
//

func ValueStr(v string) slog.Attr {
	return slog.String("value", v)
}

func ValueBool(v bool) slog.Attr {
	return slog.Bool("value", v)
}

func Object(object any) slog.Attr {
	return slog.Any("object", object)
}
