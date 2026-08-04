package application

import (
	"context"
	"errors"
	"testing"

	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	orgdomain "github.com/appkernia/appkernia/server/internal/modules/org/domain"
	"github.com/google/uuid"
)

type fakeAuthenticator struct {
	context iamdomain.AuthenticatedContext
}

func (fake fakeAuthenticator) Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error) {
	return fake.context, nil
}

type fakeRepository struct{ units []orgdomain.Unit }

func (fake fakeRepository) ListUnits(context.Context, uuid.UUID) ([]orgdomain.Unit, error) {
	return fake.units, nil
}
func (fake fakeRepository) CreateUnit(context.Context, orgdomain.Principal, orgdomain.UnitInput) (orgdomain.Unit, error) {
	return orgdomain.Unit{}, nil
}
func (fake fakeRepository) UpdateUnit(context.Context, orgdomain.Principal, uuid.UUID, orgdomain.UnitInput) (orgdomain.Unit, error) {
	return orgdomain.Unit{}, nil
}
func (fake fakeRepository) MoveUnit(context.Context, orgdomain.Principal, uuid.UUID, orgdomain.UnitMove) (orgdomain.Unit, error) {
	return orgdomain.Unit{}, nil
}
func (fake fakeRepository) DeleteUnit(context.Context, orgdomain.Principal, uuid.UUID) (orgdomain.UnitOccupancy, error) {
	return orgdomain.UnitOccupancy{}, nil
}
func (fake fakeRepository) ListPositions(context.Context, uuid.UUID, string, string, *uuid.UUID) ([]orgdomain.Position, error) {
	return nil, nil
}
func (fake fakeRepository) CreatePosition(context.Context, orgdomain.Principal, orgdomain.PositionInput) (orgdomain.Position, error) {
	return orgdomain.Position{}, nil
}
func (fake fakeRepository) UpdatePosition(context.Context, orgdomain.Principal, uuid.UUID, orgdomain.PositionInput) (orgdomain.Position, error) {
	return orgdomain.Position{}, nil
}
func (fake fakeRepository) DeletePosition(context.Context, orgdomain.Principal, uuid.UUID) (int64, error) {
	return 0, nil
}

func TestListUnitsRequiresPermissionAndPreservesMatchingAncestor(t *testing.T) {
	tenantID, rootID, childID := uuid.New(), uuid.New(), uuid.New()
	repository := fakeRepository{units: []orgdomain.Unit{{ID: rootID, Code: "ROOT", Name: "Company", Status: "active"}, {ID: childID, ParentID: &rootID, Code: "ENG", Name: "Engineering", Status: "active"}}}
	service := NewService(fakeAuthenticator{context: iamdomain.AuthenticatedContext{AuthContext: iamdomain.AuthContext{Tenant: iamdomain.Tenant{ID: tenantID}}}}, repository)
	if _, err := service.ListUnits(context.Background(), "token", "Engineering", "active"); !errors.Is(err, orgdomain.ErrForbidden) {
		t.Fatalf("ListUnits() without permission error = %v", err)
	}
	service = NewService(fakeAuthenticator{context: iamdomain.AuthenticatedContext{AuthContext: iamdomain.AuthContext{Tenant: iamdomain.Tenant{ID: tenantID}, Permissions: []string{"org.unit.read"}}}}, repository)
	units, err := service.ListUnits(context.Background(), "token", "Engineering", "active")
	if err != nil {
		t.Fatalf("ListUnits() error = %v", err)
	}
	if len(units) != 1 || units[0].ID != rootID || len(units[0].Children) != 1 || units[0].Children[0].ID != childID {
		t.Fatalf("filtered tree = %#v", units)
	}
}

func TestMoveUnitRejectsSelfBeforeRepository(t *testing.T) {
	id := uuid.New()
	service := NewService(fakeAuthenticator{context: iamdomain.AuthenticatedContext{AuthContext: iamdomain.AuthContext{Tenant: iamdomain.Tenant{ID: uuid.New()}, Permissions: []string{"org.unit.move"}}}}, fakeRepository{})
	if _, err := service.MoveUnit(context.Background(), "token", orgdomain.Principal{}, id, orgdomain.UnitMove{ParentID: &id}); !errors.Is(err, orgdomain.ErrUnitCycle) {
		t.Fatalf("MoveUnit(self) error = %v", err)
	}
}
