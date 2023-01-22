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

	output, err := executeCommand(cmd.NewArrowerCLI(), "version")
	assert.NoError(t, err)
	assert.Contains(t, output, "arrower version:")
}
