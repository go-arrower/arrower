package testdata

import (
	"bytes"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/labstack/echo/v4"
)

const (
	C0Content                    = "c0"
	C1Content                    = "c1"
	P0Content                    = "p0"
	P1Content                    = "p1"
	P2Content                    = "p2"
	F0Content                    = "f0"
	F1Content                    = "f1"
	BaseLayoutContent            = "base"
	BaseLayoutPagePlaceholder    = "pageLayout placeholder"
	BaseLayoutContentPlaceholder = "content placeholder"
	BaseDefaultLayoutContent     = "defaultBaseLayout"
)

var FilesEmpty = fstest.MapFS{}

func FilesSharedViews() fstest.MapFS {
	return fstest.MapFS{
		"components/c0.html":       {Data: []byte(C0Content)},
		"components/c1.html":       {Data: []byte(C1Content)},
		"pages/p0.html":            {Data: []byte(P0Content)},
		"pages/p1.html":            {Data: []byte(P1Content + ` {{template "c0" .}}`)},
		"pages/p2.html":            {Data: []byte(P2Content + fmt.Sprintf(`{{block "f0" .}}%s{{end}} {{block "f1" .}}%s{{end}}`, F0Content, F1Content))},
		"pages/shared.html":        {Data: []byte(P0Content + ` {{template "c0" .}}`)},
		"pages/conflict-page.html": {Data: []byte(P0Content)},
		"global.base.html": {Data: []byte(BaseLayoutContent + `
    {{block "layout" .}}
		` + BaseLayoutPagePlaceholder + `
        {{block "content" .}}
            ` + BaseLayoutContentPlaceholder + `
        {{end}}
    {{end}}`)},
	}
}

func FilesSharedViewsWithoutBase() fstest.MapFS {
	fs := FilesSharedViews()
	delete(fs, "global.base.html")

	return fs
}

func FilesSharedViewsWithMultiBase() fstest.MapFS {
	fs := FilesSharedViews()

	fs["global.base.html"] = &fstest.MapFile{Data: []byte(`<!DOCTYPE html>
<html lang="en">
<body>
	globalBase
    {{block "layout" .}}
        {{block "content" .}}
            contentPlaceholder
        {{end}}
    {{end}}
</body>
</html>`),
	}
	fs["other.base.html"] = &fstest.MapFile{Data: []byte(`otherBase
	{{block "layout" .}}
	   {{block "content" .}}
	       contentPlaceholder
	   {{end}}
	{{end}}`),
	}

	return fs
}

func FilesSharedViewsWithDefaultBase() fstest.MapFS {
	fs := FilesSharedViews()

	fs["default.base.html"] = &fstest.MapFile{Data: []byte(BaseDefaultLayoutContent +
		`{{block "layout" .}}` + BaseLayoutPagePlaceholder + `
			{{block "content" .}}
            	` + BaseLayoutContentPlaceholder + `
            {{end}}
		{{end}}`)}

	return fs
}

func FilesSharedViewsWithDefaultBaseWithoutLayout() fstest.MapFS {
	fs := FilesSharedViews()

	fs["default.base.html"] = &fstest.MapFile{Data: []byte(BaseDefaultLayoutContent + ` {{block "content" .}}`)}

	return fs
}

func FilesSharedViewsWithCustomFuncs() fstest.MapFS {
	fs := FilesSharedViews()

	fs["components/use-func-map.html"] = &fstest.MapFile{Data: []byte(`{{ customFunc }}`)}
	fs["pages/use-func-map.html"] = &fstest.MapFile{Data: []byte(`{{ hello }} {{ customFunc }}`)}

	return fs
}

func FilesAddBaseData() fstest.MapFS {
	return fstest.MapFS{
		"components/c0.html": {Data: []byte(C0Content)},
		"pages/p0.html":      {Data: []byte(P0Content)},
		"default.base.html": {Data: []byte(BaseLayoutContent + `
	{{ .baseTitle }}
	{{ .BaseHeader }}
	{{ .someType.Name }}

	{{ range .someTypes }}
		<li>{{ .Name }}</li>
	{{ end }}

    {{block "layout" .}}
        {{block "content" .}}
            ` + BaseLayoutContentPlaceholder + `
        {{end}}
    {{end}}`)},
		"other.base.html": {Data: []byte(`otherBase
	{{ .baseTitle }}
	{{ .BaseHeader }}
	{{ .someType.Name }}

	{{ range .someTypes }}
		<li>{{ .Name }}</li>
	{{ end }}

    {{block "layout" .}}
        {{block "content" .}}
            ` + BaseLayoutContentPlaceholder + `
        {{end}}
    {{end}}`)},
	}
}

func FilesAddLayoutData() fstest.MapFS {
	return fstest.MapFS{
		"components/c0.html": {Data: []byte(C0Content)},
		"pages/p0.html":      {Data: []byte(P0Content)},
		"pages/p1.html":      {Data: []byte(P1Content)},
		"default.layout.html": {Data: []byte(`
		{{ .layoutTitle }}
		{{ .LayoutHeader }}
        {{block "content" .}}
            ` + ContextLayoutContentPlaceholder + `
        {{end}}`)},
		"other.layout.html": {Data: []byte(`
		{{ .layoutTitle }}
		{{ .LayoutHeader }}
        {{block "content" .}}
            ` + ContextLayoutContentPlaceholder + `
        {{end}}`)},
	}
}

var SingleNonDefaultLayout = fstest.MapFS{ // TODO remove?
	"pages/p0.page.html": {Data: []byte(P0Content)},
	"global.base.html":   {Data: []byte(BaseLayoutContent)},
}

func NewEchoContext(t *testing.T) echo.Context { // TODO rename to Test instead of new
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	return echo.New().NewContext(req, rec)
}

func NewExampleContextEchoContext(t *testing.T) echo.Context { // TODO rename to Test instead of new
	t.Helper()

	c := NewEchoContext(t)
	c.SetPath(fmt.Sprintf("/%s", ExampleContext))

	return c
}

func GenRandomPages(numPages int) (fstest.MapFS, fstest.MapFS, []string) {
	fsBase := fstest.MapFS{
		"default.base.html": {Data: []byte(BaseDefaultLayoutContent + `
			{{ .BasePlaceholder }}
			{{ block "layout" . }}
	   			{{ block "content" . }}
	       			contentPlaceholder
	   			{{end}}
			{{end}}`,
		)},
	}

	fsContext := fstest.MapFS{
		"default.layout.html": {Data: []byte(`
			{{ .LayoutPlaceholder }}
			{{ block "content" . }}
				contentPlaceholder
			{{end}}`,
		)},
	}

	var pageNames []string

	for i := 0; i < numPages; i++ {
		p := randomString(5)
		fsBase["pages/"+p+".html"] = &fstest.MapFile{Data: []byte(p)} //nolint:exhaustruct

		pageNames = append(pageNames, p)
	}

	return fsBase, fsContext, pageNames
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randomString(n int) string {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec // used for ids, not security

	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rnd.Intn(len(letters))]
	}

	return string(b)
}

// DumpAllNamedTemplatesRenderedWithData pretty prints all templates
// within the given *template.Template. Use it for convenient debugging.
//
//nolint:forbidigo,lll // this is a debug helper, so the use of fmt is the feature.
func DumpAllNamedTemplatesRenderedWithData(templ *template.Template, data interface{}) {
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
