package cmd_test

import (
	"github.com/spf13/cobra"

	"github.com/go-arrower/arrower/arrower/cmd"
	"github.com/go-arrower/arrower/arrower/internal"
)

func testArrowerCLI() *cobra.Command {
	return cmd.NewArrowerCLI(cmd.NewInterruptSignalChannel(), internal.NoBrowser)
}
