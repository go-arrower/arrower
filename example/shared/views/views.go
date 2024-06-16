package views

import (
	"embed"
)

//go:embed **/*.html
var SharedViews embed.FS
