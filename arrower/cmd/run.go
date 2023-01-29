package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/go-arrower/arrower/arrower/internal"
)

// NewInterruptSignalChannel returns a channel listening for os.Signals the arrower cli will react to.
func NewInterruptSignalChannel() chan os.Signal {
	signalsToListenTo := []os.Signal{
		syscall.SIGINT,                   // Strg + c
		syscall.SIGTERM, syscall.SIGQUIT, // terminate but finish/cleanup first, e.g. kill
		os.Interrupt,
	}

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, signalsToListenTo...)

	return osSignal
}

func newRunCmd(osSignal <-chan os.Signal) *cobra.Command {
	return &cobra.Command{
		Use:                   "run",
		Short:                 "run and hot reload the application",
		Long:                  ``,
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), "Run arrower")

			waitUntilShutdownFinished := make(chan struct{})

			ctx, cancel := context.WithCancel(context.Background())
			go func(ctx context.Context) {
				err := internal.WatchBuildAndRunApp(ctx, ".")
				if err != nil {
					panic(err)
				}
			}(ctx)

			fmt.Fprintln(cmd.OutOrStdout(), "Waiting for shutdown")

			go func(cmd *cobra.Command) {
				<-osSignal
				fmt.Fprintln(cmd.OutOrStdout(), "Shutdown signal received")

				cancel()
				close(waitUntilShutdownFinished)
			}(cmd)

			<-waitUntilShutdownFinished
			fmt.Fprintln(cmd.OutOrStdout(), "Done")
		},
	}
}
