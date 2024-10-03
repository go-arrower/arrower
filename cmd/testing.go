package cmd

import (
	"bytes"
	"sync"

	"github.com/spf13/cobra"
)

var mu sync.Mutex

// TestExecute is a helper that executes a cobra command and returns its output and error.
func TestExecute(root *cobra.Command, args ...string) (string, error) {
	mu.Lock()
	defer mu.Unlock()

	buf := new(syncBuffer)
	root.SetOut(buf)
	root.SetErr(buf)

	root.SetArgs(args)

	_, err := root.ExecuteC()

	return buf.String(), err
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
