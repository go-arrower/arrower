package internal_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/arrower/internal"
)

func TestWatchBuildAndRunApp(t *testing.T) {
	t.Parallel()

	const exampleServer = "/example-server"

	t.Run("start and stop", func(t *testing.T) {
		t.Parallel()

		wg := sync.WaitGroup{}

		buf := &SyncBuffer{}
		dir := t.TempDir()
		copyDir(t, "./testdata/example-server", dir)
		dir += exampleServer

		browserStarted := false

		ctx, cancel := context.WithCancel(context.Background())

		wg.Add(1)
		go func() {
			err := internal.WatchBuildAndRunApp(ctx, buf, dir, make(chan internal.File, 1), func(url string) error {
				browserStarted = true

				return nil
			})
			assert.NoError(t, err)
			assert.True(t, browserStarted)
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

		buf := &SyncBuffer{}
		dir := t.TempDir()
		copyDir(t, "./testdata/example-server", dir)
		dir += exampleServer

		ctx, cancel := context.WithCancel(context.Background())

		wg.Add(1)
		go func() {
			err := internal.WatchBuildAndRunApp(ctx, buf, dir, make(chan internal.File, 1), noBrowser)
			assert.NoError(t, err)
			wg.Done()
		}()

		time.Sleep(500 * time.Millisecond) // time for the example-server to start

		_ = os.WriteFile(fmt.Sprintf("%s/%s", dir, "/test0.go"), []byte(`package main`), 0o644) //nolint:gosec

		time.Sleep(500 * time.Millisecond) // time for arrower to detect the fs change

		assert.Contains(t, buf.String(), arrowerCLIChangeDetected)
		assert.NotContains(t, buf.String(), internal.ErrBuildFailed.Error())

		cancel()
		wg.Wait()
	})

	t.Run("signal browser about hot reload", func(t *testing.T) {
		t.Parallel()

		wg := sync.WaitGroup{}

		buf := &SyncBuffer{}
		dir := t.TempDir()
		copyDir(t, "./testdata/example-server", dir)
		dir += exampleServer

		ctx, cancel := context.WithCancel(context.Background())
		hotReload := make(chan internal.File, 1)

		wg.Add(1)
		go func() {
			err := internal.WatchBuildAndRunApp(ctx, buf, dir, hotReload, noBrowser)
			assert.NoError(t, err)
			wg.Done()
		}()

		time.Sleep(500 * time.Millisecond) // time for the example-server to start

		_ = os.WriteFile(fmt.Sprintf("%s/%s", dir, "/some.css"), []byte(``), 0o644) //nolint:gosec

		time.Sleep(500 * time.Millisecond) // time for arrower to detect the fs change

		assert.Contains(t, buf.String(), arrowerCLIChangeDetected)
		assert.NotContains(t, buf.String(), internal.ErrBuildFailed.Error())

		f := <-hotReload
		assert.NotEmpty(t, f)
		assert.True(t, strings.HasSuffix(string(f), "/some.css"))

		cancel()
		wg.Wait()
	})
}

func copyDir(t *testing.T, oldDir string, newDir string) {
	t.Helper()

	cmd := exec.Command("cp", "--recursive", oldDir, newDir)
	err := cmd.Run()
	assert.NoError(t, err)
}

func TestOpenBrowser(t *testing.T) {
	t.Parallel()

	tests := []struct { //nolint:govet
		testName string
		url      string
		err      error
	}{
		{
			"empty url",
			"",
			fmt.Errorf("%w: invalid url", internal.ErrCommandFailed),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			got := internal.OpenBrowser(tt.url)
			assert.Equal(t, tt.err, got)
		})
	}
}
