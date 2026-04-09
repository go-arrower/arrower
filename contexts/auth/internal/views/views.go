package views

import "embed"

//go:embed *.html **/*.html
var AuthViews embed.FS

//go:embed static/*
var PublicAssets embed.FS
