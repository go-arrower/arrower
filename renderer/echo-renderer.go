package renderer

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"strings"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-arrower/arrower/alog"
)

func NewEchoRenderer(
	logger alog.Logger,
	traceProvider trace.TracerProvider,
	echo *echo.Echo,
	viewFS fs.FS,
	hotReload bool,
) (*EchoRenderer, error) {
	renderer, err := New(logger, traceProvider, viewFS, template.FuncMap{
		"route": echo.Reverse,
	}, hotReload)
	if err != nil {
		return nil, fmt.Errorf("could not create echo renderer: %w", err)
	}

	return &EchoRenderer{Renderer: renderer}, nil
}

// EchoRenderer is a wrapper that makes the Renderer available for the echo router: https://echo.labstack.com/
type EchoRenderer struct {
	*Renderer
}

var _ echo.Renderer = (*EchoRenderer)(nil)

func (r *EchoRenderer) Render(w io.Writer, templateName string, data interface{}, c echo.Context) error {
	_, _, context := r.isRegisteredContext(c) // todo test how it is split

	return r.Renderer.Render(c.Request().Context(), w, context, templateName, data)
}

// isRegisteredContext returns if a call is to be rendered for a context registered via AddContext.
// If false => it is a shared view. // TODO refactor.
func (r *EchoRenderer) isRegisteredContext(c echo.Context) (bool, bool, string) {
	paths := strings.Split(c.Path(), "/")

	isAdmin := false

	for _, path := range paths {
		if path == "" {
			continue
		}

		if path == "admin" {
			isAdmin = true

			continue
		}

		_, exists := r.views[path]
		if exists {
			if isAdmin {
				return true, true, "/admin/" + path
			}

			return true, false, path
		}
	}

	if isAdmin {
		return true, true, "admin" // todo return normal context name, as the flag isAdmin is returned already
	}

	return false, false, ""
}

// --- --- ---
//
// Helpers used for white box testing.
// Hopefully, these functions make it harder to break the (partially useful) tests
// if larger refactoring is done on the Renderer's structure.
// Feel free to delete them anytime! Don't feel forced to test implementation detail!
//
// --- --- ---

// layout returns the default layout of this renderer.
func (r *Renderer) layout() string {
	return r.views[SharedViews].defaultLayout
}

func (r *Renderer) viewsForContext(name string) viewTemplates {
	return r.views[name]
}

func (r *Renderer) totalCachedTemplates() int {
	count := 0

	r.cache.Range(func(_, _ any) bool {
		count++

		return true
	})

	return count
}

// rawTemplateNames takes the names of the templates from the keys of the map.
func rawTemplateNames(pages map[string]string) []string {
	var names []string

	for k := range pages {
		names = append(names, k)
	}

	return names
}
