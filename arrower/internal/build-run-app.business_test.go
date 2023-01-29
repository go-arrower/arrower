package internal_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/arrower/internal"
)

func TestBuildAndRunApp(t *testing.T) {
	t.Parallel()

	binaryPath := "./app"

	t.Run("build, run, and delete the app", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		copyDir(t, "./testdata/example-cli", dir)

		cleanup, err := internal.BuildAndRunApp(dir + "/example-cli")
		assert.NoError(t, err)
		assert.NotNil(t, cleanup)

		err = cleanup()
		assert.NoError(t, err)
		assert.NoFileExists(t, binaryPath)
	})

	t.Run("use non local path", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		copyDir(t, "./testdata/example-server", dir)

		cleanup, err := internal.BuildAndRunApp(dir + "/example-server")
		assert.NoError(t, err)

		err = cleanup()
		assert.NoError(t, err)
		assert.NoFileExists(t, binaryPath)
	})

	t.Run("invalid path fails the build", func(t *testing.T) {
		t.Parallel()

		_, err := internal.BuildAndRunApp("/non/existing/path")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, internal.ErrBuildFailed))
		assert.NoFileExists(t, binaryPath)
	})

	t.Run("stop webserver", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		copyDir(t, "./testdata/example-server", dir)

		stop, err := internal.BuildAndRunApp(dir + "/example-server")
		assert.NoError(t, err)

		time.Sleep(50 * time.Millisecond) // wait so the process can actually start

		err = stop()
		assert.NoError(t, err)
		assert.NoFileExists(t, binaryPath)
	})

	t.Run("panic", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		copyDir(t, "./testdata/example-panic", dir)

		stop, err := internal.BuildAndRunApp(dir + "/example-panic")
		assert.NoError(t, err)

		time.Sleep(50 * time.Millisecond) // wait so the process can actually start
		err = stop()
		assert.NoError(t, err)
		assert.NoFileExists(t, binaryPath)
	})

	/*
		test cases:
			a path that is not the current path, but in t.TempDir dir := t.TempDir()
			program that ignores sigterm
	*/

	_ = 0
}
