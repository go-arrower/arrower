package cmd

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:                   "version",
		Short:                 "Print the version number of Arrower",
		Long:                  ``,
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			hash, ts := getVersionHashAndTimestamp()

			fmt.Fprintf(cmd.OutOrStdout(), "arrower version: %s from %s\n", hash, ts)
		},
	}
}

// getVersionHashAndTimestamp returns the last git hash and commit timestamp.
func getVersionHashAndTimestamp() (string, string) {
	hash, timestamp, modified := readBuildInfo()

	if modified || hash == "" {
		return "@latest", time.Now().UTC().Format("2006-01-02T15:04:05Z")
	}

	return hash, timestamp
}

// readBuildInfo returns the last commit hash, commit timestamp, and if the binary contains uncommitted code.
// The information needs to be available to the `go build` command. `go run` and `go test` do not contain that info.
func readBuildInfo() (string, string, bool) {
	var (
		commitHash  string
		commitTS    string
		vcsModified string
	)

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings { // called from a Go test info.Settings is always empty: []
			if setting.Key == "vcs.revision" {
				commitHash = setting.Value
			}

			if setting.Key == "vcs.time" {
				commitTS = setting.Value
			}

			// if true, the binary builds from uncommitted changes
			if setting.Key == "vcs.modified" {
				vcsModified = setting.Value
			}
		}
	}

	return commitHash, commitTS, vcsModified == "true"
}
