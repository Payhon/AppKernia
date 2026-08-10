//go:build integration

package seed

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	"github.com/appkernia/appkernia/server/internal/modules/iam/application"
	"github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	settingsrepository "github.com/appkernia/appkernia/server/internal/modules/systemsettings/repository"
	"github.com/appkernia/appkernia/server/internal/platform/buildinfo"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCoreDictionariesSeedsSystemLanguageIdempotently(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	catalogPath := filepath.Join("..", "..", "..", "blueprint", "backend", "spec", "core-dictionaries.json")
	catalog, err := readDictionaryCatalog(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	wantCount := len(catalog.Types) + len(catalog.Items) + len(catalog.Templates)
	for run := 0; run < 2; run++ {
		count, seedErr := CoreDictionaries(ctx, pool, catalogPath)
		if seedErr != nil || count != wantCount {
			t.Fatalf("run=%d count=%d want=%d error=%v", run+1, count, wantCount, seedErr)
		}
	}

	resolved, err := settingsrepository.NewPostgres(pool).ResolveDictionary(ctx, nil, "system.language", "en-US", false)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ExtensionPolicy != "fixed" || len(resolved.Items) != 2 || resolved.Items[0].Value != "zh-CN" || resolved.Items[0].Label != "Simplified Chinese" || !resolved.Items[0].IsDefault || resolved.Items[1].Value != "en-US" || resolved.Items[1].Label != "English" {
		t.Fatalf("unexpected resolved system language dictionary: %+v", resolved)
	}
}

func TestBootstrapAdminReusesExistingTenantAndPreservesCredential(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	suffix := strings.ToLower(strings.ReplaceAll(time.Now().UTC().Format("20060102T150405.000000000"), ".", ""))
	tenantCode := "seed-existing-" + suffix
	email := "seed-existing-" + suffix + "@example.test"
	tenant, err := db.New(pool).CreateTenant(ctx, db.CreateTenantParams{
		Code: tenantCode, Name: "Existing Seed Tenant", Status: "active", Settings: []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	var userID string
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `
			DELETE FROM sys.role_menus WHERE tenant_id = $1;
			DELETE FROM iam.role_permissions WHERE tenant_id = $1;
			DELETE FROM iam.user_roles WHERE tenant_id = $1;
			DELETE FROM iam.roles WHERE tenant_id = $1;
			DELETE FROM sys.config_items WHERE tenant_id = $1;
			DELETE FROM iam.tenant_members WHERE tenant_id = $1;
			DELETE FROM iam.user_credentials WHERE user_id::text = NULLIF($2, '');
			DELETE FROM iam.users WHERE id::text = NULLIF($2, '');
			DELETE FROM iam.tenants WHERE id = $1;
		`, tenant.ID, userID)
	}()

	firstPassword := "integration seed password 2026!"
	input := BootstrapAdminInput{
		TenantCode: tenantCode, TenantName: "Existing Seed Tenant", Email: email,
		DisplayName: "Seed Administrator", Locale: "zh-CN", Password: firstPassword,
	}
	firstUser, firstTenant, firstPermissions, firstMenus, err := BootstrapAdmin(ctx, pool, input)
	if err != nil {
		t.Fatal(err)
	}
	userID = firstUser.ID.String()
	if firstTenant.ID != tenant.ID || firstPermissions == 0 || firstMenus == 0 {
		t.Fatalf("tenant=%s want=%s permissions=%d menus=%d", firstTenant.ID, tenant.ID, firstPermissions, firstMenus)
	}

	input.Password = "different integration password 2026!"
	secondUser, secondTenant, secondPermissions, secondMenus, err := BootstrapAdmin(ctx, pool, input)
	if err != nil {
		t.Fatal(err)
	}
	if secondUser.ID != firstUser.ID || secondTenant.ID != firstTenant.ID {
		t.Fatalf("bootstrap was not idempotent: first_user=%s second_user=%s first_tenant=%s second_tenant=%s", firstUser.ID, secondUser.ID, firstTenant.ID, secondTenant.ID)
	}
	if secondPermissions != 0 || secondMenus != 0 {
		t.Fatalf("second bootstrap grants permissions=%d menus=%d want=0", secondPermissions, secondMenus)
	}
	credential, err := db.New(pool).GetCredentialByEmail(ctx, &email)
	if err != nil {
		t.Fatal(err)
	}
	if !application.VerifyPassword(credential.PasswordHash, firstPassword) || application.VerifyPassword(credential.PasswordHash, input.Password) {
		t.Fatal("idempotent bootstrap must preserve the existing credential")
	}
}

func TestBootstrapAdminRejectsExistingIdentityOutsideTargetTenant(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	suffix := strings.ToLower(strings.ReplaceAll(time.Now().UTC().Format("20060102T150405.000000000"), ".", ""))
	email := "seed-cross-tenant-" + suffix + "@example.test"
	passwordHash, err := application.HashPassword("integration seed password 2026!")
	if err != nil {
		t.Fatal(err)
	}
	user, sourceTenant, err := repository.NewPostgres(pool).CreateIdentity(ctx, domain.CreateIdentity{
		TenantCode: "seed-source-" + suffix, TenantName: "Seed Source Tenant", Email: email,
		DisplayName: "Seed Source User", Locale: "zh-CN", PasswordHash: passwordHash,
	})
	if err != nil {
		t.Fatal(err)
	}
	targetTenant, err := db.New(pool).CreateTenant(ctx, db.CreateTenantParams{
		Code: "seed-target-" + suffix, Name: "Seed Target Tenant", Status: "active", Settings: []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `
			DELETE FROM iam.tenant_members WHERE user_id = $1;
			DELETE FROM iam.user_credentials WHERE user_id = $1;
			DELETE FROM iam.users WHERE id = $1;
			DELETE FROM iam.tenants WHERE id IN ($2, $3);
		`, user.ID, sourceTenant.ID, targetTenant.ID)
	}()

	_, _, _, _, err = BootstrapAdmin(ctx, pool, BootstrapAdminInput{
		TenantCode: targetTenant.Code, TenantName: targetTenant.Name, Email: email,
		DisplayName: "Unexpected Administrator", Locale: "zh-CN",
		Password: "different integration password 2026!",
	})
	if err == nil || !strings.Contains(err.Error(), "not an active member") {
		t.Fatalf("cross-tenant identity must be rejected, err=%v", err)
	}
	var assignments int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM iam.user_roles WHERE tenant_id = $1 AND user_id = $2`, targetTenant.ID, user.ID).Scan(&assignments); err != nil {
		t.Fatal(err)
	}
	if assignments != 0 {
		t.Fatalf("target tenant role assignments=%d want=0", assignments)
	}
}

func TestCoreModulesIsExactIdempotentAndVersioned(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	catalogPath := filepath.Join("..", "..", "..", "blueprint", "backend", "spec", "core-modules.json")
	previousVersion := buildinfo.Version
	buildinfo.Version = "integration-module-catalog"
	defer func() {
		buildinfo.Version = previousVersion
		restoreCtx, restoreCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer restoreCancel()
		_, _ = CoreModules(restoreCtx, pool, catalogPath)
	}()

	if _, err = pool.Exec(ctx, `
		INSERT INTO sys.modules(code,name,name_key,version,description,description_key,capabilities,status)
		VALUES('e2e.catalog','fixture','fixture.name','fixture','fixture','fixture.description','{"fixture":true}','enabled')
		ON CONFLICT (code) DO UPDATE SET version='fixture'
	`); err != nil {
		t.Fatal(err)
	}
	for run := 0; run < 2; run++ {
		count, seedErr := CoreModules(ctx, pool, catalogPath)
		if seedErr != nil || count != 8 {
			t.Fatalf("run=%d count=%d error=%v", run+1, count, seedErr)
		}
	}

	rows, err := pool.Query(ctx, `SELECT code::text, version FROM sys.modules ORDER BY code`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var codes []string
	for rows.Next() {
		var code, version string
		if err = rows.Scan(&code, &version); err != nil {
			t.Fatal(err)
		}
		codes = append(codes, code)
		if version != buildinfo.Version {
			t.Fatalf("module %s version=%q want=%q", code, version, buildinfo.Version)
		}
	}
	want := []string{"audit", "iam", "jobs", "notify", "ops", "org", "storage", "sys"}
	if !reflect.DeepEqual(codes, want) {
		t.Fatalf("module codes=%v want=%v", codes, want)
	}
}
