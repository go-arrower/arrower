package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "arrower",
		Short: "Arrower is your aid, you focus on your DDD and arrower gives you a fullstack serverless experience.",
		Long: `A toolkit to get you started with your next modular monolith.
Complete documentation is available at http://arrower.org`,
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}
}

// NewArrowerCLI initialises the complete arrower cli with its commands and returns the root command.
func NewArrowerCLI(osSignal <-chan os.Signal) *cobra.Command {
	rootCmd := newRootCmd()
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newRunCmd(osSignal))

	return rootCmd
}

// Execute runs the arrower cli.
func Execute() {
	if err := NewArrowerCLI(NewInterruptSignalChannel()).Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
