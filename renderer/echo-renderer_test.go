package renderer_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/go-arrower/arrower/renderer"
	"github.com/go-arrower/arrower/renderer/testdata"
)

func TestNewEchoRenderer(t *testing.T) {
	t.Parallel()

	t.Run("new", func(t *testing.T) {
		t.Parallel()

		r, err := renderer.NewEchoRenderer(slog.New(slog.DiscardHandler), noop.NewTracerProvider(), nil, testdata.FilesEmpty, nil, true)
		assert.NoError(t, err)
		assert.NotNil(t, r)
	})

	t.Run("route func is set", func(t *testing.T) {
		t.Parallel()

		buf := bytes.Buffer{}
		router := echo.New()
		router.AddRoute(echo.Route{
			Method:  http.MethodGet,
			Path:    "/",
			Name:    "named-route",
			Handler: func(c *echo.Context) error { return c.NoContent(http.StatusOK) },
		})

		req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "", nil)
		r, err := renderer.NewEchoRenderer(slog.New(slog.DiscardHandler), noop.NewTracerProvider(), router, testdata.FilesEcho, nil, true)
		assert.NoError(t, err)

		context := router.NewContext(req, httptest.NewRecorder())
		err = r.Render(context, &buf, "hello", nil)
		assert.NoError(t, err)
		assert.Contains(t, buf.String(), `<a href="/">Go to named route</a>`)
	})
}
