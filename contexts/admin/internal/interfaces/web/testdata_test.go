package web_test

import (
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/go-arrower/arrower/contexts/admin/internal/views"
	"github.com/go-arrower/arrower/renderer"
)

// newTestRouter is a helper for unit tests, by returning a valid web router.
func newTestRouter(t *testing.T) *echo.Echo {
	t.Helper()

	e := echo.New()
	e.Renderer = renderer.TestEcho(t, e, views.AdminViews, nil)

	return e
}
