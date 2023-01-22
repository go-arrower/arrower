package cmd_test

import (
	"github.com/spf13/cobra"

	"github.com/go-arrower/arrower/arrower/cmd"
)

func NewTestArrowerCLI() *cobra.Command {
	return cmd.NewArrowerCLI(cmd.NewInterruptSignalChannel())
}
