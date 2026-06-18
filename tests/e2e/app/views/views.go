package views

import "embed"

// SharedViews holds the shared base layouts the context pages render into. The
// renderer discovers layouts as root-level *.base.html files (baseName strips the
// .base.html suffix → layout name, renderer/renderer.go:410):
//   - default.base.html → "default"  (fallback; used by admin, which renders 1-segment names)
//   - auth.base.html    → "auth"     (required: auth renders "auth=>=>login", i.e. baseLayout "auth")
//
// See README.md for the full template-name → layout contract.
//
//go:embed *.html
var SharedViews embed.FS
