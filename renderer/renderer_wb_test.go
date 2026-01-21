// Use white box testing, to make it easier to assert on the inner workings of partially loaded and cached templates.
// If a white box test case fails, consider just deleting it over fixing it to prevent coupling to the implementation.
package renderer

import (
	"html/template"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/go-arrower/arrower/renderer/testdata"
)

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

		renderer, err := New(slog.New(slog.DiscardHandler), noop.NewTracerProvider(), testdata.FilesEmpty, template.FuncMap{}, false)
		assert.NoError(t, err)
		assert.NotNil(t, renderer)

		assert.Empty(t, renderer.layout())
	})

	t.Run("layout ex, but not the default one", func(t *testing.T) {
		t.Parallel()

		renderer, err := New(slog.New(slog.DiscardHandler), noop.NewTracerProvider(), testdata.SingleNonDefaultLayout, template.FuncMap{}, false)
		assert.NoError(t, err)
		assert.NotNil(t, renderer)

		assert.Empty(t, renderer.layout())
	})

	t.Run("multiple layouts but with default", func(t *testing.T) {
		t.Parallel()

		renderer, err := New(slog.New(slog.DiscardHandler), noop.NewTracerProvider(), testdata.FilesSharedViewsWithDefaultBase(), template.FuncMap{}, false)
		assert.NoError(t, err)
		assert.NotNil(t, renderer)

		assert.Equal(t, "default", renderer.layout())
	})
}
