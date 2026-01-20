package cmd_test

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/go-arrower/arrower/arrower/cmd"
)

func NewTestArrowerCLI() *cobra.Command {
	return cmd.NewArrowerCLI(cmd.NewInterruptSignalChannel(), noBrowser)
}

func noBrowser(_ context.Context, _ string) error { return nil } // todo func is redundant, because already defined somewhere else
