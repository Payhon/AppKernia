package application

import (
	"context"
	"strings"

	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	orgdomain "github.com/appkernia/appkernia/server/internal/modules/org/domain"
	"github.com/google/uuid"
)

const adminAudience = "ak-admin"

type Authenticator interface {
	Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error)
}

type Service struct {
	authenticator Authenticator
	repository    orgdomain.Repository
}

func NewService(authenticator Authenticator, repository orgdomain.Repository) *Service {
	return &Service{authenticator: authenticator, repository: repository}
}

func (service *Service) ListUnits(ctx context.Context, token, query, status string) ([]orgdomain.Unit, error) {
	auth, err := service.authorize(ctx, token, "org.unit.read")
	if err != nil {
		return nil, err
	}
	if status != "" && status != "active" && status != "disabled" {
		return nil, orgdomain.ErrInvalid
	}
	units, err := service.repository.ListUnits(ctx, auth.Tenant.ID)
	if err != nil {
		return nil, err
	}
	return buildFilteredTree(units, strings.TrimSpace(query), status), nil
}

func (service *Service) CreateUnit(ctx context.Context, token string, principal orgdomain.Principal, input orgdomain.UnitInput) (orgdomain.Unit, error) {
	auth, err := service.authorize(ctx, token, "org.unit.create")
	if err != nil {
		return orgdomain.Unit{}, err
	}
	if !validUnit(input) {
		return orgdomain.Unit{}, orgdomain.ErrInvalid
	}
	principal = scopedPrincipal(auth, principal)
	return service.repository.CreateUnit(ctx, principal, normalizeUnit(input))
}

func (service *Service) UpdateUnit(ctx context.Context, token string, principal orgdomain.Principal, id uuid.UUID, input orgdomain.UnitInput) (orgdomain.Unit, error) {
	auth, err := service.authorize(ctx, token, "org.unit.update")
	if err != nil {
		return orgdomain.Unit{}, err
	}
	if id == uuid.Nil || !validUnit(input) {
		return orgdomain.Unit{}, orgdomain.ErrInvalid
	}
	principal = scopedPrincipal(auth, principal)
	return service.repository.UpdateUnit(ctx, principal, id, normalizeUnit(input))
}

func (service *Service) MoveUnit(ctx context.Context, token string, principal orgdomain.Principal, id uuid.UUID, input orgdomain.UnitMove) (orgdomain.Unit, error) {
	auth, err := service.authorize(ctx, token, "org.unit.move")
	if err != nil {
		return orgdomain.Unit{}, err
	}
	if id == uuid.Nil || (input.ParentID != nil && *input.ParentID == id) {
		return orgdomain.Unit{}, orgdomain.ErrUnitCycle
	}
	principal = scopedPrincipal(auth, principal)
	return service.repository.MoveUnit(ctx, principal, id, input)
}

func (service *Service) DeleteUnit(ctx context.Context, token string, principal orgdomain.Principal, id uuid.UUID) (orgdomain.UnitOccupancy, error) {
	auth, err := service.authorize(ctx, token, "org.unit.delete")
	if err != nil {
		return orgdomain.UnitOccupancy{}, err
	}
	if id == uuid.Nil {
		return orgdomain.UnitOccupancy{}, orgdomain.ErrInvalid
	}
	principal = scopedPrincipal(auth, principal)
	return service.repository.DeleteUnit(ctx, principal, id)
}

func (service *Service) ListPositions(ctx context.Context, token, query, status string, unitID *uuid.UUID) ([]orgdomain.Position, error) {
	auth, err := service.authorize(ctx, token, "org.position.read")
	if err != nil {
		return nil, err
	}
	if status != "" && status != "active" && status != "disabled" {
		return nil, orgdomain.ErrInvalid
	}
	return service.repository.ListPositions(ctx, auth.Tenant.ID, strings.TrimSpace(query), status, unitID)
}

func (service *Service) CreatePosition(ctx context.Context, token string, principal orgdomain.Principal, input orgdomain.PositionInput) (orgdomain.Position, error) {
	auth, err := service.authorize(ctx, token, "org.position.create")
	if err != nil {
		return orgdomain.Position{}, err
	}
	if !validPosition(input) {
		return orgdomain.Position{}, orgdomain.ErrInvalid
	}
	principal = scopedPrincipal(auth, principal)
	return service.repository.CreatePosition(ctx, principal, normalizePosition(input))
}

