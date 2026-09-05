package main

import (
	"context"
	"errors"
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
		fmt.Fprintln(os.Stderr, "ak-worker:", err)
		os.Exit(1)
	}
}

func run() error {
	fmt.Printf("ak-worker version=%s commit=%s\n", buildinfo.Version, buildinfo.Commit)
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	worker, err := bootstrap.NewWorker(ctx, cfg)
	if err != nil {
		return err
	}
	if err = worker.Start(ctx); err != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		return errors.Join(err, worker.Shutdown(shutdownCtx))
	}
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	return worker.Shutdown(shutdownCtx)
}
