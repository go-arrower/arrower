package internal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rjeczalik/notify"
)

var ErrObserverFSFailed = errors.New("could not listen to file system changes")

// File is a file name & path. It is expected to be always valid.
type File string

// WatchFolder watches the folder of path and sends a File to fileChanged each time a IsObservedFile changed.
// The debounceInterval is a time in ms in which no new changes will be sent to fileChanged.
// It terminates once the ctx is cancelled.
func WatchFolder(ctx context.Context, path string, fileChanged chan<- File, debounceInterval int) error {
	fsEvents := make(chan notify.EventInfo, 1)

	path += "/..."

	if err := notify.Watch(path, fsEvents, notify.All); err != nil {
		return fmt.Errorf("%w: %v", ErrObserverFSFailed, err)
	}
	defer notify.Stop(fsEvents)

	var debounceActive bool

	for {
		select {
		case <-ctx.Done():
			return nil
		case e := <-fsEvents:
			f := File(e.Path())
			if IsObservedFile(f) {
				if !debounceActive { // not fired within the last interval => event can be fired
					fileChanged <- f

					debounceActive = true
				}
			}
		case <-time.After(time.Duration(debounceInterval) * time.Millisecond):
			debounceActive = false // reset debounce, so events will fire again
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

	if fn.IsFrontendSource() {
		return true
	}

	return false
}

// IsFrontendSource checks if a given File f is a frontend resource.
func (f File) IsFrontendSource() bool {
	if strings.HasSuffix(string(f), ".html") ||
		strings.HasSuffix(string(f), ".css") ||
		strings.HasSuffix(string(f), ".js") ||
		strings.HasSuffix(string(f), ".png") ||
		strings.HasSuffix(string(f), ".jpg") ||
		strings.HasSuffix(string(f), ".jpeg") ||
		strings.HasSuffix(string(f), ".gif") ||
		strings.HasSuffix(string(f), ".ico") {
		return true
	}

	return false
}

// IsCSS checks if the given File f is a css file.
func (f File) IsCSS() bool {
	return strings.HasSuffix(string(f), ".css")
}
