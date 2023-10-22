package internal_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/arrower/internal"
)

// The file watcher library github.com/rjeczalik/notify does not operate on any fs abstraction, but takes a path string
// as input. This means unit tests utilising abstractions like testing/fstest can not be used, and we have to fall back
// to slower integration tests, operating on the real file system of the OS.
func TestWatchFolder(t *testing.T) {
	t.Parallel()

	const debounceInterval = 20

	t.Run("start and stop watching a folder for changes", func(t *testing.T) {
		t.Parallel()

		wg := sync.WaitGroup{}

		filesChanged := make(chan internal.File)
		ctx, cancel := context.WithCancel(context.Background())

		wg.Add(1)
		go func() {
			err := internal.WatchFolder(ctx, ".", filesChanged, 300)
			assert.NoError(t, err)
			wg.Done()
		}()

		cancel()
		wg.Wait()
	})

	t.Run("watch specific folder", func(t *testing.T) {
		t.Parallel()

		t.Skip()
		/*
			cases
				* valid folder
				* invalid folder
				* relative path
				* absolute path
		*/

		wg := sync.WaitGroup{}

		filesChanged := make(chan internal.File)
		ctx, cancel := context.WithCancel(context.Background())

		dir := t.TempDir()

		wg.Add(1)
		go func() {
			err := internal.WatchFolder(ctx, dir, filesChanged, 300)
			assert.NoError(t, err)
			wg.Done()
		}()

		cancel()
		wg.Wait()
	})

	t.Run("add a file", func(t *testing.T) {
		t.Parallel()

		wg := sync.WaitGroup{}

		filesChanged := make(chan internal.File, 1)
		ctx, cancel := context.WithCancel(context.Background())

		dir := t.TempDir()

		wg.Add(1)
		go func() {
			err := internal.WatchFolder(ctx, dir, filesChanged, 300)
			assert.NoError(t, err)
			wg.Done()
		}()

		time.Sleep(10 * time.Millisecond) // wait until the goroutine has started WatchFolder.
		_, _ = os.Create(fmt.Sprintf("%s/%s", dir, "test.go"))

		file := <-filesChanged // expect to pull one File from the channel
		assert.Contains(t, file, "test.go")

		cancel()
		wg.Wait()
	})

	t.Run("debounce changes", func(t *testing.T) { //nolint:dupl // triggers the debounce
		t.Parallel()

		wg := sync.WaitGroup{}

		filesChanged := make(chan internal.File, 10)
		ctx, cancel := context.WithCancel(context.Background())

		dir := t.TempDir()

		wg.Add(1)
		go func() {
			err := internal.WatchFolder(ctx, dir, filesChanged, debounceInterval)
			assert.NoError(t, err)
			wg.Done()
		}()

		time.Sleep(100 * time.Millisecond) // wait until the goroutine has started WatchFolder.
		_, _ = os.Create(fmt.Sprintf("%s/%s", dir, "test0.go"))

		time.Sleep(debounceInterval / 2 * time.Millisecond) // wait to enforce order of filesChanged elements
		_, _ = os.Create(fmt.Sprintf("%s/%s", dir, "test1.go"))

		const expected = 1
		wait(filesChanged, expected)
		assert.Len(t, filesChanged, expected)

		cancel()
		wg.Wait()
	})

	t.Run("debounce changes until interval", func(t *testing.T) { //nolint:dupl // doesn't trigger debounce
		t.Parallel()

		wg := sync.WaitGroup{}

		filesChanged := make(chan internal.File, 10)
		ctx, cancel := context.WithCancel(context.Background())

		dir := t.TempDir()

		wg.Add(1)
		go func() {
			err := internal.WatchFolder(ctx, dir, filesChanged, debounceInterval)
			assert.NoError(t, err)
			wg.Done()
		}()

		time.Sleep(10 * time.Millisecond) // wait until the goroutine has started WatchFolder.
		_, _ = os.Create(fmt.Sprintf("%s/%s", dir, "test0.go"))

		time.Sleep(debounceInterval * 2 * time.Millisecond) // wait to enforce order of filesChanged elements
		_, _ = os.Create(fmt.Sprintf("%s/%s", dir, "test1.go"))

		const expected = 2
		wait(filesChanged, expected)
		assert.Len(t, filesChanged, expected)

		cancel()
		wg.Wait()
	})

	t.Run("change files that are ignored", func(t *testing.T) {
		t.Parallel()

		wg := sync.WaitGroup{}

		filesChanged := make(chan internal.File, 1)
		ctx, cancel := context.WithCancel(context.Background())

		dir := t.TempDir()

		wg.Add(1)
		go func() {
			err := internal.WatchFolder(ctx, dir, filesChanged, 300)
			assert.NoError(t, err)
			wg.Done()
		}()

		time.Sleep(10 * time.Millisecond) // wait until the goroutine has started WatchFolder.
		_, _ = os.Create(fmt.Sprintf("%s/%s", dir, "something_test.go"))

		select {
		case <-filesChanged:
			t.FailNow()
		default: // No value ready, moving on
		}

		cancel()
		wg.Wait()
	})
}

func TestIsObservedFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName string
		file     internal.File
		observed bool
	}{
		{
			"empty",
			"",
			false,
		},
		{
			"app binary",
			"app",
			false,
		},
		{
			"go",
			"main.go",
			true,
		},
		{
			"config",
			"arrower.config",
			true,
		},
		{
			"config",
			"arrower.conf",
			true,
		},
		{
			"yaml",
			"arrower.yaml",
			true,
		},
		{
			"test files",
			"main_test.go",
			false,
		},
		{
			"test.go",
			"test.go",
			true,
		},
		{
			"some.html",
			"some.html",
			true,
		},
		{
			"some.css",
			"some.css",
			true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			got := internal.IsObservedFile(tt.file)
			assert.Equal(t, tt.observed, got)
		})
	}
}

func TestFile_IsCSS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName string
		file     internal.File
		observed bool
	}{
		{
			"empty",
			"",
			false,
		},
		{
			"some file",
			"some.html",
			false,
		},
		{
			"css file",
			"some.css",
			true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			got := tt.file.IsCSS()
			assert.Equal(t, tt.observed, got)
		})
	}
}

func TestFile_IsFrontendSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName string
		file     internal.File
		observed bool
	}{
		{
			"empty",
			"",
			false,
		},
		{
			"some file",
			"some.html",
			true,
		},
		{
			"css file",
			"some.css",
			true,
		},
		{
			"js file",
			"some.js",
			true,
		},
		{
			"png file",
			"some.png",
			true,
		},
		{
			"jpg file",
			"some.jpg",
			true,
		},
		{
			"jpeg file",
			"some.jpeg",
			true,
		},
		{
			"gif file",
			"some.gif",
			true,
		},
		{
			"favicon",
			"some.ico",
			true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			got := tt.file.IsFrontendSource()
			assert.Equal(t, tt.observed, got)
		})
	}
}

// wait is a helper to pass time, until the watcher detects changes.
func wait(filesChanged chan internal.File, expected int) {
	for {
		if len(filesChanged) == expected {
			time.Sleep(10 * time.Millisecond)

			break
		}
	}
}
