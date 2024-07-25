package testdata

import (
	"testing/fstest"
)

var FilesEcho = fstest.MapFS{
	"default.base.html": &fstest.MapFile{Data: []byte(`{{block "layout" .}}{{block "content" .}}{{end}}{{end}}`)},
	"pages/hello.html":  &fstest.MapFile{Data: []byte(`<a href="{{ route "named-route" }}">Go to named route</a>`)},
}
