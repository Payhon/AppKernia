package application

import (
	"context"
	"errors"
	"testing"
	"time"

	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	sessiondomain "github.com/appkernia/appkernia/server/internal/modules/sessionadmin/domain"
	"github.com/google/uuid"
)

type fakeAuth struct {
	value iamdomain.AuthenticatedContext
}

func (f fakeAuth) Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error) {
	return f.value, nil
}

type fakeRepo struct {
	tenant, current uuid.UUID
	principal       sessiondomain.Principal
}

func (f *fakeRepo) List(_ context.Context, tenant, current uuid.UUID, _ sessiondomain.Filter) (sessiondomain.Page, error) {
	f.tenant, f.current = tenant, current
	return sessiondomain.Page{}, nil
}
func (f *fakeRepo) Revoke(_ context.Context, principal sessiondomain.Principal, id uuid.UUID) (sessiondomain.RevokeResult, error) {
	f.principal = principal
	return sessiondomain.RevokeResult{ID: id, Revoked: true, Current: id == principal.SessionID}, nil
}

func sessionAuth(permissions ...string) iamdomain.AuthenticatedContext {
	return iamdomain.AuthenticatedContext{AuthContext: iamdomain.AuthContext{User: iamdomain.User{ID: uuid.New()}, Tenant: iamdomain.Tenant{ID: uuid.New()}, Permissions: permissions}, SessionID: uuid.New()}
}

func TestServiceUsesExactPermissionsAndAuthenticatedScope(t *testing.T) {
	repo := &fakeRepo{}
	denied := NewService(fakeAuth{value: sessionAuth("iam.session.reader")}, repo)
	if _, err := denied.List(context.Background(), "token", sessiondomain.Filter{}); !errors.Is(err, sessiondomain.ErrForbidden) {
		t.Fatalf("permission error=%v", err)
	}
	auth := sessionAuth("iam.session.read", "iam.session.revoke")
	service := NewService(fakeAuth{value: auth}, repo)
	service.now = func() time.Time { return time.Date(2026, 8, 3, 8, 0, 0, 0, time.UTC) }
	if _, err := service.List(context.Background(), "token", sessiondomain.Filter{}); err != nil {
		t.Fatal(err)
	}
	if repo.tenant != auth.Tenant.ID || repo.current != auth.SessionID {
		t.Fatalf("scope tenant=%s current=%s", repo.tenant, repo.current)
	}
	if _, err := service.Revoke(context.Background(), "token", sessiondomain.Principal{RequestID: "request"}, auth.SessionID); err != nil {
		t.Fatal(err)
	}
	if repo.principal.TenantID != auth.Tenant.ID || repo.principal.UserID != auth.User.ID || repo.principal.SessionID != auth.SessionID {
		t.Fatalf("principal=%#v", repo.principal)
	}
}

func TestServiceRejectsInvalidFilters(t *testing.T) {
	service := NewService(fakeAuth{value: sessionAuth("iam.session.read")}, &fakeRepo{})
	cases := []sessiondomain.Filter{{Status: "unknown"}, {PageSize: 101}, {FromAt: time.Now(), ToAt: time.Now().Add(-time.Hour)}, {Platform: "browser"}}
	for _, input := range cases {
		if _, err := service.List(context.Background(), "token", input); !errors.Is(err, sessiondomain.ErrInvalid) {
			t.Fatalf("filter=%#v error=%v", input, err)
		}
	}
}
