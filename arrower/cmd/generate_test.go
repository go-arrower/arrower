package cmd_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateCmd(t *testing.T) {
	t.Parallel()

	t.Run("generate", func(t *testing.T) {
		t.Parallel()

		output, err := executeCommand(NewTestArrowerCLI(), "generate")
		assert.NoError(t, err)
		t.Log(output)
	})
}
