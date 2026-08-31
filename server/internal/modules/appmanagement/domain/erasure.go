package domain

import "github.com/google/uuid"

const ObjectErasureJobKind = "account-object-erasure"

type ObjectErasureJobArgs struct {
	ObjectID uuid.UUID `json:"object_id"`
}

func (ObjectErasureJobArgs) Kind() string { return ObjectErasureJobKind }
