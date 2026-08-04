package application

import (
	"context"
	"errors"
	"testing"

	accessdomain "github.com/appkernia/appkernia/server/internal/modules/accessadmin/domain"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/google/uuid"
)

type fakeAuthenticator struct {
	auth iamdomain.AuthenticatedContext
}

func (f fakeAuthenticator) Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error) {
	return f.auth, nil
}

type fakeRepository struct {
	tenantID  uuid.UUID
	principal accessdomain.Principal
}

func (f *fakeRepository) ListRoles(_ context.Context, id uuid.UUID, _ accessdomain.Filters) (accessdomain.RolePage, error) {
	f.tenantID = id
	return accessdomain.RolePage{}, nil
}
func (f *fakeRepository) CreateRole(_ context.Context, p accessdomain.Principal, _ accessdomain.RoleInput) (accessdomain.Role, error) {
	f.principal = p
	return accessdomain.Role{}, nil
}
func (*fakeRepository) UpdateRole(context.Context, accessdomain.Principal, uuid.UUID, accessdomain.RoleInput) (accessdomain.Role, error) {
	return accessdomain.Role{}, nil
}
func (*fakeRepository) DeleteRole(context.Context, accessdomain.Principal, uuid.UUID) error {
	return nil
}
func (*fakeRepository) ReplaceRolePermissions(context.Context, accessdomain.Principal, uuid.UUID, []uuid.UUID) (accessdomain.Role, error) {
	return accessdomain.Role{}, nil
}
func (*fakeRepository) ReplaceRoleMenus(context.Context, accessdomain.Principal, uuid.UUID, []uuid.UUID) (accessdomain.Role, error) {
	return accessdomain.Role{}, nil
}
func (*fakeRepository) ReplaceRoleDataScope(context.Context, accessdomain.Principal, uuid.UUID, string, []uuid.UUID) (accessdomain.Role, error) {
	return accessdomain.Role{}, nil
}
func (*fakeRepository) ListPermissions(context.Context, accessdomain.PermissionFilters) ([]accessdomain.Permission, error) {
	return nil, nil
}
func (*fakeRepository) ListMenus(context.Context, uuid.UUID) ([]accessdomain.Menu, error) {
	return nil, nil
}
func (*fakeRepository) CreateMenu(context.Context, accessdomain.Principal, accessdomain.MenuInput) (accessdomain.Menu, error) {
	return accessdomain.Menu{}, nil
}
func (*fakeRepository) UpdateMenu(context.Context, accessdomain.Principal, uuid.UUID, accessdomain.MenuInput) (accessdomain.Menu, error) {
	return accessdomain.Menu{}, nil
}
func (*fakeRepository) MoveMenu(context.Context, accessdomain.Principal, uuid.UUID, accessdomain.MenuMove) (accessdomain.Menu, error) {
	return accessdomain.Menu{}, nil
}
func (*fakeRepository) DeleteMenu(context.Context, accessdomain.Principal, uuid.UUID) error {
	return nil
}

func accessAuth(permissions ...string) iamdomain.AuthenticatedContext {
	return iamdomain.AuthenticatedContext{AuthContext: iamdomain.AuthContext{User: iamdomain.User{ID: uuid.New()}, Tenant: iamdomain.Tenant{ID: uuid.New()}, Permissions: permissions}, SessionID: uuid.New()}
}

func TestServiceUsesExactPermissionAndTenantScope(t *testing.T) {
	repository := &fakeRepository{}
	denied := NewService(fakeAuthenticator{auth: accessAuth("iam.role.reader")}, repository)
	if _, err := denied.Roles(context.Background(), "token", accessdomain.Filters{}); !errors.Is(err, accessdomain.ErrForbidden) {
		t.Fatalf("prefix permission error=%v", err)
	}
	auth := accessAuth("iam.role.read", "iam.role.create")
	service := NewService(fakeAuthenticator{auth: auth}, repository)
	if _, err := service.Roles(context.Background(), "token", accessdomain.Filters{}); err != nil {
		t.Fatalf("Roles error=%v", err)
	}
	if repository.tenantID != auth.Tenant.ID {
		t.Fatalf("tenant=%s want=%s", repository.tenantID, auth.Tenant.ID)
	}
	if _, err := service.CreateRole(context.Background(), "token", accessdomain.Principal{RequestID: "request"}, accessdomain.RoleInput{Code: "support-agent", Name: "Support agent", Status: "active"}); err != nil {
		t.Fatalf("CreateRole error=%v", err)
	}
	if repository.principal.TenantID != auth.Tenant.ID || repository.principal.UserID != auth.User.ID || repository.principal.SessionID != auth.SessionID {
		t.Fatalf("principal=%#v", repository.principal)
	}
}

func TestServiceRejectsInvalidScopeAndMenu(t *testing.T) {
	auth := accessAuth("iam.role.update_data_scope", "sys.menu.create")
	service := NewService(fakeAuthenticator{auth: auth}, &fakeRepository{})
	if _, err := service.ReplaceDataScope(context.Background(), "token", accessdomain.Principal{}, uuid.New(), "custom", nil); !errors.Is(err, accessdomain.ErrInvalid) {
		t.Fatalf("custom empty error=%v", err)
	}
	if _, err := service.ReplaceDataScope(context.Background(), "token", accessdomain.Principal{}, uuid.New(), "self", []uuid.UUID{uuid.New()}); !errors.Is(err, accessdomain.ErrInvalid) {
		t.Fatalf("self units error=%v", err)
	}
	if _, err := service.CreateMenu(context.Background(), "token", accessdomain.Principal{}, accessdomain.MenuInput{Code: "bad", Title: "Bad", I18nKey: "menu.bad", Type: "page", Path: "relative", ComponentKey: "dashboard", OpenMode: "same_tab", Status: "active"}); !errors.Is(err, accessdomain.ErrInvalid) {
		t.Fatalf("relative path error=%v", err)
	}
}
