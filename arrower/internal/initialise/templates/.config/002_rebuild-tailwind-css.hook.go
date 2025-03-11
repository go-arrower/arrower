package main

import (
	"os"
	"os/exec"
	"strings"

	. "github.com/go-arrower/arrower/arrower/hooks"
)

func init() {
	Register(Hook{
		Name: "Tailwind",
		OnChanged: func(file string) {
			// rebuild tailwind css
			isTailwindRelevantChange := strings.HasSuffix(file, "shared/views/input.css") ||
				strings.HasSuffix(file, ".html")

			if isTailwindRelevantChange {
				cmd := exec.Command("npx", "tailwindcss", "-c", ".config/tailwind.config.js", "-i", "./shared/views/input.css", "-o", "./public/css/main.css")
				cmd.Stderr = os.Stderr
				cmd.Run()
			}
		},
		OnShutdown: func() {
			// build the minified production ready version of the css file
			cmd := exec.Command("npx", "tailwindcss", "-c", ".config/tailwind.config.js", "-i", "./shared/views/input.css", "-o", "./public/css/main.css", "--minify")
			cmd.Stderr = os.Stderr
			cmd.Run()
		},
	})
}
