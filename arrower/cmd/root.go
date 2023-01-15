package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "arrower",
	Short: "Arrower is your aid, you focus on your DDD and arrower gives you a fullstack serverless experience.",
	Long: `A toolkit to get you started with your next modular monolith.
Complete documentation is available at http://arrower.org`,
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
