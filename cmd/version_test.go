package cmd_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/cmd"
)

func TestVersion(t *testing.T) {
	t.Parallel()

	// Achieving a high test coverage without actually building the binary is difficult, as
	// runtime/debug.ReadBuildInfo()'s info.Settings called from a Go test is always empty: []
	// This could be improved by making this an integration test actually building
	// a binary with different git states (committed & uncommitted changes)
	// Left for the future => manually tested as of now

	t.Run("show version", func(t *testing.T) {
		t.Parallel()

		output, err := cmd.TestExecute(cmd.Version("arrower"), []string{}...)
		assert.NoError(t, err)
		assert.Contains(t, output, "arrower version: ", "should start with program name and `version:`")
		assert.Contains(t, output, " from ", "should contain a date indicator")
	})

	t.Run("no program name", func(t *testing.T) {
		t.Parallel()

		t.Run("command output", func(t *testing.T) {
			t.Parallel()

			output, err := cmd.TestExecute(cmd.Version(""), []string{}...)
			assert.NoError(t, err)
			assert.Contains(t, output[:8], "version:", "should not start with leading space")
			assert.NotContains(t, output, "%!(EXTRA", "should not contain fmt placeholder count mismatch error")
		})

		t.Run("help output", func(t *testing.T) {
			t.Parallel()

			output, err := cmd.TestExecute(cmd.Version(""), "-h")
			assert.NoError(t, err)
			assert.NotContains(t, output, "Print  ", "should not leaf space instead of name")
		})
	})

	t.Run("don't allow sub commands", func(t *testing.T) {
		t.Parallel()

		output, err := cmd.TestExecute(cmd.Version(""), "sub-command")
		assert.Error(t, err)
		assert.Contains(t, output, "unknown command")
		assert.Contains(t, output, "Usage:")
	})

	t.Run("help message does not show use of flags", func(t *testing.T) {
		t.Parallel()

		output, err := cmd.TestExecute(cmd.Version(""), "version", "sub-command")
		assert.Error(t, err)
		assert.Contains(t, output, "unknown command")
		assert.NotContains(t, output, "[flags]")
	})
}
