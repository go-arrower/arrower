//nolint:misspell // external library uses "color" (American spelling), not "colour"
package internal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/fatih/color"

	"github.com/go-arrower/arrower/arrower/hooks"
)

var ErrCommandFailed = errors.New("command failed")

// WatchBuildAndRunApp controls the whole `arrower run` cycle and orchestrates it.
func WatchBuildAndRunApp( //nolint:funlen
	ctx context.Context,
	w io.Writer,
	pathInfos PathInfos,
	hooks hooks.Hooks,
	hotReload chan File,
	openBrowser OpenBrowserFunc,
) error {
	red := color.New(color.FgRed, color.Bold).FprintlnFunc()
	magenta := color.New(color.FgMagenta, color.Bold).FprintlnFunc()
	wg := sync.WaitGroup{}

	projectPath := pathInfos[0].AbsPath // use the first path as the main project path for building

	filesChanged := make(chan File)

	wg.Add(1)
	go func(wg *sync.WaitGroup) { //nolint:wsl_v5
		const interval = 350 * time.Millisecond

		err := WatchFolders(ctx, pathInfos, filesChanged, interval)
		if err != nil {
			panic(err)
		}

		wg.Done()
	}(&wg)

	stop, err := BuildAndRunApp(ctx, w, projectPath, "")
	if err != nil {
		red(w, "build & run failed: ", err)
	}

	err = openBrowser(ctx, "localhost:8080")
	if err != nil {
		return fmt.Errorf("could not open app in browser: %w", err)
	}

	for {
		select {
		case file := <-filesChanged:
			relativePath := pathInfos.FormatFilePath(string(file))
			magenta(w, "changed:", relativePath)

			hooks.OnChanged(relativePath)

			if file.IsFrontendSource() {
				hotReload <- file

				continue
			}

			checkAndStop(w, stop) // ensures that no two builds are running at the same time

			stop, err = BuildAndRunApp(ctx, w, projectPath, "")
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

// OpenBrowserFunc is a signature used to open a browser from the CLI.
type OpenBrowserFunc func(ctx context.Context, url string) error

// OpenBrowser starts the systems default browser with the given website.
func OpenBrowser(ctx context.Context, url string) error {
	if url == "" {
		return fmt.Errorf("%w: invalid url", ErrCommandFailed)
	}

	err := exec.CommandContext(ctx, "xdg-open", url).Run()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCommandFailed, err)
	}

	return nil
}

func NoBrowser(_ context.Context, _ string) error { return nil }
