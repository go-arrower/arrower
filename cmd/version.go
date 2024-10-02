package cmd

import (
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Version returns a `version` command to be added to any cobra (root) command.
func Version(name string) *cobra.Command {
	short := "Print " + name + " version"
	if strings.TrimSpace(name) == "" {
		short = "Print version"
	}

	return &cobra.Command{
		Use:                   "version",
		Short:                 short,
		Long:                  ``,
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, _ []string) {
			hash, ts := getVersionHashAndTimestamp()

			if strings.TrimSpace(name) != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "%s version: %s from %s\n", name, hash, ts)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "version: %s from %s\n", hash, ts)
			}
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
// The information needs to be available to the `go build` command.
// `go run` and `go test` do not contain that info.
func readBuildInfo() (string, string, bool) {
	var (
		commitHash  string
		commitTS    string
		vcsModified bool
	)

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings { // called from a Go test info.Settings are always empty: []
			if setting.Key == "vcs.revision" {
				commitHash = setting.Value
			}

			if setting.Key == "vcs.time" {
				commitTS = setting.Value
			}

			// if true, the binary builds from uncommitted changes
			if setting.Key == "vcs.modified" {
				vcsModified = setting.Value == "true"
			}
		}
	}

	return commitHash, commitTS, vcsModified
}