func (service *Service) UpdatePosition(ctx context.Context, token string, principal orgdomain.Principal, id uuid.UUID, input orgdomain.PositionInput) (orgdomain.Position, error) {
	auth, err := service.authorize(ctx, token, "org.position.update")
	if err != nil {
		return orgdomain.Position{}, err
	}
	if id == uuid.Nil || !validPosition(input) {
		return orgdomain.Position{}, orgdomain.ErrInvalid
	}
	principal = scopedPrincipal(auth, principal)
	return service.repository.UpdatePosition(ctx, principal, id, normalizePosition(input))
}

func (service *Service) DeletePosition(ctx context.Context, token string, principal orgdomain.Principal, id uuid.UUID) (int64, error) {
	auth, err := service.authorize(ctx, token, "org.position.delete")
	if err != nil {
		return 0, err
	}
	if id == uuid.Nil {
		return 0, orgdomain.ErrInvalid
	}
	principal = scopedPrincipal(auth, principal)
	return service.repository.DeletePosition(ctx, principal, id)
}

func (service *Service) authorize(ctx context.Context, token, permission string) (iamdomain.AuthenticatedContext, error) {
	auth, err := service.authenticator.Authenticate(ctx, token, adminAudience)
	if err != nil {
		return iamdomain.AuthenticatedContext{}, err
	}
	for _, current := range auth.Permissions {
		if current == permission {
			return auth, nil
		}
	}
	return iamdomain.AuthenticatedContext{}, orgdomain.ErrForbidden
}

func scopedPrincipal(auth iamdomain.AuthenticatedContext, client orgdomain.Principal) orgdomain.Principal {
	client.TenantID, client.UserID, client.SessionID = auth.Tenant.ID, auth.User.ID, auth.SessionID
	return client
}

func validUnit(input orgdomain.UnitInput) bool {
	return len(strings.TrimSpace(input.Code)) >= 2 && len(strings.TrimSpace(input.Code)) <= 64 && len(strings.TrimSpace(input.Name)) >= 1 && len(strings.TrimSpace(input.Name)) <= 160 &&
		contains([]string{"company", "division", "department", "team", "group"}, input.UnitType) && contains([]string{"active", "disabled"}, input.Status)
}
func normalizeUnit(input orgdomain.UnitInput) orgdomain.UnitInput {
	input.Code = strings.TrimSpace(input.Code)
	input.Name = strings.TrimSpace(input.Name)
	input.Phone = strings.TrimSpace(input.Phone)
	input.Email = strings.TrimSpace(input.Email)
	return input
}
func validPosition(input orgdomain.PositionInput) bool {
	return len(strings.TrimSpace(input.Code)) >= 2 && len(strings.TrimSpace(input.Code)) <= 64 && len(strings.TrimSpace(input.Name)) >= 1 && len(strings.TrimSpace(input.Name)) <= 120 && len(strings.TrimSpace(input.Description)) <= 500 && contains([]string{"active", "disabled"}, input.Status)
}
func normalizePosition(input orgdomain.PositionInput) orgdomain.PositionInput {
	input.Code = strings.TrimSpace(input.Code)
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	return input
}
func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func buildFilteredTree(units []orgdomain.Unit, query, status string) []orgdomain.Unit {
	query = strings.ToLower(query)
	byID := make(map[uuid.UUID]orgdomain.Unit, len(units))
	keep := make(map[uuid.UUID]bool, len(units))
	for _, unit := range units {
		byID[unit.ID] = unit
	}
	for _, unit := range units {
		matches := (status == "" || unit.Status == status) && (query == "" || strings.Contains(strings.ToLower(unit.Name), query) || strings.Contains(strings.ToLower(unit.Code), query))
		if !matches {
			continue
		}
		for current := &unit; current != nil; {
			keep[current.ID] = true
			if current.ParentID == nil {
				break
			}
			parent, ok := byID[*current.ParentID]
			if !ok {
				break
			}
			current = &parent
		}
	}
	children := make(map[uuid.UUID][]orgdomain.Unit)
	roots := []orgdomain.Unit{}
	for _, unit := range units {
		if !keep[unit.ID] {
			continue
		}
		unit.Children = []orgdomain.Unit{}
		if unit.ParentID == nil || !keep[*unit.ParentID] {
			roots = append(roots, unit)
		} else {
			children[*unit.ParentID] = append(children[*unit.ParentID], unit)
		}
	}
	var attach func(orgdomain.Unit) orgdomain.Unit
	attach = func(unit orgdomain.Unit) orgdomain.Unit {
		for _, child := range children[unit.ID] {
			unit.Children = append(unit.Children, attach(child))
		}
		return unit
	}
	for index := range roots {
		roots[index] = attach(roots[index])
	}
	return roots
}
