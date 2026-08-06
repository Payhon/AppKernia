package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	jobworker "github.com/appkernia/appkernia/server/internal/modules/jobadmin/worker"
	notificationworker "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/worker"
	settingsrepo "github.com/appkernia/appkernia/server/internal/modules/systemsettings/repository"
	"github.com/appkernia/appkernia/server/internal/platform/buildinfo"
	"github.com/appkernia/appkernia/server/internal/platform/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "ak-worker:", err)
		os.Exit(1)
	}
}

func run() error {
	fmt.Printf("ak-worker version=%s commit=%s\n", buildinfo.Version, buildinfo.Commit)
	if err := jobworker.ValidateRegistry(); err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create PostgreSQL pool: %w", err)
	}
	defer pool.Close()
	workers := river.NewWorkers()
	river.AddWorker(workers, jobworker.NewRunWorker(pool))
	masterKey, err := base64.StdEncoding.DecodeString(cfg.ConfigMasterKeyBase64)
	if err != nil {
		return fmt.Errorf("decode configuration master key: %w", err)
	}
	sealer, err := settingsrepo.NewAESGCMSealer(masterKey, cfg.ConfigMasterKeyVersion)
	if err != nil {
		return fmt.Errorf("create configuration secret sealer: %w", err)
	}
	river.AddWorker(workers, notificationworker.NewDeliveryWorker(pool, sealer, cfg.Environment))
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues:          map[string]river.QueueConfig{"default": {MaxWorkers: 4}, "notifications": {MaxWorkers: 8}},
		Workers:         workers,
		JobTimeout:      24 * time.Hour,
		SoftStopTimeout: cfg.ShutdownTimeout,
	})
	if err != nil {
		return fmt.Errorf("create River client: %w", err)
	}
	if err = client.Start(ctx); err != nil {
		return fmt.Errorf("start River client: %w", err)
	}
	scheduler := jobworker.NewScheduler(pool, client, 15*time.Second)
	go scheduler.Run(ctx)
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err = client.Stop(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("stop River client: %w", err)
	}
	return nil
}
