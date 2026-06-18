//go:build e2e

// Package main is the e2e system-under-test: a minimal Arrower application that mounts
// the auth and admin contexts,
// so the e2e client suites (contexts/{auth,admin}/tests/...) can drive them.
package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-arrower/arrower"
	admininit "github.com/go-arrower/arrower/contexts/admin/init"
	authinit "github.com/go-arrower/arrower/contexts/auth/init"
	"github.com/go-arrower/arrower/tests/e2e/app/views"
)

const shutdownTimeout = 5 * time.Second

//nolint:wsl_v5
func main() {
	cfgFile := flag.String("config", "", "path to the test config yaml")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	cfg, err := loadConfig(*cfgFile)
	if err != nil {
		panic("config: " + err.Error())
	}

	// nil migrations -> ArrowerDefaultMigrations
	dc, err := arrower.InitialiseDefaultDependencies(ctx, &cfg, nil, embed.FS{}, views.SharedViews, nil)
	if err != nil {
		panic("initialise dependencies: " + err.Error())
	}

	authContext, err := authinit.NewAuthContext(ctx, dc)
	if err != nil {
		panic("init auth context: " + err.Error())
	}

	adminContext, err := admininit.NewAdminContext(ctx, dc)
	if err != nil {
		panic("init admin context: " + err.Error())
	}

	if err := dc.Start(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic("start server: " + err.Error())
	}

	<-ctx.Done()
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	err = adminContext.Shutdown(shutdownCtx)
	if err != nil {
		panic("shutdown admin context: " + err.Error())
	}

	err = authContext.Shutdown(shutdownCtx)
	if err != nil {
		panic("shutdown auth context: " + err.Error())
	}

	err = dc.Shutdown(shutdownCtx)
	if err != nil {
		panic("shutdown error: " + err.Error())
	}
}

func loadConfig(cfgFile string) (arrower.Config, error) {
	vip := arrower.DefaultViper()
	if cfgFile != "" {
		vip.SetConfigFile(cfgFile)
	}

	if err := vip.ReadInConfig(); err != nil {
		return arrower.Config{}, fmt.Errorf("could not read config %q: %w", cfgFile, err)
	}

	cfg := arrower.Config{}
	if err := vip.Unmarshal(&cfg); err != nil {
		return arrower.Config{}, fmt.Errorf("could not unmarshal into configuration: %w", err)
	}

	return cfg, nil
}
