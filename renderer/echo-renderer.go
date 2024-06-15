package renderer

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"strings"

	"github.com/go-arrower/arrower/alog"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/trace"
)

func NewEchoRenderer(
	logger alog.Logger,
	traceProvider trace.TracerProvider,
	echo *echo.Echo,
	viewFS fs.FS,
	hotReload bool,
) (*EchoRenderer, error) {
	r, err := New(logger, traceProvider, viewFS, template.FuncMap{
		"route": echo.Reverse, // todo test case for reverse func
	}, hotReload)
	if err != nil {
		return &EchoRenderer{}, fmt.Errorf("%w", err)
	}

	return &EchoRenderer{Renderer: r}, nil
}

// EchoRenderer is a wrapper that makes the Renderer available for the echo router: https://echo.labstack.com/
type EchoRenderer struct {
	*Renderer
}

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

// dumpAllNamedTemplatesRenderedWithData pretty prints all templates
// within the given *template.Template. Use it for convenient debugging.
// todo move to _test file
//
//nolint:forbidigo,lll // this is a debug helper, so the use of fmt is the feature.
func dumpAllNamedTemplatesRenderedWithData(templ *template.Template, data interface{}) {
	templ, err := templ.Clone() // ones ExecuteTemplate is called the template cannot be pared any more and could fail calling code.
	if err != nil {
		fmt.Println("CAN NOT DUMP TEMPLATE: ", err)

		return
	}

	fmt.Println()
	fmt.Println("--- --- ---   --- --- ---   --- --- ---")
	fmt.Println("--- --- ---   Render all templates:", strings.TrimPrefix(templ.DefinedTemplates(), "; defined templates are: "))
	fmt.Println("--- --- ---   --- --- ---   --- --- ---")

	buf := &bytes.Buffer{}

	for _, t := range templ.Templates() {
		fmt.Printf("--- --- ---   %s:\n", t.Name())

		_ = templ.ExecuteTemplate(buf, t.Name(), data)

		fmt.Println(buf.String())
		buf.Reset()
	}

	fmt.Println("--- --- ---   --- --- ---   --- --- ---")
	fmt.Println()
}
