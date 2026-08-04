package application

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	userdomain "github.com/appkernia/appkernia/server/internal/modules/useradmin/domain"
	"github.com/google/uuid"
)

type fakeAuthenticator struct {
	auth iamdomain.AuthenticatedContext
}

func (fake fakeAuthenticator) Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error) {
	return fake.auth, nil
}

type fakeRepository struct {
	created   int
	principal userdomain.Principal
}

func (fake *fakeRepository) ListRoleOptions(context.Context, uuid.UUID) ([]userdomain.Reference, error) {
	return nil, nil
}
func (fake *fakeRepository) ListUsers(context.Context, uuid.UUID, userdomain.Filters) (userdomain.Page, error) {
	return userdomain.Page{}, nil
}
func (fake *fakeRepository) GetUser(context.Context, uuid.UUID, uuid.UUID) (userdomain.User, error) {
	return userdomain.User{}, nil
}
func (fake *fakeRepository) CreateUser(_ context.Context, p userdomain.Principal, _ userdomain.CreateInput, _ string) (userdomain.User, error) {
	fake.created++
	fake.principal = p
	return userdomain.User{}, nil
}
func (fake *fakeRepository) UpdateUser(context.Context, userdomain.Principal, uuid.UUID, userdomain.UpdateInput) (userdomain.User, error) {
	return userdomain.User{}, nil
}
func (fake *fakeRepository) SetMemberStatus(context.Context, userdomain.Principal, uuid.UUID, string) (userdomain.User, error) {
	return userdomain.User{}, nil
}
func (fake *fakeRepository) UnlockUser(context.Context, userdomain.Principal, uuid.UUID) error {
	return nil
}
func (fake *fakeRepository) ResetPassword(context.Context, userdomain.Principal, uuid.UUID, string) (int64, error) {
	return 0, nil
}
func (fake *fakeRepository) ReplaceRoles(context.Context, userdomain.Principal, uuid.UUID, []uuid.UUID) (userdomain.User, error) {
	return userdomain.User{}, nil
}
func (fake *fakeRepository) ReplaceAssignments(context.Context, userdomain.Principal, uuid.UUID, userdomain.AssignmentInput) (userdomain.User, error) {
	return userdomain.User{}, nil
}
func (fake *fakeRepository) ListSessions(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) ([]userdomain.Session, error) {
	return nil, nil
}
func (fake *fakeRepository) RevokeSession(context.Context, userdomain.Principal, uuid.UUID, uuid.UUID) error {
	return nil
}

func TestUserServiceRequiresPermissionAndScopesImportPrincipal(t *testing.T) {
	tenantID, userID, sessionID := uuid.New(), uuid.New(), uuid.New()
	repository := &fakeRepository{}
	service := NewService(fakeAuthenticator{auth: iamdomain.AuthenticatedContext{AuthContext: iamdomain.AuthContext{Tenant: iamdomain.Tenant{ID: tenantID}}, SessionID: sessionID}}, repository)
	if _, err := service.List(context.Background(), "token", userdomain.Filters{}); !errors.Is(err, userdomain.ErrForbidden) {
		t.Fatalf("List() without permission error=%v", err)
	}
	service = NewService(fakeAuthenticator{auth: iamdomain.AuthenticatedContext{AuthContext: iamdomain.AuthContext{User: iamdomain.User{ID: userID}, Tenant: iamdomain.Tenant{ID: tenantID}, Permissions: []string{"iam.user.import"}}, SessionID: sessionID}}, repository)
	csvData := "email,display_name,locale,time_zone,temporary_password\nvalid@example.test,Valid User,en-US,UTC,Temporary!Password2026\ninvalid,Invalid User,en-US,UTC,Temporary!Password2026\n"
	result, err := service.Import(context.Background(), "token", userdomain.Principal{RequestID: "request"}, strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("Import() error=%v", err)
	}
	if result.Created != 1 || result.Failed != 1 || repository.created != 1 {
		t.Fatalf("Import() result=%#v created=%d", result, repository.created)
	}
	if repository.principal.TenantID != tenantID || repository.principal.UserID != userID || repository.principal.SessionID != sessionID {
		t.Fatalf("scoped principal=%#v", repository.principal)
	}
}

func TestUserServiceRejectsSelfDisableAndExportsOnlySafeFields(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	repository := &fakeRepository{}
	service := NewService(fakeAuthenticator{auth: iamdomain.AuthenticatedContext{AuthContext: iamdomain.AuthContext{User: iamdomain.User{ID: userID}, Tenant: iamdomain.Tenant{ID: tenantID}, Permissions: []string{"iam.user.disable", "iam.user.export"}}}}, repository)
	if _, err := service.SetStatus(context.Background(), "token", userdomain.Principal{}, userID, false); !errors.Is(err, userdomain.ErrInvalid) {
		t.Fatalf("SetStatus(self) error=%v", err)
	}
	var output bytes.Buffer
	if err := service.Export(context.Background(), "token", userdomain.Filters{}, &output); err != nil {
		t.Fatalf("Export() error=%v", err)
	}
	header := output.String()
	if strings.Contains(header, "password") || strings.Contains(header, "secret") || !strings.HasPrefix(header, "email,display_name,status") {
		t.Fatalf("unsafe export header=%q", header)
	}
}
