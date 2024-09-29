package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/fatih/color" //nolint:misspell
	"github.com/spf13/cobra"

	"github.com/go-arrower/arrower/arrower/hooks"
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

//nolint:funlen // allow length because of init work
func newRunCmd(osSignal <-chan os.Signal, openBrowser internal.OpenBrowserFunc) *cobra.Command {
	return &cobra.Command{
		Use:                   "run",
		Short:                 "run and hot reload the application",
		Long:                  ``,
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, _ []string) {
			// log.Debug().Msg("start command `run`")

			blue := color.New(color.FgBlue, color.Bold).FprintfFunc()
			wg := sync.WaitGroup{}

			version, _ := getVersionHashAndTimestamp()
			blue(cmd.OutOrStdout(), "Arrower version %s\n", version)

			waitUntilShutdownFinished := make(chan struct{})

			config := &hooks.RunConfig{
				Port:      3030,
				WatchPath: ".",
			}

			hooks, err := hooks.Load(".config")
			if err != nil {
				panic(err)
			}

			if len(hooks) > 0 {
				blue(cmd.OutOrStdout(), "hooks loaded: %s\n", hooks.NamesFmt())
			}

			hooks.OnConfigLoaded(config)

			blue(cmd.OutOrStdout(), "watching %s\n", config.WatchPath)
			path, err := filepath.Abs(config.WatchPath)
			if err != nil {
				panic(err)
			}

			hooks.OnStart()

			hotReload := make(chan internal.File, 1)

			ctx, cancel := context.WithCancel(context.Background())
			wg.Add(1)
			go func(ctx context.Context, wg *sync.WaitGroup) {
				// log.Debug().Str("path", path).Msg("start to watch file system")

				//nolint:govet // shadowing err prevents a race condition
				err := internal.WatchBuildAndRunApp(ctx, cmd.OutOrStdout(), path, hooks, hotReload, openBrowser)
				if err != nil {
					panic(err)
				}

				wg.Done()
			}(ctx, &wg)

			hotReloadServer, err := internal.NewHotReloadServer(hotReload)
			if err != nil {
				panic(err)
			}

			wg.Add(1)
			go func() {
				// log.Debug().Msg("start hot reload server")

				_ = hotReloadServer.Start(fmt.Sprintf(":%d", config.Port))

				wg.Done()
			}()

			// log.Debug().Msg("Waiting for shutdown")

			go func() {
				<-osSignal
				// log.Debug().Msg("Shutdown signal received")

				hooks.OnShutdown()

				cancel()
				err = hotReloadServer.Shutdown(context.Background())
				if err != nil {
					panic(err)
				}

				wg.Wait()

				close(waitUntilShutdownFinished)
			}()

			<-waitUntilShutdownFinished
			blue(cmd.OutOrStdout(), "done\n")
		},
	}
}
