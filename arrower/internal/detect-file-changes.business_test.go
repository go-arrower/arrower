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
func TestWatchFolders(t *testing.T) {
	t.Parallel()

	const debounceInterval = 200 * time.Millisecond

	t.Run("start and stop watching a folder for changes", func(t *testing.T) {
		t.Parallel()

		wg := sync.WaitGroup{}

		filesChanged := make(chan internal.File)
		ctx, cancel := context.WithCancel(t.Context())

		wg.Add(1)
		go func(wg *sync.WaitGroup) { //nolint:wsl_v5
			pathInfos, err := internal.NewPathInfos([]string{"."})
			assert.NoError(t, err)
			err = internal.WatchFolders(ctx, pathInfos, filesChanged, debounceInterval)
			assert.NoError(t, err)
			wg.Done()
		}(&wg)

		cancel()
		wg.Wait()
	})

	t.Run("watch specific folder", func(t *testing.T) {
		t.Parallel()

		wg := sync.WaitGroup{}

		filesChanged := make(chan internal.File)
		ctx, cancel := context.WithCancel(t.Context())

		dir := t.TempDir()

		wg.Add(1)
		go func(wg *sync.WaitGroup) { //nolint:wsl_v5
			pathInfos, err := internal.NewPathInfos([]string{dir})
			assert.NoError(t, err)
			err = internal.WatchFolders(ctx, pathInfos, filesChanged, debounceInterval)
			assert.NoError(t, err)
			wg.Done()
		}(&wg)

		cancel()
		wg.Wait()
	})

	t.Run("add a file", func(t *testing.T) {
		t.Parallel()

		wg := sync.WaitGroup{}

		filesChanged := make(chan internal.File, 1)
		ctx, cancel := context.WithCancel(t.Context())

		dir := t.TempDir()

		wg.Add(1)
		go func(wg *sync.WaitGroup) { //nolint:wsl_v5
			pathInfos, err := internal.NewPathInfos([]string{dir})
			assert.NoError(t, err)
			err = internal.WatchFolders(ctx, pathInfos, filesChanged, debounceInterval)
			assert.NoError(t, err)
			wg.Done()
		}(&wg)

		time.Sleep(10 * time.Millisecond) // wait until the goroutine has started WatchFolders.

		_, _ = os.Create(fmt.Sprintf("%s/%s", dir, "test.go"))

		file := <-filesChanged // expect to pull one File from the channel
		assert.Contains(t, file, "test.go")

		cancel()
		wg.Wait()
	})

	t.Run("debounce changes", func(t *testing.T) {
		t.Parallel()

		wg := sync.WaitGroup{}

		filesChanged := make(chan internal.File, 10)
		ctx, cancel := context.WithCancel(t.Context())

		dir := t.TempDir()

		wg.Add(1)
		go func(wg *sync.WaitGroup) { //nolint:wsl_v5
			pathInfos, err := internal.NewPathInfos([]string{dir})
			assert.NoError(t, err)
			err = internal.WatchFolders(ctx, pathInfos, filesChanged, debounceInterval)
			assert.NoError(t, err)
			wg.Done()
		}(&wg)

		time.Sleep(100 * time.Millisecond) // wait until the goroutine has started WatchFolders.

		_, _ = os.Create(fmt.Sprintf("%s/%s", dir, "test0.go"))

		time.Sleep(debounceInterval / 2) // wait to enforce order of filesChanged elements

		_, _ = os.Create(fmt.Sprintf("%s/%s", dir, "test1.go"))

		const expected = 1

		assert.Eventually(t, func() bool {
			return len(filesChanged) == expected
		}, time.Second, time.Millisecond*100, "expected to receive one file change only")

		cancel()
		wg.Wait()
	})

	t.Run("debounce changes until interval", func(t *testing.T) {
		t.Parallel()

		wg := sync.WaitGroup{}

		filesChanged := make(chan internal.File, 10)
		ctx, cancel := context.WithCancel(t.Context())

		dir := t.TempDir()

		wg.Add(1)
		go func(wg *sync.WaitGroup) { //nolint:wsl_v5
			pathInfos, err := internal.NewPathInfos([]string{dir})
			assert.NoError(t, err)
			err = internal.WatchFolders(ctx, pathInfos, filesChanged, debounceInterval)
			assert.NoError(t, err)
			wg.Done()
		}(&wg)

		time.Sleep(10 * time.Millisecond) // wait until the goroutine has started WatchFolders.

		_, err := os.Create(fmt.Sprintf("%s/%s", dir, "test0.go"))
		assert.NoError(t, err)

		time.Sleep(debounceInterval * 2) // wait to enforce order of filesChanged elements

		_, err = os.Create(fmt.Sprintf("%s/%s", dir, "test1.go"))
		assert.NoError(t, err)

		const expected = 2

		assert.Eventually(t, func() bool {
			return len(filesChanged) == expected
		}, time.Second, time.Millisecond*100, "expected to receive two file changes")

		cancel()
		wg.Wait()
	})

	t.Run("change files that are ignored", func(t *testing.T) {
		t.Parallel()

		wg := sync.WaitGroup{}

		filesChanged := make(chan internal.File, 1)
		ctx, cancel := context.WithCancel(t.Context())

		dir := t.TempDir()

		wg.Add(1)
		go func(wg *sync.WaitGroup) { //nolint:wsl_v5
			pathInfos, err := internal.NewPathInfos([]string{dir})
			assert.NoError(t, err)
			err = internal.WatchFolders(ctx, pathInfos, filesChanged, debounceInterval)
			assert.NoError(t, err)
			wg.Done()
		}(&wg)

		time.Sleep(10 * time.Millisecond) // wait until the goroutine has started WatchFolders.

		_, _ = os.Create(fmt.Sprintf("%s/%s", dir, "something_test.go"))

		select {
		case <-filesChanged:
			t.FailNow()
		default: // No value ready, moving on
		}

		cancel()
		wg.Wait()
	})

	t.Run("watch multiple folders and detect changes in both", func(t *testing.T) {
		t.Parallel()

		wg := sync.WaitGroup{}

		filesChanged := make(chan internal.File, 2)
		ctx, cancel := context.WithCancel(t.Context())

		dir1 := t.TempDir()
		dir2 := t.TempDir()

		wg.Add(1)
		go func(wg *sync.WaitGroup) { //nolint:wsl_v5
			pathInfos, err := internal.NewPathInfos([]string{dir1, dir2})
			assert.NoError(t, err)
			err = internal.WatchFolders(ctx, pathInfos, filesChanged, debounceInterval)
			assert.NoError(t, err)
			wg.Done()
		}(&wg)

		time.Sleep(10 * time.Millisecond) // wait until the goroutine has started WatchFolderss.

		_, _ = os.Create(fmt.Sprintf("%s/%s", dir1, "file1.go"))

		time.Sleep(350 * time.Millisecond) // wait for debounce interval to pass

		_, _ = os.Create(fmt.Sprintf("%s/%s", dir2, "file2.go"))

		const expected = 2

		assert.Eventually(t, func() bool {
			return len(filesChanged) == expected
		}, time.Second, time.Millisecond*100, "expected to receive two file changes")

		// Verify both files were detected
		detectedFiles := make(map[string]bool)

		for range expected {
			file := <-filesChanged
			detectedFiles[string(file)] = true
		}

		assert.True(t, detectedFiles[dir1+"/file1.go"], "file1.go from dir1 should be detected")
		assert.True(t, detectedFiles[dir2+"/file2.go"], "file2.go from dir2 should be detected")

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
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			got := tt.file.IsFrontendSource()
			assert.Equal(t, tt.observed, got)
		})
	}
}
