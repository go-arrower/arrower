package cmd

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/fatih/color" //nolint:misspell
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
			blue := color.New(color.FgBlue, color.Bold).FprintfFunc()
			wg := sync.WaitGroup{}

			version, _ := getVersionHashAndTimestamp()
			blue(cmd.OutOrStdout(), "Arrower version %s\n", version)

			waitUntilShutdownFinished := make(chan struct{})

			path := "."
			blue(cmd.OutOrStdout(), "watching %s\n", path)
			path, err := filepath.Abs(path)
			if err != nil {
				panic(err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			wg.Add(1)
			go func(ctx context.Context, wg *sync.WaitGroup) {
				err := internal.WatchBuildAndRunApp(ctx, path)
				if err != nil {
					panic(err)
				}

				wg.Done()
			}(ctx, &wg)

			// fmt.Fprintln(cmd.OutOrStdout(), "Waiting for shutdown")

			go func() {
				<-osSignal
				// fmt.Fprintln(cmd.OutOrStdout(), "Shutdown signal received")

				cancel()
				wg.Wait()

				close(waitUntilShutdownFinished)
			}()

			<-waitUntilShutdownFinished
			blue(cmd.OutOrStdout(), "done\n")
		},
	}
}
