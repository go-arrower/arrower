//nolint:misspell // external library uses "color" (American spelling), not "colour"
package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
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
			return cmd.Help()
		},
	}

	cmd.AddCommand(newGenerateUsecase())
	cmd.AddCommand(newGenerateRequest())
	cmd.AddCommand(newGenerateCommand())
	cmd.AddCommand(newGenerateQuery())
	cmd.AddCommand(newGenerateJob())

	return cmd
}

func newGenerateUsecase() *cobra.Command {
	return &cobra.Command{
		Use:     "usecase",
		Aliases: []string{"uc"},
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("%w", err)
			}

			files, err := generate.Generate(cmd.Context(), path, args, generate.Unknown)
			if err != nil {
				return fmt.Errorf("%w", err)
			}

			blue := color.New(color.FgBlue, color.Bold).FprintlnFunc()
			yellow := color.New(color.FgYellow, color.Bold).FprintlnFunc()

			blue(cmd.OutOrStdout(), "New usecase generated")
			for _, f := range files { //nolint:wsl_v5
				yellow(cmd.OutOrStdout(), f)
			}

			return nil
		},
	}
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

			files, err := generate.Generate(cmd.Context(), path, args, generate.Request)
			if err != nil {
				return fmt.Errorf("%w", err)
			}

			blue := color.New(color.FgBlue, color.Bold).FprintlnFunc()
			yellow := color.New(color.FgYellow, color.Bold).FprintlnFunc()

			blue(cmd.OutOrStdout(), "New request generated")
			for _, f := range files { //nolint:wsl_v5
				yellow(cmd.OutOrStdout(), f)
			}

			return nil
		},
	}
}

func newGenerateCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "command",
		Aliases: []string{"cmd"},
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("%w", err)
			}

			files, err := generate.Generate(cmd.Context(), path, args, generate.Command)
			if err != nil {
				return fmt.Errorf("%w", err)
			}

			blue := color.New(color.FgBlue, color.Bold).FprintlnFunc()
			yellow := color.New(color.FgYellow, color.Bold).FprintlnFunc()

			blue(cmd.OutOrStdout(), "New command generated\n")
			for _, f := range files { //nolint:wsl_v5
				yellow(cmd.OutOrStdout(), f)
			}

			return nil
		},
	}
}

func newGenerateQuery() *cobra.Command {
	return &cobra.Command{
		Use: "query",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("%w", err)
			}

			files, err := generate.Generate(cmd.Context(), path, args, generate.Query)
			if err != nil {
				return fmt.Errorf("%w", err)
			}

			blue := color.New(color.FgBlue, color.Bold).FprintlnFunc()
			yellow := color.New(color.FgYellow, color.Bold).FprintlnFunc()

			blue(cmd.OutOrStdout(), "New query generated\n")

			for _, f := range files {
				yellow(cmd.OutOrStdout(), f)
			}

			return nil
		},
	}
}

func newGenerateJob() *cobra.Command {
	return &cobra.Command{
		Use: "job",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("%w", err)
			}

			files, err := generate.Generate(cmd.Context(), path, args, generate.Job)
			if err != nil {
				return fmt.Errorf("%w", err)
			}

			blue := color.New(color.FgBlue, color.Bold).FprintlnFunc()
			yellow := color.New(color.FgYellow, color.Bold).FprintlnFunc()

			blue(cmd.OutOrStdout(), "New job generated\n")
			for _, f := range files { //nolint:wsl_v5
				yellow(cmd.OutOrStdout(), f)
			}

			return nil
		},
	}
}
