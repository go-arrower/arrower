package internal

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/fatih/color" //nolint:misspell
)

var (
	ErrBuildFailed      = errors.New("could not build app binary")
	ErrRunFailed        = errors.New("could not run the app")
	ErrStopFailed       = errors.New("could not stop app")
	ErrBuildCleanFailed = errors.New("could not delete app binary")
)

// BuildAndRunApp will build and run the developer's app at the given path. It returns a cleanup function,
// used to stop the app and leave a clean directory.
func BuildAndRunApp(appPath string) (func() error, error) {
	yellow := color.New(color.FgYellow, color.Bold).PrintlnFunc()
	binaryPath := "./app"

	//
	// build the app
	yellow("building...")

	cmd := exec.Command("go", "build", "-o", binaryPath, appPath)
	cmd.Dir = appPath
	// cmd.Stderr = os.Stdout // show error message of the `go build` command

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBuildFailed, err)
	}

	//
	// run the app
	yellow("running...")

	cmd = exec.Command(binaryPath)
	cmd.Stdout = os.Stdout // stream output of app to same terminal as arrower is running in
	cmd.Dir = appPath

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRunFailed, err)
	}

	return func() error { // this function does shutdown and cleanup for the running app from cmd.
		//
		// send shutdown signal for graceful shutdown to app.
		err := cmd.Process.Signal(syscall.SIGTERM)
		if err != nil {
			return fmt.Errorf("%w: send sigterm failed: %v", ErrStopFailed, err)
		}

		//
		// wait for shutdown of the app.
		err = cmd.Wait()
		if err != nil &&
			err.Error() != "signal: terminated" && // in case: this cleanup function is called before the app started
			err.Error() != "exit status 2" { // in case of a panic: don't return so cleanup can continue
			return fmt.Errorf("%w: could not wait for app to complete: %v", ErrStopFailed, err)
		}

		//
		// delete the app binary, to leave a clean working directory.
		err = os.Remove(fmt.Sprintf("%s/%s", appPath, binaryPath))
		if err != nil {
			return fmt.Errorf("%w: %v", ErrBuildCleanFailed, err)
		}

		return nil
	}, nil
}
