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

	t.Run("build, run, and delete the app", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		copyDir(t, "./testdata/example-cli", dir)
		binaryPath, _ := internal.RandomBinaryPath()

		cleanup, err := internal.BuildAndRunApp(dir+"/example-cli", binaryPath)
		assert.NoError(t, err)
		assert.NotNil(t, cleanup)

		err = cleanup()
		assert.NoError(t, err)
		assert.NoFileExists(t, binaryPath)
	})

	t.Run("empty binaryPath", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		copyDir(t, "./testdata/example-cli", dir)

		cleanup, err := internal.BuildAndRunApp(dir+"/example-cli", "")
		assert.NoError(t, err)
		assert.NotNil(t, cleanup)

		err = cleanup()
		assert.NoError(t, err)
	})

	t.Run("invalid path fails the build", func(t *testing.T) {
		t.Parallel()

		binaryPath, _ := internal.RandomBinaryPath()

		_, err := internal.BuildAndRunApp("/non/existing/path", binaryPath)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, internal.ErrBuildFailed))
		assert.NoFileExists(t, binaryPath)
	})

	t.Run("run and stop a webserver", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		copyDir(t, "./testdata/example-server", dir)
		binaryPath, _ := internal.RandomBinaryPath()

		stop, err := internal.BuildAndRunApp(dir+"/example-server", binaryPath)
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
		binaryPath, _ := internal.RandomBinaryPath()

		stop, err := internal.BuildAndRunApp(dir+"/example-panic", binaryPath)
		assert.NoError(t, err)

		time.Sleep(50 * time.Millisecond) // wait so the process can actually start
		err = stop()
		assert.NoError(t, err)
		assert.NoFileExists(t, binaryPath)
	})
}
