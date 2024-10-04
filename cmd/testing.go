package cmd

import (
	"bytes"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/spf13/cobra"
)

// mu synchronisation is required:
// As TestExecute accepts a pointer to the cobra command,
// concurrent tests will create a race condition.
// This will also minimise the potential race condition
// when capturing & restoring os.Stdout & os.Stderr.
var mu sync.Mutex

// TestExecute is a helper that executes a cobra command and returns its output and error.
func TestExecute(t *testing.T, command *cobra.Command, args ...string) (string, error) {
	t.Helper()

	mu.Lock()
	defer mu.Unlock()

	// capture command outputs
	buf := new(syncBuffer)
	command.SetOut(buf)
	command.SetErr(buf)

	// capture os outputs
	storeStdout := os.Stdout
	storeStderr := os.Stderr

	rOut, wOut, err := os.Pipe()
	assert.NoError(t, err)
	rErr, wErr, err := os.Pipe()
	assert.NoError(t, err)
	os.Stdout = wOut
	os.Stderr = wErr

	// execute the command
	command.SetArgs(args)
	_, cmdErr := command.ExecuteC()

	// restore os outputs
	err = wOut.Close()
	assert.NoError(t, err)
	err = wErr.Close()
	assert.NoError(t, err)

	all, err := io.ReadAll(rOut)
	assert.NoError(t, err)
	_, err = buf.Write(all)
	assert.NoError(t, err)

	all, err = io.ReadAll(rErr)
	assert.NoError(t, err)
	_, err = buf.Write(all)
	assert.NoError(t, err)

	os.Stdout = storeStdout
	os.Stderr = storeStderr

	return buf.String(), cmdErr
}

// syncBuffer is a helper implementing io.Writer, used for concurrency save testing.
type syncBuffer struct {
	b bytes.Buffer
	m sync.Mutex
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.m.Lock()
	defer b.m.Unlock()

	return b.b.Write(p) //nolint:wrapcheck
}

func (b *syncBuffer) String() string {
	b.m.Lock()
	defer b.m.Unlock()

	return b.b.String()
}
