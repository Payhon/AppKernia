package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/appkernia/appkernia/server/internal/bootstrap"
	"github.com/appkernia/appkernia/server/internal/platform/buildinfo"
	"github.com/appkernia/appkernia/server/internal/platform/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "ak-api:", err)
		os.Exit(1)
	}
}

func run() error {
	fmt.Printf("ak-api version=%s commit=%s\n", buildinfo.Version, buildinfo.Commit)
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	app, err := bootstrap.NewAPI(context.Background(), cfg)
	if err != nil {
		return err
	}
	if err := app.Start(); err != nil {
		return err
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals
	return app.Shutdown()
}
