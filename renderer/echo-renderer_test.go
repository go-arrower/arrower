package renderer_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
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
		router.GET("/", func(c echo.Context) error { return c.NoContent(http.StatusOK) }).Name = "named-route"

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "", nil)
		r, err := renderer.NewEchoRenderer(slog.New(slog.DiscardHandler), noop.NewTracerProvider(), router, testdata.FilesEcho, nil, true)
		assert.NoError(t, err)

		err = r.Render(&buf, "hello", nil, router.NewContext(req, nil))
		assert.NoError(t, err)
		assert.Contains(t, buf.String(), `<a href="/">Go to named route</a>`)
	})
}
