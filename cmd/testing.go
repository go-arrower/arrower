package cmd

import (
	"bytes"
	"os"

	"github.com/spf13/cobra"
)

// TestExecute is a helper that executes a cobra command and returns its output and error.
func TestExecute(root *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)

	root.SetArgs(args)

	_, err := root.ExecuteC()

	os.Stdout.Sync()
	os.Stderr.Sync()

	return buf.String(), err
}
