package repository

import (
	"context"
	"sync"

	"github.com/appkernia/appkernia/server/internal/modules/iam/application"
)

// LocalPasswordResetNotifier is a development-only in-memory adapter. It never
// logs or persists recovery tokens and is not wired outside the development
// environment. Production delivery adapters are intentionally feature-gated.
type LocalPasswordResetNotifier struct {
	mu         sync.Mutex
	deliveries []application.PasswordResetNotification
}

func NewLocalPasswordResetNotifier() *LocalPasswordResetNotifier {
	return &LocalPasswordResetNotifier{}
}

func (notifier *LocalPasswordResetNotifier) SendPasswordReset(
	_ context.Context,
	notification application.PasswordResetNotification,
) error {
	notifier.mu.Lock()
	defer notifier.mu.Unlock()
	notifier.deliveries = append(notifier.deliveries, notification)
	if len(notifier.deliveries) > 20 {
		notifier.deliveries = append([]application.PasswordResetNotification(nil), notifier.deliveries[len(notifier.deliveries)-20:]...)
	}
	return nil
}
