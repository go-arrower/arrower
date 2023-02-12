package internal

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/fatih/color" //nolint:misspell
)

var (
	ErrBuildFailed      = errors.New("could not build app binary")
	ErrRunFailed        = errors.New("could not run the app")
	ErrStopFailed       = errors.New("could not stop app")
	ErrBuildCleanFailed = errors.New("could not delete app binary")
)

// BuildAndRunApp will build the developer's app at the given appPath to the destination binaryPath.
// It returns a cleanup function, used to stop the app and leave a clean directory.
func BuildAndRunApp(w io.Writer, appPath string, binaryPath string) (func() error, error) {
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

	cmd := exec.Command("go", "build", "-o", binaryPath, appPath)
	cmd.Dir = appPath
	// cmd.Stderr = os.Stdout // show error message of the `go build` command

	err := cmd.Run()
	if err != nil {
		err = os.Remove(binaryPath)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBuildCleanFailed, err)
		}

		return nil, fmt.Errorf("%w: %v", ErrBuildFailed, err)
	}

	//
	// run the app
	yellow(w, "running...")

	cmd = exec.Command(binaryPath)
	cmd.Stdout = w // stream output to some io.Writer
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
		//
		// wait for shutdown of the app.
		appStopped := make(chan error)
		go func() { //nolint:wsl
			err := cmd.Wait()
			log.Println("failed to wait: ", err)

			close(appStopped)
		}()

		//
		// send shutdown signal for graceful shutdown to app.
		err := cmd.Process.Signal(syscall.SIGTERM)
		if err != nil {
			return fmt.Errorf("%w: send sigterm failed: %v", ErrStopFailed, err)
		}

		// wait for our process to die before we return or hard kill
		const waitBeforeKill = 2
		select {
		case <-time.After(waitBeforeKill * time.Second):
			if err = cmd.Process.Kill(); err != nil {
				log.Println("failed to kill: ", err)
			}
		case <-appStopped:
		}

		//
		// delete the app binary, to leave a clean working directory.
		err = os.Remove(binaryPath)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrBuildCleanFailed, err)
		}

		return nil
	}
}

// RandomBinaryPath return a unique path to build the app binary in the operating systems /tmp folder.
func RandomBinaryPath() (string, error) {
	f, err := os.CreateTemp(os.TempDir(), "arrower-app-")
	if err != nil {
		return "", fmt.Errorf("%w: could not create random build path: %v", ErrBuildFailed, err)
	}

	return f.Name(), nil
}
