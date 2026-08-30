// Package jobdefs owns the compile-time execution policy for notification
// tasks. Queue producers and workers share these constants to prevent policy
// drift without exposing River to the application layer.
package jobdefs

import (
	"time"

	notify "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/domain"
	"github.com/appkernia/appkernia/server/internal/platform/jobqueue"
)

const (
	Queue               = "notifications"
	PublishMaxAttempts  = 5
	FanoutMaxAttempts   = 5
	DeliveryMaxAttempts = 5
	PublishTimeout      = 30 * time.Second
	FanoutTimeout       = 90 * time.Second
	DeliveryTimeout     = 90 * time.Second
)

func Definitions() []jobqueue.Definition {
	return []jobqueue.Definition{
		{
			Kind: notify.MessagePublishJobKind, Queue: Queue, MaxAttempts: PublishMaxAttempts, Timeout: PublishTimeout,
			RetryClasses: []string{jobqueue.RetryClassTransient},
		},
		{
			Kind: notify.PushFanoutJobKind, Queue: Queue, MaxAttempts: FanoutMaxAttempts, Timeout: FanoutTimeout,
			RetryClasses: []string{jobqueue.RetryClassTransient},
		},
		{
			Kind: notify.DeliveryJobKind, Queue: Queue, MaxAttempts: DeliveryMaxAttempts, Timeout: DeliveryTimeout,
			RetryClasses: []string{jobqueue.RetryClassThrottled, jobqueue.RetryClassTransient},
		},
	}
}

func Registry() *jobqueue.Registry {
	return jobqueue.MustRegistry(Definitions()...)
}
