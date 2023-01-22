package cmd_test

import (
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/arrower/cmd"
)

func TestRunCmd(t *testing.T) {
	t.Parallel()

	t.Run("run", func(t *testing.T) {
		t.Parallel()

		var (
			wg       = sync.WaitGroup{}
			osSignal = cmd.NewInterruptSignalChannel()
		)

		wg.Add(1)
		go func() {
			output, err := executeCommand(cmd.NewArrowerCLI(osSignal), "run")
			output = strings.ToLower(output)
			assert.NoError(t, err)

			assert.Contains(t, output, "run arrower")
			assert.Contains(t, output, "waiting for shutdown")
			assert.Contains(t, output, "shutdown signal received")
			assert.Contains(t, output, "done")

			wg.Done()
		}()

		osSignal <- syscall.SIGTERM
		wg.Wait()
	})

	t.Run("don't allow sub commands", func(t *testing.T) {
		t.Parallel()

		output, err := executeCommand(NewTestArrowerCLI(), "run", "sub-command")
		assert.Error(t, err)
		assert.Contains(t, output, "unknown command")
		assert.Contains(t, output, "Usage:")
	})

	t.Run("help message does not show use of flags", func(t *testing.T) {
		t.Parallel()

		output, err := executeCommand(NewTestArrowerCLI(), "run", "sub-command")
		assert.Error(t, err)
		assert.Contains(t, output, "unknown command")
		assert.NotContains(t, output, "[flags]")
	})
}
