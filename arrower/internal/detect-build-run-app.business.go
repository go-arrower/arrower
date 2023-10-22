package internal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/fatih/color" //nolint:misspell
)

var ErrCommandFailed = errors.New("command failed")

// WatchBuildAndRunApp controls the whole `arrower run` cycle and orchestrates it.
func WatchBuildAndRunApp(
	ctx context.Context,
	w io.Writer,
	path string,
	hotReload chan File,
	openBrowser OpenBrowserFunc,
) error {
	red := color.New(color.FgRed, color.Bold).FprintlnFunc()
	magenta := color.New(color.FgMagenta, color.Bold).FprintlnFunc()
	wg := sync.WaitGroup{}

	filesChanged := make(chan File)

	wg.Add(1)
	go func(wg *sync.WaitGroup) { //nolint:wsl
		const interval = 350

		err := WatchFolder(ctx, path, filesChanged, interval)
		if err != nil {
			panic(err)
		}

		wg.Done()
	}(&wg)

	stop, err := BuildAndRunApp(w, path, "")
	if err != nil {
		red(w, "build & run failed: ", err)
	}

	err = openBrowser("localhost:8080")
	if err != nil {
		return fmt.Errorf("could not open app in browser: %w", err)
	}

	for {
		select {
		case file := <-filesChanged:
			magenta(w, "changed:", filesRelativePath(file, path))

			if file.IsFrontendSource() {
				hotReload <- file

				continue
			}

			checkAndStop(w, stop) // ensures that no two builds are running at the same time

			stop, err = BuildAndRunApp(w, path, "")
			if err != nil {
				red(w, "build & run failed: ", err)
			}

			hotReload <- file // reload browser tabs
		case <-ctx.Done():
			checkAndStop(w, stop)
			wg.Wait()

			return nil
		}
	}
}

func checkAndStop(w io.Writer, stop func() error) {
	red := color.New(color.FgRed, color.Bold).FprintlnFunc()

	if stop != nil {
		err := stop()
		if err != nil {
			red(w, "shutdown failed: ", err)
		}
	}
}

// filesRelativePath assumes File f is an absolute path and turns it into a relative path by removing absolutePrefix.
func filesRelativePath(f File, absolutePrefix string) string {
	return strings.TrimPrefix(string(f), absolutePrefix+"/")
}

// OpenBrowserFunc is a signature used to open a browser from the CLI.
type OpenBrowserFunc func(url string) error

// OpenBrowser starts the systems default browser with the given website.
func OpenBrowser(url string) error {
	if url == "" {
		return fmt.Errorf("%w: invalid url", ErrCommandFailed)
	}

	err := exec.Command("xdg-open", url).Run()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCommandFailed, err) //nolint:errorlint // prevent err in api
	}

	return nil
}
