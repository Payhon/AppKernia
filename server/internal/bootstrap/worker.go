package bootstrap

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	appmanagementjobdefs "github.com/appkernia/appkernia/server/internal/modules/appmanagement/jobdefs"
	appmanagementworker "github.com/appkernia/appkernia/server/internal/modules/appmanagement/worker"
	feedbackworker "github.com/appkernia/appkernia/server/internal/modules/feedback/worker"
	jobworker "github.com/appkernia/appkernia/server/internal/modules/jobadmin/worker"
	notificationjobdefs "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/jobdefs"
	notificationworker "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/worker"
	pushdomain "github.com/appkernia/appkernia/server/internal/modules/push/domain"
	pushprovider "github.com/appkernia/appkernia/server/internal/modules/push/provider"
	storagedomain "github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	storagerepo "github.com/appkernia/appkernia/server/internal/modules/storage/repository"
	settingsrepo "github.com/appkernia/appkernia/server/internal/modules/systemsettings/repository"
	"github.com/appkernia/appkernia/server/internal/platform/config"
	"github.com/appkernia/appkernia/server/internal/platform/jobqueue"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

// Worker owns the background River client and its companion maintenance
// loops. Both ak-worker and akone use this lifecycle so worker registration
// cannot drift between binaries.
type Worker struct {
	pool        *pgxpool.Pool
	client      *river.Client[pgx.Tx]
	start       func(context.Context, chan<- struct{})
	cancel      context.CancelFunc
	cleanupDone chan struct{}
	started     bool
	closeOnce   sync.Once
}

func NewWorker(ctx context.Context, cfg config.Config) (*Worker, error) {
	if cfg.DatabaseDriver != config.DatabaseDriverPostgreSQL {
		return nil, errors.New("the standalone worker requires database.driver=postgresql; SQLite jobs are disabled")
	}
	if err := jobworker.ValidateRegistry(); err != nil {
		return nil, err
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}
	closeOnError := func(err error) (*Worker, error) {
		pool.Close()
		return nil, err
	}
	masterKey, err := base64.StdEncoding.DecodeString(cfg.ConfigMasterKeyBase64)
	if err != nil {
		return closeOnError(fmt.Errorf("decode configuration master key: %w", err))
	}
	sealer, err := settingsrepo.NewAESGCMSealer(masterKey, cfg.ConfigMasterKeyVersion)
	if err != nil {
		return closeOnError(fmt.Errorf("create configuration secret sealer: %w", err))
	}
	insertClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{SkipUnknownJobCheck: true})
	if err != nil {
		return closeOnError(fmt.Errorf("create River insert client: %w", err))
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
			return closeOnError(fmt.Errorf("create object storage adapter: %w", err))
		}
	}
	var pushSender pushdomain.Sender
	if cfg.PushAdapter == "local-mock" {
		pushSender = pushprovider.NewMockSender()
	} else {
		pushSender = pushprovider.NewOfficialSender(pool, sealer, cfg.Environment)
	}
	workers := river.NewWorkers()
	river.AddWorker(workers, jobworker.NewRunWorker(pool))
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
		return closeOnError(fmt.Errorf("create River client: %w", err))
	}
	return &Worker{
		pool: pool, client: client, cleanupDone: make(chan struct{}),
		start: func(runCtx context.Context, cleanupDone chan<- struct{}) {
			go jobworker.NewScheduler(pool, client, 15*time.Second).Run(runCtx)
			go notificationworker.NewOperationsMaintenance(pool, cfg.Environment, time.Hour).Run(runCtx)
			go func() {
				defer close(cleanupDone)
				feedbackworker.NewCleanup(pool, objectStore).Run(runCtx)
			}()
		},
	}, nil
}

func (worker *Worker) Start(ctx context.Context) error {
	if worker.cancel != nil {
		return errors.New("worker already started")
	}
	runCtx, cancel := context.WithCancel(ctx)
	worker.cancel = cancel
	if err := worker.client.Start(runCtx); err != nil {
		cancel()
		close(worker.cleanupDone)
		return fmt.Errorf("start River client: %w", err)
	}
	worker.started = true
	startLoops := worker.start
	worker.start = func(context.Context, chan<- struct{}) {}
	startLoops(runCtx, worker.cleanupDone)
	return nil
}

func (worker *Worker) Shutdown(ctx context.Context) error {
	var stopErr error
	worker.closeOnce.Do(func() {
		if worker.cancel != nil {
			worker.cancel()
		}
		if worker.cancel != nil {
			select {
			case <-worker.cleanupDone:
			case <-ctx.Done():
				stopErr = ctx.Err()
			}
		}
		if worker.started {
			if err := worker.client.Stop(ctx); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				stopErr = errors.Join(stopErr, fmt.Errorf("stop River client: %w", err))
			}
		}
		worker.pool.Close()
	})
	return stopErr
}
