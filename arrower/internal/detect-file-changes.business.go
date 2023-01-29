package internal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rjeczalik/notify"
)

var ErrObserverFSFailed = errors.New("could not listen to file system changes")

// File is a file name & path. It is expected to be always valid.
type File string

// WatchFolder watches the folder of path and sends a File to fileChanged each time a IsObservedFile changed.
// It terminates once the ctx is cancelled.
func WatchFolder(ctx context.Context, path string, fileChanged chan<- File) error {
	fsEvents := make(chan notify.EventInfo, 1)

	path = fmt.Sprintf("%s/...", path)

	if err := notify.Watch(path, fsEvents, notify.All); err != nil {
		return fmt.Errorf("%w: %v", ErrObserverFSFailed, err)
	}
	defer notify.Stop(fsEvents)

	for {
		select {
		case <-ctx.Done():
			return nil
		case e := <-fsEvents:
			f := File(e.Path())
			if IsObservedFile(f) {
				fileChanged <- f
			}
		}
	}
}

// IsObservedFile returns weather a File is being observed or not.
func IsObservedFile(fn File) bool {
	if strings.HasSuffix(string(fn), ".go") && !strings.HasSuffix(string(fn), "_test.go") {
		return true
	}

	if strings.HasSuffix(string(fn), ".config") || strings.HasSuffix(string(fn), ".conf") {
		return true
	}

	if strings.HasSuffix(string(fn), ".yaml") {
		return true
	}

	return false
}
