//go:build integration

package application

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"

	iam "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestScannerConfigTenantLockAuditAndPublicProjection(t *testing.T) {
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
	var tenantID, otherTenantID, userID, appID uuid.UUID
	if err = pool.QueryRow(ctx, `INSERT INTO iam.tenants(code,name) VALUES($1,'Scanner Integration') RETURNING id`, "scanner-"+suffix).Scan(&tenantID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO iam.tenants(code,name) VALUES($1,'Scanner Other Integration') RETURNING id`, "scanner-other-"+suffix).Scan(&otherTenantID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO iam.users(email,display_name,status) VALUES($1,'Scanner Admin','active') RETURNING id`, "scanner-"+suffix+"@example.test").Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO iam.tenant_members(tenant_id,user_id,status) VALUES($1,$2,'active')`, tenantID, userID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT id FROM app.applications WHERE tenant_id=$1 AND is_default`, tenantID).Scan(&appID); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanupCtx := context.Background()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM app.applications WHERE tenant_id IN ($1,$2)`, tenantID, otherTenantID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM iam.tenant_members WHERE tenant_id IN ($1,$2)`, tenantID, otherTenantID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM iam.tenants WHERE id IN ($1,$2)`, tenantID, otherTenantID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM iam.users WHERE id=$1`, userID)
	}()

	auth := &adminIntegrationAuthenticator{principal: iam.AuthenticatedContext{AuthContext: iam.AuthContext{
		User: iam.User{ID: userID, Status: "active"}, Tenant: iam.Tenant{ID: tenantID, Status: "active"},
		Permissions: []string{"app.scanner_config.read", "app.scanner_config.update"},
	}}}
	service := NewService(pool, auth)
	initial, err := service.GetAdminScannerConfig(ctx, "admin-token", appID)
	if err != nil || initial.WebViewEnabled || initial.LockVersion != 0 || len(initial.AllowedHostPatterns) != 0 {
		t.Fatalf("initial scanner config=%#v err=%v", initial, err)
	}

	created, err := service.UpdateAdminScannerConfig(ctx, "admin-token", appID, ScannerConfigInput{
		WebViewEnabled: true, AllowedHostPatterns: []string{"Example.COM.", "*.Sub.Example.com", "example.com"}, LockVersion: 0,
	}, "scanner-create")
	if err != nil {
		t.Fatal(err)
	}
	if created.LockVersion != 1 || !reflect.DeepEqual(created.AllowedHostPatterns, []string{"*.sub.example.com", "example.com"}) {
		t.Fatalf("created scanner config=%#v", created)
	}
	if _, err = service.UpdateAdminScannerConfig(ctx, "admin-token", appID, ScannerConfigInput{LockVersion: 0}, "scanner-stale"); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale scanner update error=%v", err)
	}

	public, err := service.PublicScannerConfig(ctx, appID)
	if err != nil || !public.WebView.Enabled || !reflect.DeepEqual(public.WebView.AllowedHostPatterns, created.AllowedHostPatterns) {
		t.Fatalf("public scanner config=%#v err=%v", public, err)
	}

	updated, err := service.UpdateAdminScannerConfig(ctx, "admin-token", appID, ScannerConfigInput{
		WebViewEnabled: false, AllowedHostPatterns: created.AllowedHostPatterns, LockVersion: created.LockVersion,
	}, "scanner-update")
	if err != nil || updated.LockVersion != 2 || updated.WebViewEnabled {
		t.Fatalf("updated scanner config=%#v err=%v", updated, err)
	}
	var auditCount int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit.operation_logs WHERE tenant_id=$1 AND permission_code='app.scanner_config.update' AND request_id IN ('scanner-create','scanner-update') AND COALESCE(before_data::text,'') NOT ILIKE '%raw_value%' AND COALESCE(after_data::text,'') NOT ILIKE '%raw_value%'`, tenantID).Scan(&auditCount); err != nil || auditCount != 2 {
		t.Fatalf("scanner audit count=%d err=%v", auditCount, err)
	}

	originalTenant := auth.principal.Tenant
	auth.principal.Tenant = iam.Tenant{ID: otherTenantID, Status: "active"}
	if _, err = service.GetAdminScannerConfig(ctx, "admin-token", appID); !errors.Is(err, ErrAppNotFound) {
		t.Fatalf("cross-tenant scanner read error=%v", err)
	}
	if _, err = service.UpdateAdminScannerConfig(ctx, "admin-token", appID, ScannerConfigInput{LockVersion: updated.LockVersion}, "scanner-cross-tenant"); !errors.Is(err, ErrAppNotFound) {
		t.Fatalf("cross-tenant scanner update error=%v", err)
	}
	auth.principal.Tenant = originalTenant
}
