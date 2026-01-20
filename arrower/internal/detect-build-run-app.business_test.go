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

	"github.com/go-arrower/arrower/arrower/hooks"
	"github.com/go-arrower/arrower/arrower/internal"
)

func TestWatchBuildAndRunApp(t *testing.T) {
	t.Parallel()

	const exampleServer = "/example-server"

	t.Run("start and stop", func(t *testing.T) {
		t.Parallel()

		wg := sync.WaitGroup{}

		buf := &syncBuffer{}
		dir := t.TempDir()
		copyDir(t, "./testdata/example-server", dir)
		dir += exampleServer

		browserStarted := false

		ctx, cancel := context.WithCancel(t.Context())

		wg.Add(1)
		go func(wg *sync.WaitGroup) { //nolint:wsl_v5
			err := internal.WatchBuildAndRunApp(ctx, buf, dir, hooks.Hooks{},
				make(chan internal.File, 1),
				func(_ context.Context, _ string) error {
					browserStarted = true

					return nil
				})
			assert.NoError(t, err)
			assert.True(t, browserStarted)
			wg.Done()
		}(&wg)

		waitUntilRunning(buf)

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

		buf := &syncBuffer{}
		dir := t.TempDir()
		copyDir(t, "./testdata/example-server", dir)
		dir += exampleServer

		ctx, cancel := context.WithCancel(context.Background())

		wg.Add(1)
		go func(wg *sync.WaitGroup) { //nolint:wsl_v5
			err := internal.WatchBuildAndRunApp(ctx, buf, dir, hooks.Hooks{}, make(chan internal.File, 1), noBrowser)
			assert.NoError(t, err)
			wg.Done()
		}(&wg)

		waitUntilRunning(buf) // time for the example-server to start

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

		buf := &syncBuffer{}
		dir := t.TempDir()
		copyDir(t, "./testdata/example-server", dir)
		dir += exampleServer

		ctx, cancel := context.WithCancel(context.Background())
		hotReload := make(chan internal.File, 1)

		wg.Add(1)
		go func(wg *sync.WaitGroup) { //nolint:wsl_v5
			err := internal.WatchBuildAndRunApp(ctx, buf, dir, hooks.Hooks{}, hotReload, noBrowser)
			assert.NoError(t, err)

			wg.Done()
		}(&wg)

		waitUntilRunning(buf) // time for the example-server to start

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

	// todo test the execution of OnChanged plugins
}

func copyDir(t *testing.T, oldDir string, newDir string) {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), "cp", "--recursive", oldDir, newDir)
	err := cmd.Run()
	assert.NoError(t, err)
}

// wait until time for the example-server starts.
// CI is a looot slower than local machine, so wait longer accordingly.
func waitUntilRunning(buf *syncBuffer) {
	const maxTries = 50

	retriesUntilServerStarted := 0
	for retriesUntilServerStarted < maxTries {
		time.Sleep(100 * time.Millisecond)

		retriesUntilServerStarted--

		if strings.Contains(buf.String(), "running...") {
			break
		}
	}

	time.Sleep(time.Duration(retriesUntilServerStarted) * 200 * time.Millisecond) // extra time to execute the server
}

func TestOpenBrowser(t *testing.T) {
	t.Parallel()

	tests := []struct {
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
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			got := internal.OpenBrowser(t.Context(), tt.url)
			assert.Equal(t, tt.err, got)
		})
	}
}
