package views

import "embed"

//go:embed *.html **/*.html
var AdminViews embed.FS

//go:embed static/*
var PublicAssets embed.FS
