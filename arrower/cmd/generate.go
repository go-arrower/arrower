package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color" //nolint:misspell
	"github.com/spf13/cobra"

	"github.com/go-arrower/arrower/arrower/internal/generate"
)

func newGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "generate",
		Aliases:               []string{"gen"},
		Short:                 "Code generation to safe you from dealing with boilerplate",
		Long:                  ``,
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "arrower generate\n")

			return nil
		},
	}

	cmd.AddCommand(newGenerateRequest())

	return cmd
}

func newGenerateRequest() *cobra.Command {
	return &cobra.Command{
		Use:     "request",
		Aliases: []string{"req"},
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("%w", err)
			}

			files, err := generate.Generate(path, args, generate.Request)
			if err != nil {
				return fmt.Errorf("%w", err)
			}

			blue := color.New(color.FgBlue, color.Bold).FprintfFunc()

			fmt.Fprintf(cmd.OutOrStdout(), "New request generated\n")
			for _, f := range files {
				blue(cmd.OutOrStdout(), "%s\n", f)
			}

			return nil
		},
	}
}
