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
			err := internal.WatchFolder(ctx, dir, filesChanged, 20)
			assert.NoError(t, err)
			wg.Done()
		}()

		time.Sleep(10 * time.Millisecond) // wait until the goroutine has started WatchFolder.
		_, _ = os.Create(fmt.Sprintf("%s/%s", dir, "test0.go"))

		time.Sleep(10 * time.Millisecond) // wait to enforce order of filesChanged elements
		_, _ = os.Create(fmt.Sprintf("%s/%s", dir, "test1.go"))

		time.Sleep(10 * time.Millisecond) // wait for watcher to detect changes
		assert.Equal(t, 1, len(filesChanged))

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
			err := internal.WatchFolder(ctx, dir, filesChanged, 20)
			assert.NoError(t, err)
			wg.Done()
		}()

		time.Sleep(10 * time.Millisecond) // wait until the goroutine has started WatchFolder.
		_, _ = os.Create(fmt.Sprintf("%s/%s", dir, "test0.go"))

		time.Sleep(40 * time.Millisecond) // wait to enforce order of filesChanged elements
		_, _ = os.Create(fmt.Sprintf("%s/%s", dir, "test1.go"))

		time.Sleep(10 * time.Millisecond) // wait for watcher to detect changes
		assert.Equal(t, 2, len(filesChanged))

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
