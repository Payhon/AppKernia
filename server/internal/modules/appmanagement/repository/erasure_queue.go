package repository

import (
	"context"
	"fmt"

	appdomain "github.com/appkernia/appkernia/server/internal/modules/appmanagement/domain"
	"github.com/appkernia/appkernia/server/internal/modules/appmanagement/jobdefs"
	"github.com/appkernia/appkernia/server/internal/platform/jobqueue"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ObjectErasureQueue struct{ queue jobqueue.Enqueuer }

func NewObjectErasureQueue(queue jobqueue.Enqueuer) *ObjectErasureQueue {
	return &ObjectErasureQueue{queue: queue}
}

func (q *ObjectErasureQueue) Enqueue(ctx context.Context, tx pgx.Tx, tenantID, appID, eventID, objectID uuid.UUID) error {
	if q == nil || q.queue == nil || tx == nil || tenantID == uuid.Nil || appID == uuid.Nil || eventID == uuid.Nil || objectID == uuid.Nil {
		return fmt.Errorf("account object erasure queue is unavailable")
	}
	_, err := q.queue.EnqueueTx(ctx, tx, jobqueue.Spec{
		Scope: jobqueue.Scope{TenantID: tenantID, AppID: &appID, ModuleCode: "iam", ResourceType: "privacy_erasure_object", ResourceID: &objectID, CorrelationID: &eventID},
		Args:  appdomain.ObjectErasureJobArgs{ObjectID: objectID}, Queue: jobdefs.Queue, MaxAttempts: jobdefs.MaxAttempts, UniqueByArgs: true,
	})
	if err != nil {
		return fmt.Errorf("enqueue account object erasure: %w", err)
	}
	return nil
}
