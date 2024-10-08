package cmd_test

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/cmd"
)

var errCmdFailed = errors.New("cmd failed")

func TestTestExecute(t *testing.T) {
	t.Parallel()

	t.Run("run command: stdout", func(t *testing.T) {
		t.Parallel()

		rootCmd := &cobra.Command{Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), "hello cmd out")
			fmt.Fprintln(os.Stdout, "hello os out")
		}}

		output, err := cmd.TestExecute(t, rootCmd)
		assert.NoError(t, err)
		assert.Contains(t, output, "hello cmd out")
		assert.Contains(t, output, "hello os out")
	})

	t.Run("run command: stderr", func(t *testing.T) {
		t.Parallel()

		rootCmd := &cobra.Command{Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), "hello cmd out")
			fmt.Fprintln(os.Stderr, "hello os out")
		}}

		output, err := cmd.TestExecute(t, rootCmd)
		assert.NoError(t, err)
		assert.Contains(t, output, "hello cmd out")
		assert.Contains(t, output, "hello os out")
	})

	t.Run("return error of command", func(t *testing.T) {
		t.Parallel()

		rootCmd := &cobra.Command{RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 2 {
				return fmt.Errorf("%w", errCmdFailed)
			}

			return nil
		}}

		output, err := cmd.TestExecute(t, rootCmd, "some", "args")
		assert.ErrorIs(t, err, errCmdFailed)
		assert.Contains(t, output, errCmdFailed.Error())
	})

	t.Run("test command in parallel", func(t *testing.T) {
		t.Parallel()

		rootCmd := &cobra.Command{Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), "hello cmd out")
			fmt.Fprintln(os.Stdout, "hello os out")
		}}

		for range 10 {
			go func() {
				output, err := cmd.TestExecute(t, rootCmd)
				assert.NoError(t, err)
				assert.Contains(t, output, "hello cmd out")
				assert.Contains(t, output, "hello os out")
			}()
		}
	})
}
