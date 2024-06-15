// Use white box testing, to make it easier to assert on the inner workings of partially loaded and cached templates.
// If a white box test case fails, consider just deleting it over fixing it to prevent coupling to the implementation.
//
//nolint:testpackage
package renderer

import (
	"html/template"
	"testing"

	"github.com/go-arrower/arrower/alog"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/go-arrower/arrower/renderer/testdata"
)

func TestNewRenderer(t *testing.T) {
	t.Parallel()

	// todo could be improved by
	// assert the page has itself and all dependencies loaded as a template
	// assert template is cached

	// white box test. if it fails, feel free to delete it
	t.Run("initialise new", func(t *testing.T) {
		t.Parallel()

		renderer, err := New(alog.NewTest(nil), noop.NewTracerProvider(), testdata.FilesSharedViews(), template.FuncMap{}, false)
		assert.NoError(t, err)
		assert.NotNil(t, renderer)

		assert.Equal(t, 0, renderer.totalCachedTemplates())

		// assert component templates from testdata.FilesSharedViews is loaded
		// contains always an empty component, so expected 1 + 2 => 3
		assert.Len(t, renderer.viewsForContext("").components.Templates(), 3)

		// assert pages are found and extracted
		assert.Len(t, renderer.viewsForContext("").rawPages, 5)
		// if the file is called p0.html, the template is called p0
		assert.NotEmpty(t, renderer.viewsForContext("").rawPages["p0"])
		assert.NotEmpty(t, renderer.viewsForContext("").rawPages["p1"])
		assert.NotEmpty(t, renderer.viewsForContext("").rawPages["p2"])
		assert.Empty(t, renderer.viewsForContext("").rawPages["non-existent"])
	})

	// white box test. if it fails, feel free to delete it
	t.Run("fs with no files", func(t *testing.T) {
		t.Parallel()

		renderer, err := New(alog.NewTest(nil), noop.NewTracerProvider(), testdata.FilesEmpty, template.FuncMap{}, false)
		assert.NoError(t, err)
		assert.NotNil(t, renderer)

		assert.Len(t, renderer.viewsForContext("").components.Templates(), 1)
		assert.Equal(t, 0, renderer.totalCachedTemplates())
	})
}

// white box test. if it fails, feel free to delete it.
func TestParsedTemplate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		template string
		parsed   parsedTemplate
		err      error
	}{
		"empty":                    {"", parsedTemplate{}, nil},
		"component":                {"#c", parsedTemplate{fragment: "c", isComponent: true}, nil},
		"page":                     {"p", parsedTemplate{page: "p"}, nil},
		"page with fragment":       {"p#f", parsedTemplate{page: "p", fragment: "f"}, nil},
		"page with empty fragment": {"p#", parsedTemplate{}, ErrRenderFailed},
		"context layout":           {"cl=>p", parsedTemplate{contextLayout: "cl", page: "p"}, nil},
		"full layout":              {"gl =>cl=> p", parsedTemplate{baseLayout: "gl", contextLayout: "cl", page: "p"}, nil}, // TODO rename shortcuts from global layout to base layout
		"complete template name":   {"gl=>cl=>p #f ", parsedTemplate{baseLayout: "gl", contextLayout: "cl", page: "p", fragment: "f"}, nil},
		"too many separators":      {"=>=>=>", parsedTemplate{}, ErrRenderFailed},
		"too many fragments":       {"p#p#", parsedTemplate{}, ErrRenderFailed},
		"separator after fragment": {"gl=>cl=>p#f=>", parsedTemplate{}, ErrRenderFailed},
		"fragment in layouts":      {"gl#=>cl=>p#f", parsedTemplate{}, ErrRenderFailed}, // todo have own error if the template name is wrong?
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			template, err := parseTemplateName(tt.template)
			assert.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.parsed, template)
		})
	}
}

// white box test. if it fails, feel free to delete it.
func TestRenderer_Layout(t *testing.T) {
	t.Parallel()

	t.Run("no default layout present", func(t *testing.T) {
		t.Parallel()

		renderer, err := New(alog.NewTest(nil), noop.NewTracerProvider(), testdata.FilesEmpty, template.FuncMap{}, false)
		assert.NoError(t, err)
		assert.NotNil(t, renderer)

		assert.Equal(t, "", renderer.layout())
	})

	t.Run("layout ex, but not the default one", func(t *testing.T) {
		t.Parallel()

		renderer, err := New(alog.NewTest(nil), noop.NewTracerProvider(), testdata.SingleNonDefaultLayout, template.FuncMap{}, false)
		assert.NoError(t, err)
		assert.NotNil(t, renderer)

		assert.Equal(t, "", renderer.layout())
	})

	t.Run("multiple layouts but with default", func(t *testing.T) {
		t.Parallel()

		renderer, err := New(alog.NewTest(nil), noop.NewTracerProvider(), testdata.FilesSharedViewsWithDefaultBase(), template.FuncMap{}, false)
		assert.NoError(t, err)
		assert.NotNil(t, renderer)

		assert.Equal(t, "default", renderer.layout())
	})
}
