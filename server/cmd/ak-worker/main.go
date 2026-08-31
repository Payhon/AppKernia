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

	appmanagementjobdefs "github.com/appkernia/appkernia/server/internal/modules/appmanagement/jobdefs"
	appmanagementworker "github.com/appkernia/appkernia/server/internal/modules/appmanagement/worker"
	jobworker "github.com/appkernia/appkernia/server/internal/modules/jobadmin/worker"
	notificationjobdefs "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/jobdefs"
	notificationworker "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/worker"
	pushdomain "github.com/appkernia/appkernia/server/internal/modules/push/domain"
	pushprovider "github.com/appkernia/appkernia/server/internal/modules/push/provider"
	storagedomain "github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	storagerepo "github.com/appkernia/appkernia/server/internal/modules/storage/repository"
	settingsrepo "github.com/appkernia/appkernia/server/internal/modules/systemsettings/repository"
	"github.com/appkernia/appkernia/server/internal/platform/buildinfo"
	"github.com/appkernia/appkernia/server/internal/platform/config"
	"github.com/appkernia/appkernia/server/internal/platform/jobqueue"
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
	insertClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{SkipUnknownJobCheck: true})
	if err != nil {
		return fmt.Errorf("create River insert client: %w", err)
	}
	queueDefinitions := append(notificationjobdefs.Definitions(), appmanagementjobdefs.Definitions()...)
	trackedQueue := jobqueue.NewRiverAdapter(pool, insertClient, jobqueue.MustRegistry(queueDefinitions...))
	var objectStore storagedomain.ObjectStore
	if cfg.AvatarUploadEnabled || cfg.FileStorageEnabled {
		switch cfg.ObjectStorageAdapter {
		case "local":
			objectStore, err = storagerepo.NewLocalObjectStore(cfg.LocalObjectStorageDir)
		case "configured":
			objectStore, err = storagerepo.NewConfiguredObjectStore(pool, sealer, cfg.LocalObjectStorageDir, cfg.Environment)
		}
		if err != nil {
			return fmt.Errorf("create object storage adapter: %w", err)
		}
	}
	var pushSender pushdomain.Sender
	if cfg.PushAdapter == "local-mock" {
		pushSender = pushprovider.NewMockSender()
	} else {
		pushSender = pushprovider.NewOfficialSender(pool, sealer, cfg.Environment)
	}
	river.AddWorker(workers, notificationworker.NewDeliveryWorker(pool, sealer, cfg.Environment, cfg.PushEnabled, pushSender))
	river.AddWorker(workers, notificationworker.NewMessagePublishWorker(pool, trackedQueue))
	river.AddWorker(workers, notificationworker.NewPushFanoutWorker(pool, trackedQueue, sealer, cfg.Environment, cfg.PushEnabled))
	river.AddWorker(workers, appmanagementworker.NewObjectErasureWorker(pool, objectStore))
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues:          map[string]river.QueueConfig{"default": {MaxWorkers: 4}, "notifications": {MaxWorkers: 8}, "privacy": {MaxWorkers: 2}},
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
	maintenance := notificationworker.NewOperationsMaintenance(pool, cfg.Environment, time.Hour)
	go maintenance.Run(ctx)
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err = client.Stop(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("stop River client: %w", err)
	}
	return nil
}
