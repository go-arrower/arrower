package views

import (
	"embed"
)

//go:embed *.html **/*.html
var Views embed.FS
