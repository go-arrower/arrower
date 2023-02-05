package internal_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/arrower/internal"
)

func TestWatchBuildAndRunApp(t *testing.T) {
	t.Parallel()

	t.Run("start and stop", func(t *testing.T) {
		t.Parallel()

		wg := sync.WaitGroup{}

		buf := &SyncBuffer{} //nolint:exhaustruct
		dir := t.TempDir()
		copyDir(t, "./testdata/example-server", dir)
		dir += "/example-server"

		ctx, cancel := context.WithCancel(context.Background())

		wg.Add(1)
		go func() {
			err := internal.WatchBuildAndRunApp(ctx, buf, dir)
			assert.NoError(t, err)
			wg.Done()
		}()

		time.Sleep(500 * time.Millisecond) // time for the example-server to start

		cancel()
		wg.Wait()

		assert.Contains(t, buf.String(), exampleServerStart)
		assert.Contains(t, buf.String(), exampleServerWaitForShutdown)
		assert.Contains(t, buf.String(), exampleServerShutdown)
		assert.Contains(t, buf.String(), exampleServerDone)
	})

	t.Run("restart app on detected file change", func(t *testing.T) {
		t.Parallel()

		wg := sync.WaitGroup{}

		buf := &SyncBuffer{} //nolint:exhaustruct
		dir := t.TempDir()
		copyDir(t, "./testdata/example-server", dir)
		dir += "/example-server"

		ctx, cancel := context.WithCancel(context.Background())

		wg.Add(1)
		go func() {
			err := internal.WatchBuildAndRunApp(ctx, buf, dir)
			assert.NoError(t, err)
			wg.Done()
		}()

		time.Sleep(500 * time.Millisecond) // time for the example-server to start

		_ = os.WriteFile(fmt.Sprintf("%s/%s", dir, "/test0.go"), []byte(`package main`), 0o644) //nolint:gosec

		time.Sleep(500 * time.Millisecond) // time for arrower to detect the fs change
		assert.Contains(t, buf.String(), arrowerCLIChangeDetected)

		cancel()
		wg.Wait()
	})

	t.Run("debounce app runs", func(t *testing.T) {
		t.Parallel()
	})
}

func copyDir(t *testing.T, oldDir string, newDir string) {
	t.Helper()

	cmd := exec.Command("cp", "--recursive", oldDir, newDir)
	err := cmd.Run()
	assert.NoError(t, err)
}
