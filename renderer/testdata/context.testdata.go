package testdata

import "testing/fstest"

const (
	// LDefaultContextContent       = "defaultContextLayout"
	C0ContextContent                = "context component 0"
	P0ContextContent                = "context p0"
	P1ContextContent                = "context p1"
	ContextLayoutPagePlaceholder    = "context layout placeholder"
	ContextLayoutContentPlaceholder = "context content placeholder"
	ContextLayoutContent            = "context other layout content"
)

var ExampleContext = "example"

var SharedViews = fstest.MapFS{ // TODO REMOVE as real shared files are used now
	"components/c0.html":       {Data: []byte(C0Content)},
	"components/c1.html":       {Data: []byte(C1Content)},
	"pages/shared-p0.html":     {Data: []byte(P0Content + ` {{template "c0" .}}`)},
	"pages/shared-p1.html":     {Data: []byte(P1Content)},
	"pages/conflict-page.html": {Data: []byte(P1Content)},
	"default.base.html": {Data: []byte(`<!DOCTYPE html>
<html lang="en">
<body>
	defaultLayout
   {{block "layout" .}}
		defaultContextLayoutOfBase
       {{block "content" .}}
			contentPlaceholder
       {{end}}
   {{end}}
</body>
</html>`)},
	"other.layout.html": {Data: []byte(`otherLayout
   {{block "layout" .}}
       {{block "content" .}}
           contentPlaceholder
       {{end}}
   {{end}}`)},
}

var ContextViews = fstest.MapFS{
	"components/c0.html":       {Data: []byte(C0ContextContent)},
	"pages/p0.html":            {Data: []byte(P0ContextContent + ` {{template "c0" .}}`)},
	"pages/p1.html":            {Data: []byte(P1ContextContent + ` {{block "f" . }}fragment{{end}}`)},
	"pages/conflict-page.html": {Data: []byte("context conflict")},
	"default.layout.html": {Data: []byte(ContextLayoutPagePlaceholder + `
        {{block "content" .}}
			` + ContextLayoutContentPlaceholder + `
        {{end}}`)},
	"other.layout.html": {Data: []byte(ContextLayoutContent + `
        {{block "content" .}}
			` + ContextLayoutContentPlaceholder + `
        {{end}}`)},
}

var ContextAdmin = fstest.MapFS{
	"default.layout.html": {Data: []byte(`
    {{define "layout"}}
		adminLayout
        {{block "content" .}}
			adminPlaceholder
        {{end}}
    {{end}}`)},
}
