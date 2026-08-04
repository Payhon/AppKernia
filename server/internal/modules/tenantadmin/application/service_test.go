package application

import (
	"context"
	"errors"
	"testing"

	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	tenantdomain "github.com/appkernia/appkernia/server/internal/modules/tenantadmin/domain"
	"github.com/google/uuid"
)

type fakeTenantAuthenticator struct {
	auth iamdomain.AuthenticatedContext
}

func (f fakeTenantAuthenticator) Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error) {
	return f.auth, nil
}

type fakeTenantRepository struct {
	listedTenant uuid.UUID
	principal    tenantdomain.Principal
}

func (f *fakeTenantRepository) List(_ context.Context, tenantID uuid.UUID, _ tenantdomain.Filters) (tenantdomain.Page, error) {
	f.listedTenant = tenantID
	return tenantdomain.Page{}, nil
}
func (*fakeTenantRepository) Get(context.Context, uuid.UUID, uuid.UUID) (tenantdomain.Tenant, error) {
	return tenantdomain.Tenant{}, nil
}
func (f *fakeTenantRepository) Create(_ context.Context, p tenantdomain.Principal, _ tenantdomain.CreateInput) (tenantdomain.Tenant, error) {
	f.principal = p
	return tenantdomain.Tenant{}, nil
}
func (*fakeTenantRepository) Update(context.Context, tenantdomain.Principal, uuid.UUID, tenantdomain.UpdateInput) (tenantdomain.Tenant, error) {
	return tenantdomain.Tenant{}, nil
}
func (*fakeTenantRepository) Members(context.Context, uuid.UUID, uuid.UUID) ([]tenantdomain.Member, error) {
	return nil, nil
}
func (*fakeTenantRepository) AddMember(context.Context, tenantdomain.Principal, uuid.UUID, tenantdomain.AddMemberInput) (tenantdomain.Member, error) {
	return tenantdomain.Member{}, nil
}
func (*fakeTenantRepository) SetMemberStatus(context.Context, tenantdomain.Principal, uuid.UUID, uuid.UUID, string) (tenantdomain.Member, error) {
	return tenantdomain.Member{}, nil
}

func tenantAuth(permissions ...string) iamdomain.AuthenticatedContext {
	return iamdomain.AuthenticatedContext{
		AuthContext: iamdomain.AuthContext{
			User:        iamdomain.User{ID: uuid.New()},
			Tenant:      iamdomain.Tenant{ID: uuid.New()},
			Permissions: permissions,
		},
		SessionID: uuid.New(),
	}
}

func TestTenantServiceFeatureFlagAndExactPermissions(t *testing.T) {
	repository := &fakeTenantRepository{}
	disabled := NewService(fakeTenantAuthenticator{auth: tenantAuth("iam.tenant.read")}, repository, false)
	if _, err := disabled.List(context.Background(), "token", tenantdomain.Filters{}); !errors.Is(err, tenantdomain.ErrNotFound) {
		t.Fatalf("disabled List() error = %v", err)
	}

	enabled := NewService(fakeTenantAuthenticator{auth: tenantAuth("iam.tenant.reader")}, repository, true)
	if _, err := enabled.List(context.Background(), "token", tenantdomain.Filters{}); !errors.Is(err, tenantdomain.ErrForbidden) {
		t.Fatalf("prefix permission must not authorize List(), error = %v", err)
	}
}

func TestTenantServiceScopesReadsAndWritesToAuthenticatedContext(t *testing.T) {
	auth := tenantAuth("iam.tenant.read", "iam.tenant.create")
	repository := &fakeTenantRepository{}
	service := NewService(fakeTenantAuthenticator{auth: auth}, repository, true)
	if _, err := service.List(context.Background(), "token", tenantdomain.Filters{}); err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if repository.listedTenant != auth.Tenant.ID {
		t.Fatalf("List() tenant = %s, want %s", repository.listedTenant, auth.Tenant.ID)
	}
	if _, err := service.Create(context.Background(), "token", tenantdomain.Principal{RequestID: "request"}, tenantdomain.CreateInput{Code: "new-tenant", Name: "New Tenant"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if repository.principal.TenantID != auth.Tenant.ID || repository.principal.UserID != auth.User.ID || repository.principal.SessionID != auth.SessionID {
		t.Fatalf("Create() principal = %#v", repository.principal)
	}
}

func TestTenantServiceRejectsInvalidCodesAndMemberStatus(t *testing.T) {
	auth := tenantAuth("iam.tenant.create", "iam.tenant.member.update")
	service := NewService(fakeTenantAuthenticator{auth: auth}, &fakeTenantRepository{}, true)
	if _, err := service.Create(context.Background(), "token", tenantdomain.Principal{}, tenantdomain.CreateInput{Code: "Bad Code", Name: "Name"}); !errors.Is(err, tenantdomain.ErrInvalid) {
		t.Fatalf("Create(invalid code) error = %v", err)
	}
	if _, err := service.SetMemberStatus(context.Background(), "token", tenantdomain.Principal{}, uuid.New(), uuid.New(), "deleted"); !errors.Is(err, tenantdomain.ErrInvalid) {
		t.Fatalf("SetMemberStatus(invalid) error = %v", err)
	}
}
