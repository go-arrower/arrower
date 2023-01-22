package cmd_test

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/arrower/cmd"
)

// executeCommand is a helper that executes a cobra command and returns its output.
func executeCommand(root *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)

	root.SetArgs(args)

	_, err := root.ExecuteC()

	return buf.String(), err
}

func TestExecute(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, cmd.Execute, "execute whole cli like main.go does")
}

func TestRootCmd(t *testing.T) {
	t.Parallel()

	t.Run("no command: show help & list of commands", func(t *testing.T) {
		t.Parallel()

		// leaving args empty or "" leads to: unknown command error, so it's set explicitly to empty slice
		output, err := executeCommand(NewTestArrowerCLI(), []string{}...)
		assert.NoError(t, err)
		assert.Contains(t, output, "Available Commands:")
	})

	t.Run("unknown command: show help & list of commands", func(t *testing.T) {
		t.Parallel()

		output, err := executeCommand(NewTestArrowerCLI(), "non-ex-command")
		assert.Error(t, err)
		assert.Contains(t, output, "Available Commands:")
	})

	t.Run("help message does not show use of flags", func(t *testing.T) {
		t.Parallel()

		output, err := executeCommand(NewTestArrowerCLI(), []string{}...)
		assert.NoError(t, err)
		assert.NotContains(t, output, "[flags]")
	})
}
