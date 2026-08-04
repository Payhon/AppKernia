package application

import (
	"context"
	"errors"
	"testing"
	"time"

	auditdomain "github.com/appkernia/appkernia/server/internal/modules/auditadmin/domain"
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
	principal auditdomain.Principal
}

func (f *fakeRepository) ListOperations(_ context.Context, tenantID uuid.UUID, _ auditdomain.OperationFilter) (auditdomain.Page[auditdomain.Operation], error) {
	f.tenantID = tenantID
	return auditdomain.Page[auditdomain.Operation]{}, nil
}
func (*fakeRepository) ListLogins(context.Context, uuid.UUID, auditdomain.LoginFilter) (auditdomain.Page[auditdomain.Login], error) {
	return auditdomain.Page[auditdomain.Login]{}, nil
}
func (*fakeRepository) ListSecurityEvents(context.Context, uuid.UUID, auditdomain.SecurityFilter) (auditdomain.Page[auditdomain.SecurityEvent], error) {
	return auditdomain.Page[auditdomain.SecurityEvent]{}, nil
}
func (*fakeRepository) GetSecurityEvent(context.Context, uuid.UUID, uuid.UUID) (auditdomain.SecurityEvent, error) {
	return auditdomain.SecurityEvent{}, nil
}
func (f *fakeRepository) ResolveSecurityEvent(_ context.Context, principal auditdomain.Principal, _ uuid.UUID) (auditdomain.SecurityEvent, error) {
	f.principal = principal
	return auditdomain.SecurityEvent{}, nil
}

func auditAuth(permissions ...string) iamdomain.AuthenticatedContext {
	return iamdomain.AuthenticatedContext{AuthContext: iamdomain.AuthContext{User: iamdomain.User{ID: uuid.New()}, Tenant: iamdomain.Tenant{ID: uuid.New()}, Permissions: permissions}, SessionID: uuid.New()}
}

func TestServiceUsesExactPermissionsAndTenantScope(t *testing.T) {
	repository := &fakeRepository{}
	denied := NewService(fakeAuthenticator{auth: auditAuth("audit.operation.reader")}, repository)
	if _, err := denied.Operations(context.Background(), "token", auditdomain.OperationFilter{}); !errors.Is(err, auditdomain.ErrForbidden) {
		t.Fatalf("prefix permission error=%v", err)
	}
	auth := auditAuth("audit.operation.read", "audit.security.resolve")
	service := NewService(fakeAuthenticator{auth: auth}, repository)
	service.now = func() time.Time { return time.Date(2026, 8, 3, 8, 0, 0, 0, time.UTC) }
	if _, err := service.Operations(context.Background(), "token", auditdomain.OperationFilter{}); err != nil {
		t.Fatalf("Operations error=%v", err)
	}
	if repository.tenantID != auth.Tenant.ID {
		t.Fatalf("tenant=%s want=%s", repository.tenantID, auth.Tenant.ID)
	}
	if _, err := service.ResolveSecurityEvent(context.Background(), "token", auditdomain.Principal{RequestID: "audit-test"}, uuid.New()); err != nil {
		t.Fatalf("ResolveSecurityEvent error=%v", err)
	}
	if repository.principal.TenantID != auth.Tenant.ID || repository.principal.UserID != auth.User.ID || repository.principal.SessionID != auth.SessionID {
		t.Fatalf("principal=%#v", repository.principal)
	}
}

func TestServiceRejectsInvalidFilters(t *testing.T) {
	service := NewService(fakeAuthenticator{auth: auditAuth("audit.login.read", "audit.security.read")}, &fakeRepository{})
	if _, err := service.Logins(context.Background(), "token", auditdomain.LoginFilter{Result: "unknown"}); !errors.Is(err, auditdomain.ErrInvalid) {
		t.Fatalf("invalid login result error=%v", err)
	}
	if _, err := service.SecurityEvents(context.Background(), "token", auditdomain.SecurityFilter{PageFilter: auditdomain.PageFilter{FromAt: time.Now(), ToAt: time.Now().AddDate(0, 0, -1)}}); !errors.Is(err, auditdomain.ErrInvalid) {
		t.Fatalf("reversed range error=%v", err)
	}
}
