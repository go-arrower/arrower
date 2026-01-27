//nolint:misspell // external library uses "color" (American spelling), not "colour"
package internal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
)

var (
	ErrBuildFailed      = errors.New("could not build app binary")
	ErrRunFailed        = errors.New("could not run the app")
	ErrStopFailed       = errors.New("could not stop app")
	ErrBuildCleanFailed = errors.New("could not delete app binary")
)

// BuildAndRunApp will build the developer's app at the given appPath to the destination binaryPath.
// It returns a cleanup function, used to stop the app and leave a clean directory.
func BuildAndRunApp(ctx context.Context, w io.Writer, appPath string, binaryPath string) (func() error, error) {
	yellow := color.New(color.FgYellow, color.Bold).FprintlnFunc()

	if w == nil {
		return nil, fmt.Errorf("%w: missing io.Writer", ErrBuildFailed)
	}

	if binaryPath == "" {
		randPath, err := RandomBinaryPath()
		if err != nil {
			return nil, fmt.Errorf("%w", err)
		}

		binaryPath = randPath
	}

	//
	// build the app
	yellow(w, "building...")

	buf := &bytes.Buffer{}

	cmd := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, appPath)
	cmd.Dir = appPath
	cmd.Stderr = buf // show error message of the `go build` command

	err := cmd.Run()
	if err != nil {
		goModTidyNeeded := strings.Contains(buf.String(), "go: updates to go.mod needed;") ||
			strings.Contains(buf.String(), "no required module provides package")
		if goModTidyNeeded {
			cmd = exec.CommandContext(ctx, "go", "mod", "tidy")
			cmd.Stdout = w // stream output to same io.Writer
			cmd.Stderr = w // stream output to same io.Writer
			cmd.Dir = appPath

			err = cmd.Run()
			if err != nil {
				return nil, fmt.Errorf("%w: %v", ErrBuildFailed, err)
			}

			buf.Reset()

			cmd = exec.CommandContext(ctx, "go", "build", "-o", binaryPath, appPath)
			cmd.Dir = appPath
			cmd.Stderr = buf // show error message of the `go build` command

			err = cmd.Run()
			if err != nil {
				err2 := os.Remove(binaryPath)
				if err2 != nil {
					return nil, fmt.Errorf("%w: %v", ErrBuildCleanFailed, err2)
				}

				return nil, fmt.Errorf("%w: %v: %v", ErrBuildFailed, err, buf.String())
			}
		} else {
			err2 := os.Remove(binaryPath)
			if err2 != nil {
				return nil, fmt.Errorf("%w: %v", ErrBuildCleanFailed, err2)
			}

			return nil, fmt.Errorf("%w: %v: %v", ErrBuildFailed, err, buf.String())
		}
	}

	//
	// run the app
	yellow(w, "running...")

	cmd = exec.CommandContext(context.Background(), binaryPath)
	cmd.Stdout = w // stream output to same io.Writer
	cmd.Stderr = w // stream output to same io.Writer
	cmd.Dir = appPath
	// prevent the cmd to be stopped on strg +c from parent, so graceful shutdown works,
	// see: https://stackoverflow.com/a/33171307
	cmd.SysProcAttr = &syscall.SysProcAttr{ //nolint:exhaustruct
		Setpgid: true,
	}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRunFailed, err)
	}

	return stopAndCleanup(cmd, binaryPath), nil
}

// stopAndCleanup returns a function that: shuts down running app from cmd and cleans up afterwards.
func stopAndCleanup(cmd *exec.Cmd, binaryPath string) func() error {
	return func() error {
		defer func() {
			err := deleteAppBinary(binaryPath)
			if err != nil {
				log.Println(err)
			}
		}()

		//
		// wait for shutdown of the app.
		appStopped := make(chan error)
		go func() { //nolint:wsl_v5
			err := waitForCmdToFinish(cmd)
			if err != nil {
				log.Println(err)
			}

			close(appStopped)
		}()

		//
		// send shutdown signal for graceful shutdown to app.
		err := cmd.Process.Signal(syscall.SIGTERM)
		if err != nil && !errors.Is(err, os.ErrProcessDone) {
			return fmt.Errorf("%w: send sigterm failed: %v", ErrStopFailed, err)
		}

		// wait for our process to die before we return or hard kill
		const waitBeforeKill = 2
		select {
		case <-time.After(waitBeforeKill * time.Second):
			if err = cmd.Process.Kill(); err != nil {
				return fmt.Errorf("%w: failed to kill: %v", ErrStopFailed, err)
			}
		case <-appStopped:
		}

		return nil
	}
}

// deleteAppBinary deletes the app binary, to leave a clean working directory.
func deleteAppBinary(binaryPath string) error {
	err := os.Remove(binaryPath)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBuildCleanFailed, err)
	}

	return nil
}

func waitForCmdToFinish(cmd *exec.Cmd) error {
	err := cmd.Wait()
	if err != nil &&
		err.Error() != "signal: terminated" && // in case: this cleanup function is called before the app started
		err.Error() != "signal: killed" && // UNCLEAR: check required when test is run from CLI, not if it is run from IDE
		err.Error() != "exit status 2" { // in case of a panic: don't return so cleanup can continue
		return fmt.Errorf("%w: could not wait for app to complete: %v", ErrStopFailed, err)
	}

	return nil
}

// RandomBinaryPath return a unique path to build the app binary in the operating systems /tmp folder.
func RandomBinaryPath() (string, error) {
	f, err := os.CreateTemp(os.TempDir(), "arrower-app-")
	if err != nil {
		return "", fmt.Errorf("%w: could not create random build path: %v", ErrBuildFailed, err)
	}

	return f.Name(), nil
}
