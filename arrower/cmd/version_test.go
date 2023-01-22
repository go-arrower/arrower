package cmd_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/arrower/cmd"
)

func TestVersionCmd(t *testing.T) {
	t.Parallel()

	// achieving a high test coverage without actually building the binary is difficult, as
	// debug.ReadBuildInfo()'s info.Settings called from a Go test is always empty: []

	t.Run("show version", func(t *testing.T) {
		t.Parallel()

		output, err := executeCommand(cmd.NewArrowerCLI(), "version")
		assert.NoError(t, err)
		assert.Contains(t, output, "arrower version:")
	})

	t.Run("don't allow sub commands", func(t *testing.T) {
		t.Parallel()

		output, err := executeCommand(cmd.NewArrowerCLI(), "version", "sub-command")
		assert.Error(t, err)
		assert.Contains(t, output, "unknown command")
		assert.Contains(t, output, "Usage:")
	})

	t.Run("help message does not show use of flags", func(t *testing.T) {
		t.Parallel()

		output, err := executeCommand(cmd.NewArrowerCLI(), "version", "sub-command")
		assert.Error(t, err)
		assert.Contains(t, output, "unknown command")
		assert.NotContains(t, output, "[flags]")
	})
}
