package internal

import (
	"context"
	"io"
	"strings"
	"sync"

	"github.com/fatih/color" //nolint:misspell
)

// WatchBuildAndRunApp controls the whole `arrower run` cycle and orchestrates it.
func WatchBuildAndRunApp(ctx context.Context, w io.Writer, path string) error {
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

	for {
		select {
		case f := <-filesChanged:
			magenta(w, "changed:", filesRelativePath(f, path))

			checkAndStop(w, stop) // ensures that no two builds are running at the same time

			stop, err = BuildAndRunApp(w, path, "")
			if err != nil {
				red(w, "build & run failed: ", err)
			}
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
