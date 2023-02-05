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

		dir := t.TempDir()
		copyDir(t, "./testdata/example-server", dir)
		dir += "/example-server"

		ctx, cancel := context.WithCancel(context.Background())

		wg.Add(1)
		go func() {
			err := internal.WatchBuildAndRunApp(ctx, dir)
			assert.NoError(t, err)
			wg.Done()
		}()

		time.Sleep(100 * time.Millisecond) // time for the example-server to start

		cancel()
		wg.Wait()
	})

	t.Run("restart app on detected file change", func(t *testing.T) {
		t.Parallel()

		wg := sync.WaitGroup{}

		dir := t.TempDir()
		copyDir(t, "./testdata/example-server", dir)
		dir += "/example-server"

		ctx, cancel := context.WithCancel(context.Background())

		wg.Add(1)
		go func() {
			err := internal.WatchBuildAndRunApp(ctx, dir)
			assert.NoError(t, err)
			wg.Done()
		}()

		time.Sleep(500 * time.Millisecond) // time for the example-server to start

		_ = os.WriteFile(fmt.Sprintf("%s/%s", dir, "/test0.go"), []byte(`package main`), 0o644) //nolint:gosec

		time.Sleep(500 * time.Millisecond) // time for arrower to detect the fs change
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
