package jobdefs

import (
	"time"

	appdomain "github.com/appkernia/appkernia/server/internal/modules/appmanagement/domain"
	"github.com/appkernia/appkernia/server/internal/platform/jobqueue"
)

const (
	Queue       = "privacy"
	MaxAttempts = 10
	Timeout     = 2 * time.Minute
)

func Definitions() []jobqueue.Definition {
	return []jobqueue.Definition{{
		Kind: appdomain.ObjectErasureJobKind, Queue: Queue, MaxAttempts: MaxAttempts, Timeout: Timeout,
		RetryClasses: []string{jobqueue.RetryClassTransient},
	}}
}
