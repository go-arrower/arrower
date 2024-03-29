package cmd_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateCmd(t *testing.T) {
	t.Parallel()

	t.Run("generate", func(t *testing.T) {
		t.Parallel()

		err := os.WriteFile("go.mod", []byte(`module example/app`), 0o600)
		assert.NoError(t, err)

		output, err := executeCommand(NewTestArrowerCLI(), "generate", "request", "some-test")
		assert.NoError(t, err)
		assert.Contains(t, output, "New request generated")
		assert.Contains(t, output, "some-test.request.go")
		assert.Contains(t, output, "some-test.request_test.go")

		err = os.Remove("go.mod")
		assert.NoError(t, err)
		err = os.Remove("some-test.request.go")
		assert.NoError(t, err)
		err = os.Remove("some-test.request_test.go")
		assert.NoError(t, err)
	})
}
