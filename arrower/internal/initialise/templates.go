package initialise

import "embed"

// all includes hidden files preventing rename in application code, see: https://go-review.googlesource.com/c/go/+/359413
//
//go:embed templates/* all:*
var TemplatesFS embed.FS
