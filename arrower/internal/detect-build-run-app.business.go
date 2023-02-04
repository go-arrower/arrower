package internal

import (
	"context"
	"strings"

	"github.com/fatih/color" //nolint:misspell
)

func WatchBuildAndRunApp(ctx context.Context, path string) error {
	red := color.New(color.FgRed, color.Bold).PrintlnFunc()
	magenta := color.New(color.FgMagenta, color.Bold).PrintlnFunc()

	filesChanged := make(chan File)

	go func() {
		err := WatchFolder(ctx, path, filesChanged)
		if err != nil {
			panic(err)
		}
	}()

	// ! ensure only one process is running at the same time and others are "queued"

	stop, err := BuildAndRunApp(path, "")
	if err != nil {
		red("build & run failed: ", err)
	}

	for {
		select {
		case f := <-filesChanged:
			magenta("changed:", filesRelativePath(f, path))

			checkAndStop(stop)

			stop, err = BuildAndRunApp(path, "")
			if err != nil {
				red("build & run failed: ", err)
			}
		case <-ctx.Done():
			checkAndStop(stop)

			return nil
		}
	}
}

func checkAndStop(stop func() error) {
	red := color.New(color.FgRed, color.Bold).PrintlnFunc()

	if stop != nil {
		err := stop()
		if err != nil {
			red("shutdown failed: ", err)
		}
	}
}

// filesRelativePath assumes File f is an absolute path and turns it into a relative path by removing absolutePrefix.
func filesRelativePath(f File, absolutePrefix string) string {
	return strings.TrimPrefix(string(f), absolutePrefix+"/")
}
