//go:build integration

package application

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	iam "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type adminIntegrationAuthenticator struct {
	principal iam.AuthenticatedContext
}

func (auth *adminIntegrationAuthenticator) Authenticate(context.Context, string, string) (iam.AuthenticatedContext, error) {
	return auth.principal, nil
}

func TestAdminApplicationLifecycleTenantScopeAndAtomicDelete(t *testing.T) {
	dsn := os.Getenv("AK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	suffix := uuid.NewString()
	var tenantID, otherTenantID, userID uuid.UUID
	if err = pool.QueryRow(ctx, `INSERT INTO iam.tenants(code,name) VALUES($1,'App Admin Integration') RETURNING id`, "app-admin-"+suffix).Scan(&tenantID); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanupCtx := context.Background()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM sys.mobile_release_publications WHERE app_id IN (SELECT id FROM app.applications WHERE tenant_id IN ($1,$2))`, tenantID, otherTenantID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM sys.mobile_releases WHERE app_id IN (SELECT id FROM app.applications WHERE tenant_id IN ($1,$2))`, tenantID, otherTenantID)
		_, _ = pool.Exec(cleanupCtx, `UPDATE content.pages SET page_type='custom' WHERE tenant_id IN ($1,$2)`, tenantID, otherTenantID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM app.applications WHERE tenant_id IN ($1,$2)`, tenantID, otherTenantID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM iam.tenant_members WHERE tenant_id IN ($1,$2)`, tenantID, otherTenantID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM iam.tenants WHERE id IN ($1,$2)`, tenantID, otherTenantID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM iam.users WHERE id=$1`, userID)
	}()
	if err = pool.QueryRow(ctx, `INSERT INTO iam.tenants(code,name) VALUES($1,'Other App Admin Integration') RETURNING id`, "other-app-admin-"+suffix).Scan(&otherTenantID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO iam.users(email,display_name,status) VALUES($1,'App Admin','active') RETURNING id`, "app-admin-"+suffix+"@example.test").Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO iam.tenant_members(tenant_id,user_id,status) VALUES($1,$2,'active')`, tenantID, userID); err != nil {
		t.Fatal(err)
	}
	auth := &adminIntegrationAuthenticator{principal: iam.AuthenticatedContext{AuthContext: iam.AuthContext{
		User: iam.User{ID: userID, Status: "active"}, Tenant: iam.Tenant{ID: tenantID, Status: "active"},
		Permissions: []string{"app.application.read", "app.application.create", "app.application.update", "app.application.disable", "app.application.delete"},
	}}}
	service := NewService(pool, auth)
	manifestPrefix := "__UNI__" + strings.ToUpper(strings.ReplaceAll(suffix, "-", ""))
	h5URL := "https://example.test"
	input := AdminAppInput{
		AppID: manifestPrefix + "A", AppType: "uni_app_x", Name: "Integration App A", Description: "Description",
		DefaultLocale: "zh-CN", RegistrationVerification: "none", OwnerType: "tenant", OwnerID: &tenantID,
		Channels:      []ApplicationChannel{{ChannelCode: "h5", Name: "Web", URL: &h5URL, Enabled: true}},
		StoreListings: []ApplicationStoreListing{{Name: "Primary", Scheme: "app-admin://", Enabled: true, Priority: 10}},
	}
	created, err := service.CreateAdminApp(ctx, "admin-token", input)
	if err != nil {
		t.Fatal(err)
	}
	if created.TenantID != tenantID || created.CreatorUserID == nil || *created.CreatorUserID != userID ||
		len(created.Managers) != 1 || created.Managers[0] != userID || len(created.Channels) != 1 || len(created.StoreListings) != 1 {
		t.Fatalf("created application relations=%#v", created)
	}
	if _, err = service.CreateAdminApp(ctx, "admin-token", input); err == nil {
		t.Fatal("duplicate manifest AppID accepted")
	}

	update := input
	update.Code = created.Code
	update.Name = "Integration App A Updated"
	update.LockVersion = created.LockVersion
	update.StoreListings = created.StoreListings
	changedAppID := update
	changedAppID.AppID = manifestPrefix + "CHANGED"
	if _, err = service.UpdateAdminApp(ctx, "admin-token", created.ID, changedAppID, nil); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("manifest AppID mutation error=%v", err)
	}
	changedType := update
	changedType.AppType = "uni_app"
	if _, err = service.UpdateAdminApp(ctx, "admin-token", created.ID, changedType, nil); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("App type mutation error=%v", err)
	}
	created, err = service.UpdateAdminApp(ctx, "admin-token", created.ID, update, nil)
	if err != nil || created.Name != update.Name {
		t.Fatalf("update application=%#v err=%v", created, err)
	}

	originalTenant := auth.principal.Tenant
	auth.principal.Tenant = iam.Tenant{ID: otherTenantID, Status: "active"}
	if _, err = service.GetAdminApp(ctx, "admin-token", created.ID); !errors.Is(err, ErrAppNotFound) {
		t.Fatalf("cross-tenant application read error=%v", err)
	}
	auth.principal.Tenant = originalTenant

	var defaultAppID uuid.UUID
	if err = pool.QueryRow(ctx, `SELECT id FROM app.applications WHERE tenant_id=$1 AND is_default`, tenantID).Scan(&defaultAppID); err != nil {
		t.Fatal(err)
	}
	defaultApp, err := service.GetAdminApp(ctx, "admin-token", defaultAppID)
	if err != nil {
		t.Fatal(err)
	}
	defaultApp, err = service.SetAdminAppStatus(ctx, "admin-token", defaultApp.ID, "disabled", defaultApp.LockVersion)
	if err != nil {
		t.Fatal(err)
	}
	if err = service.DeleteAdminApps(ctx, "admin-token", []uuid.UUID{defaultApp.ID}, "integration-default-delete"); !errors.Is(err, ErrConflict) {
		t.Fatalf("default application delete error=%v", err)
	}
	if _, err = service.SetAdminAppStatus(ctx, "admin-token", defaultApp.ID, "active", defaultApp.LockVersion); err != nil {
		t.Fatal(err)
	}

	secondInput := input
	secondInput.AppID = manifestPrefix + "B"
	secondInput.Name = "Integration App B"
	secondInput.Code = ""
	secondInput.StoreListings = nil
	secondInput.Channels = nil
	second, err := service.CreateAdminApp(ctx, "admin-token", secondInput)
	if err != nil {
		t.Fatal(err)
	}
	created, err = service.SetAdminAppStatus(ctx, "admin-token", created.ID, "disabled", created.LockVersion)
	if err != nil {
		t.Fatal(err)
	}
	if err = service.DeleteAdminApps(ctx, "admin-token", []uuid.UUID{created.ID, second.ID}, "integration-atomic-delete"); !errors.Is(err, ErrConflict) {
		t.Fatalf("mixed-status batch delete error=%v", err)
	}
	if _, err = service.GetAdminApp(ctx, "admin-token", created.ID); err != nil {
		t.Fatalf("atomic delete removed eligible application: %v", err)
	}
	second, err = service.SetAdminAppStatus(ctx, "admin-token", second.ID, "disabled", second.LockVersion)
	if err != nil {
		t.Fatal(err)
	}
	if err = service.DeleteAdminApps(ctx, "admin-token", []uuid.UUID{created.ID, second.ID}, "integration-batch-delete"); err != nil {
		t.Fatal(err)
	}
	if _, err = service.GetAdminApp(ctx, "admin-token", created.ID); !errors.Is(err, ErrAppNotFound) {
		t.Fatalf("soft-deleted application remains visible: %v", err)
	}
	if err = service.DeleteAdminApps(ctx, "admin-token", []uuid.UUID{second.ID, second.ID}, "integration-duplicate-delete"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("duplicate batch IDs error=%v", err)
	}
	var auditCount int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit.operation_logs WHERE tenant_id=$1 AND request_id='integration-batch-delete' AND permission_code='app.application.delete'`, tenantID).Scan(&auditCount); err != nil || auditCount != 2 {
		t.Fatalf("delete audit count=%d err=%v", auditCount, err)
	}
}
