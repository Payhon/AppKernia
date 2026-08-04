package application

import (
	"context"
	"regexp"
	"strings"

	accessdomain "github.com/appkernia/appkernia/server/internal/modules/accessadmin/domain"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/google/uuid"
)

const adminAudience = "ak-admin"

var codePattern = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,127}$`)

type Authenticator interface {
	Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error)
}

type Service struct {
	auth       Authenticator
	repository accessdomain.Repository
}

func NewService(auth Authenticator, repository accessdomain.Repository) *Service {
	return &Service{auth: auth, repository: repository}
}

func (s *Service) authorize(ctx context.Context, token, permission string) (iamdomain.AuthenticatedContext, error) {
	a, err := s.auth.Authenticate(ctx, token, adminAudience)
	if err != nil {
		return iamdomain.AuthenticatedContext{}, err
	}
	for _, candidate := range a.Permissions {
		if candidate == permission {
			return a, nil
		}
	}
	return iamdomain.AuthenticatedContext{}, accessdomain.ErrForbidden
}

func scoped(a iamdomain.AuthenticatedContext, p accessdomain.Principal) accessdomain.Principal {
	p.TenantID, p.UserID, p.SessionID = a.Tenant.ID, a.User.ID, a.SessionID
	return p
}

func (s *Service) Roles(ctx context.Context, token string, f accessdomain.Filters) (accessdomain.RolePage, error) {
	a, err := s.authorize(ctx, token, "iam.role.read")
	if err != nil {
		return accessdomain.RolePage{}, err
	}
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PageSize == 0 {
		f.PageSize = 20
	}
	if f.Page < 1 || f.PageSize < 1 || f.PageSize > 100 || !oneOf(f.Status, "", "active", "disabled") || !oneOf(f.RoleType, "", "system", "custom") {
		return accessdomain.RolePage{}, accessdomain.ErrInvalid
	}
	f.Query = strings.TrimSpace(f.Query)
	return s.repository.ListRoles(ctx, a.Tenant.ID, f)
}

func normalizeRole(in accessdomain.RoleInput) accessdomain.RoleInput {
	in.Code = strings.ToLower(strings.TrimSpace(in.Code))
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	return in
}

func validRole(in accessdomain.RoleInput) bool {
	return codePattern.MatchString(in.Code) && len(in.Name) > 0 && len(in.Name) <= 120 && len(in.Description) <= 500 && oneOf(in.Status, "active", "disabled") && (in.ParentID == nil || *in.ParentID != uuid.Nil)
}

func (s *Service) CreateRole(ctx context.Context, token string, p accessdomain.Principal, in accessdomain.RoleInput) (accessdomain.Role, error) {
	a, err := s.authorize(ctx, token, "iam.role.create")
	if err != nil {
		return accessdomain.Role{}, err
	}
	in = normalizeRole(in)
	if !validRole(in) {
		return accessdomain.Role{}, accessdomain.ErrInvalid
	}
	return s.repository.CreateRole(ctx, scoped(a, p), in)
}

func (s *Service) UpdateRole(ctx context.Context, token string, p accessdomain.Principal, id uuid.UUID, in accessdomain.RoleInput) (accessdomain.Role, error) {
	a, err := s.authorize(ctx, token, "iam.role.update")
	if err != nil {
		return accessdomain.Role{}, err
	}
	in = normalizeRole(in)
	if id == uuid.Nil || !validRole(in) {
		return accessdomain.Role{}, accessdomain.ErrInvalid
	}
	return s.repository.UpdateRole(ctx, scoped(a, p), id, in)
}

func (s *Service) DeleteRole(ctx context.Context, token string, p accessdomain.Principal, id uuid.UUID) error {
	a, err := s.authorize(ctx, token, "iam.role.delete")
	if err != nil {
		return err
	}
	if id == uuid.Nil {
		return accessdomain.ErrInvalid
	}
	return s.repository.DeleteRole(ctx, scoped(a, p), id)
}

func (s *Service) ReplacePermissions(ctx context.Context, token string, p accessdomain.Principal, id uuid.UUID, ids []uuid.UUID) (accessdomain.Role, error) {
	a, err := s.authorize(ctx, token, "iam.role.assign_permission")
	if err != nil {
		return accessdomain.Role{}, err
	}
	if id == uuid.Nil || !unique(ids) || len(ids) > 500 {
		return accessdomain.Role{}, accessdomain.ErrInvalid
	}
	return s.repository.ReplaceRolePermissions(ctx, scoped(a, p), id, ids)
}

func (s *Service) ReplaceMenus(ctx context.Context, token string, p accessdomain.Principal, id uuid.UUID, ids []uuid.UUID) (accessdomain.Role, error) {
	a, err := s.authorize(ctx, token, "iam.role.assign_menu")
	if err != nil {
		return accessdomain.Role{}, err
	}
	if id == uuid.Nil || !unique(ids) || len(ids) > 500 {
		return accessdomain.Role{}, accessdomain.ErrInvalid
	}
	return s.repository.ReplaceRoleMenus(ctx, scoped(a, p), id, ids)
}

func (s *Service) ReplaceDataScope(ctx context.Context, token string, p accessdomain.Principal, id uuid.UUID, dataScope string, ids []uuid.UUID) (accessdomain.Role, error) {
	a, err := s.authorize(ctx, token, "iam.role.update_data_scope")
	if err != nil {
		return accessdomain.Role{}, err
	}
	if id == uuid.Nil || !oneOf(dataScope, "all", "tenant", "department", "department_tree", "self", "custom") || !unique(ids) || len(ids) > 500 || (dataScope == "custom" && len(ids) == 0) || (dataScope != "custom" && len(ids) != 0) {
		return accessdomain.Role{}, accessdomain.ErrInvalid
	}
	return s.repository.ReplaceRoleDataScope(ctx, scoped(a, p), id, dataScope, ids)
}

func (s *Service) Permissions(ctx context.Context, token string, f accessdomain.PermissionFilters) ([]accessdomain.Permission, error) {
	_, err := s.authorize(ctx, token, "iam.permission.read")
	if err != nil {
		return nil, err
	}
	if !oneOf(f.PermissionKind, "", "api", "ui_action", "feature") || !oneOf(f.Status, "", "active", "disabled") {
		return nil, accessdomain.ErrInvalid
	}
	f.Query, f.ModuleCode, f.ResourceName, f.ActionName = strings.TrimSpace(f.Query), strings.TrimSpace(f.ModuleCode), strings.TrimSpace(f.ResourceName), strings.TrimSpace(f.ActionName)
	return s.repository.ListPermissions(ctx, f)
}

func (s *Service) Menus(ctx context.Context, token string) ([]accessdomain.Menu, error) {
	a, err := s.authorize(ctx, token, "sys.menu.read")
	if err != nil {
		return nil, err
	}
	return s.repository.ListMenus(ctx, a.Tenant.ID)
}

func normalizeMenu(in accessdomain.MenuInput) accessdomain.MenuInput {
	in.Code, in.Title, in.I18nKey = strings.ToLower(strings.TrimSpace(in.Code)), strings.TrimSpace(in.Title), strings.TrimSpace(in.I18nKey)
	in.Path, in.ComponentKey, in.Icon, in.ExternalURL, in.OpenMode = strings.TrimSpace(in.Path), strings.TrimSpace(in.ComponentKey), strings.TrimSpace(in.Icon), strings.TrimSpace(in.ExternalURL), strings.TrimSpace(in.OpenMode)
	return in
}

func validMenu(in accessdomain.MenuInput) bool {
	if !codePattern.MatchString(in.Code) || len(in.Title) == 0 || len(in.Title) > 160 || len(in.I18nKey) == 0 || len(in.I18nKey) > 200 || !oneOf(in.Type, "directory", "page", "external") || !oneOf(in.OpenMode, "same_tab", "new_tab", "iframe") || !oneOf(in.Status, "active", "disabled") {
		return false
	}
	if in.Type == "page" {
		return strings.HasPrefix(in.Path, "/") && in.ComponentKey != "" && in.ExternalURL == ""
	}
	if in.Type == "external" {
		return (strings.HasPrefix(in.ExternalURL, "https://") || strings.HasPrefix(in.ExternalURL, "http://")) && in.ComponentKey == ""
	}
	return in.ComponentKey == "" && in.ExternalURL == ""
}

func (s *Service) CreateMenu(ctx context.Context, token string, p accessdomain.Principal, in accessdomain.MenuInput) (accessdomain.Menu, error) {
	a, err := s.authorize(ctx, token, "sys.menu.create")
	if err != nil {
		return accessdomain.Menu{}, err
	}
	in = normalizeMenu(in)
	if !validMenu(in) {
		return accessdomain.Menu{}, accessdomain.ErrInvalid
	}
	return s.repository.CreateMenu(ctx, scoped(a, p), in)
}

func (s *Service) UpdateMenu(ctx context.Context, token string, p accessdomain.Principal, id uuid.UUID, in accessdomain.MenuInput) (accessdomain.Menu, error) {
	a, err := s.authorize(ctx, token, "sys.menu.update")
	if err != nil {
		return accessdomain.Menu{}, err
	}
	in = normalizeMenu(in)
	if id == uuid.Nil || !validMenu(in) {
		return accessdomain.Menu{}, accessdomain.ErrInvalid
	}
	return s.repository.UpdateMenu(ctx, scoped(a, p), id, in)
}

func (s *Service) MoveMenu(ctx context.Context, token string, p accessdomain.Principal, id uuid.UUID, in accessdomain.MenuMove) (accessdomain.Menu, error) {
	a, err := s.authorize(ctx, token, "sys.menu.move")
	if err != nil {
		return accessdomain.Menu{}, err
	}
	if id == uuid.Nil || (in.ParentID != nil && *in.ParentID == uuid.Nil) {
		return accessdomain.Menu{}, accessdomain.ErrInvalid
	}
	return s.repository.MoveMenu(ctx, scoped(a, p), id, in)
}

func (s *Service) DeleteMenu(ctx context.Context, token string, p accessdomain.Principal, id uuid.UUID) error {
	a, err := s.authorize(ctx, token, "sys.menu.delete")
	if err != nil {
		return err
	}
	if id == uuid.Nil {
		return accessdomain.ErrInvalid
	}
	return s.repository.DeleteMenu(ctx, scoped(a, p), id)
}

func oneOf(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}
func unique(ids []uuid.UUID) bool {
	seen := map[uuid.UUID]struct{}{}
	for _, id := range ids {
		if id == uuid.Nil {
			return false
		}
		if _, ok := seen[id]; ok {
			return false
		}
		seen[id] = struct{}{}
	}
	return true
}
