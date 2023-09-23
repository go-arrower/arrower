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

	t.Run("fail with missing io.Writer", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		copyDir(t, "./testdata/example-cli", dir)

		_, err := internal.BuildAndRunApp(nil, dir+"/example-cli", "")
		assert.Error(t, err)
	})

	t.Run("build, run, and delete the app", func(t *testing.T) {
		t.Parallel()

		buf := &syncBuffer{}
		dir := t.TempDir()
		copyDir(t, "./testdata/example-cli", dir)
		binaryPath, _ := internal.RandomBinaryPath()

		cleanup, err := internal.BuildAndRunApp(buf, dir+"/example-cli", binaryPath)
		assert.NoError(t, err)
		assert.NotNil(t, cleanup)
		assert.Contains(t, buf.String(), arrowerCLIBuild)
		assert.Contains(t, buf.String(), arrowerCLIRun)

		// wait so the example application can start & run
		time.Sleep(50 * time.Millisecond)

		err = cleanup()
		assert.NoError(t, err)
		assert.NoFileExists(t, binaryPath)

		// ensure fmt and log output is written to the io.Writer
		assert.Contains(t, buf.String(), "hello from fmt")
		assert.Contains(t, buf.String(), "hello from log")
	})

	t.Run("empty binaryPath", func(t *testing.T) {
		t.Parallel()

		buf := &syncBuffer{}
		dir := t.TempDir()
		copyDir(t, "./testdata/example-cli", dir)

		cleanup, err := internal.BuildAndRunApp(buf, dir+"/example-cli", "")
		assert.NoError(t, err)
		assert.NotNil(t, cleanup)
		assert.Contains(t, buf.String(), arrowerCLIBuild)
		assert.Contains(t, buf.String(), arrowerCLIRun)

		err = cleanup()
		assert.NoError(t, err)
	})

	t.Run("invalid path fails the build", func(t *testing.T) {
		t.Parallel()

		buf := &syncBuffer{}
		binaryPath, _ := internal.RandomBinaryPath()

		_, err := internal.BuildAndRunApp(buf, "/non/existing/path", binaryPath)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, internal.ErrBuildFailed))
		assert.NoFileExists(t, binaryPath)
		assert.Contains(t, buf.String(), arrowerCLIBuild)
		assert.NotContains(t, buf.String(), arrowerCLIRun)
	})

	t.Run("run and stop a webserver", func(t *testing.T) {
		t.Parallel()

		buf := &syncBuffer{}
		dir := t.TempDir()
		copyDir(t, "./testdata/example-server", dir)
		binaryPath, _ := internal.RandomBinaryPath()

		stop, err := internal.BuildAndRunApp(buf, dir+"/example-server", binaryPath)
		assert.NoError(t, err)

		time.Sleep(50 * time.Millisecond) // wait so the process can actually start

		err = stop()
		assert.NoError(t, err)
		assert.NoFileExists(t, binaryPath)
		assert.Contains(t, buf.String(), arrowerCLIBuild)
		assert.Contains(t, buf.String(), arrowerCLIRun)
	})

	t.Run("panic", func(t *testing.T) {
		t.Parallel()

		buf := &syncBuffer{}
		dir := t.TempDir()
		copyDir(t, "./testdata/example-panic", dir)
		binaryPath, _ := internal.RandomBinaryPath()

		stop, err := internal.BuildAndRunApp(buf, dir+"/example-panic", binaryPath)
		assert.NoError(t, err)

		time.Sleep(50 * time.Millisecond) // wait so the process can actually start
		err = stop()
		assert.NoError(t, err)
		assert.NoFileExists(t, binaryPath)
		assert.Contains(t, buf.String(), arrowerCLIBuild)
		assert.Contains(t, buf.String(), arrowerCLIRun)
	})

	t.Run("app does not compile", func(t *testing.T) {
		t.Parallel()

		buf := &syncBuffer{}
		dir := t.TempDir()
		copyDir(t, "./testdata/example-compile-error", dir)
		binaryPath, _ := internal.RandomBinaryPath()

		cleanup, err := internal.BuildAndRunApp(buf, dir+"/example-compile-error", binaryPath)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, internal.ErrBuildFailed))
		assert.Contains(t, err.Error(), "is not in std")
		assert.Nil(t, cleanup)
	})

	t.Run("stop if sigterm is ignored", func(t *testing.T) {
		t.Parallel()

		buf := &syncBuffer{}
		dir := t.TempDir()
		copyDir(t, "./testdata/example-ignore-sigterm", dir)
		binaryPath, _ := internal.RandomBinaryPath()

		stop, err := internal.BuildAndRunApp(buf, dir+"/example-ignore-sigterm", binaryPath)
		assert.NoError(t, err)

		time.Sleep(50 * time.Millisecond) // wait so the process can actually start

		err = stop()
		assert.NoError(t, err)
		assert.NoFileExists(t, binaryPath)
		assert.Contains(t, buf.String(), arrowerCLIBuild)
		assert.Contains(t, buf.String(), arrowerCLIRun)
	})
}
