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
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}
}

func NewArrowerCLI() *cobra.Command {
	rootCmd := newRootCmd()
	rootCmd.AddCommand(newVersionCmd())

	return rootCmd
}

func Execute() {
	if err := NewArrowerCLI().Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
