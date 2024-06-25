package web_test

import (
	"io"

	"github.com/labstack/echo/v4"
)

type emptyRenderer struct{} // todo replace with new test renderer

func (t *emptyRenderer) Render(_ io.Writer, _ string, _ interface{}, c echo.Context) error {
	return nil
}

// newTestRouter is a helper for unit tests, by returning a valid web router.
func newTestRouter() *echo.Echo {
	e := echo.New()
	e.Renderer = &emptyRenderer{}

	return e
}
