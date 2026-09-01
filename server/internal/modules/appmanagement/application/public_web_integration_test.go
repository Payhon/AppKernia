//go:build integration

package application

import (
	"context"
	"encoding/json"
	"errors"
	iam "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	storage "github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	storagerepo "github.com/appkernia/appkernia/server/internal/modules/storage/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"os"
	"sync"
	"testing"
)

func TestPublicWebConfigurationScopeVersionAndAudit(t *testing.T) {
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
	var tenant, user, appID, storeID uuid.UUID
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	must(pool.QueryRow(ctx, `INSERT INTO iam.tenants(code,name) VALUES($1,'H5 Integration') RETURNING id`, "h5-"+uuid.NewString()).Scan(&tenant))
	must(pool.QueryRow(ctx, `INSERT INTO iam.users(email,display_name,status) VALUES($1,'H5 Admin','active') RETURNING id`, uuid.NewString()+"@example.test").Scan(&user))
	_, err = pool.Exec(ctx, `INSERT INTO iam.tenant_members(tenant_id,user_id,status) VALUES($1,$2,'active')`, tenant, user)
	must(err)
	defer func() {
		for _, q := range []string{`DELETE FROM app.applications WHERE tenant_id=$1`, `DELETE FROM storage.files WHERE tenant_id=$1`, `DELETE FROM iam.tenant_members WHERE tenant_id=$1`, `DELETE FROM iam.tenants WHERE id=$1`} {
			_, _ = pool.Exec(ctx, q, tenant)
		}
		_, _ = pool.Exec(ctx, `DELETE FROM iam.users WHERE id=$1`, user)
	}()
	must(pool.QueryRow(ctx, `SELECT id FROM app.applications WHERE tenant_id=$1 AND is_default`, tenant).Scan(&appID))
	must(pool.QueryRow(ctx, `INSERT INTO app.application_store_listings(tenant_id,app_id,name,scheme,enabled,priority) VALUES($1,$2,'Legacy','legacy://open',true,7) RETURNING id`, tenant, appID).Scan(&storeID))
	auth := &adminIntegrationAuthenticator{principal: iam.AuthenticatedContext{AuthContext: iam.AuthContext{User: iam.User{ID: user, Status: "active"}, Tenant: iam.Tenant{ID: tenant, Status: "active"}, Permissions: []string{"app.public_web.read", "app.public_web.update"}}}}
	svc := NewService(pool, auth, WithPublicWebBaseURL("https://public.example"))
	cfg, err := svc.GetAdminPublicWebConfig(ctx, "token", appID)
	must(err)
	if cfg.Enabled || !cfg.PromotionEnabled || cfg.LockVersion != 0 || cfg.Stores[0].Platform != "" || cfg.Stores[0].WebURL != "" {
		t.Fatalf("unsafe migration defaults: %+v", cfg)
	}
	promotionEnabled := true
	in := PublicWebInput{Enabled: true, APKEnabled: true, PromotionEnabled: &promotionEnabled, Translations: map[string]WebTranslationInput{"zh-CN": {Name: "中文应用", Introduction: "中文介绍", PromotionTitle: stringPointer("下载中文应用"), PromotionDescription: stringPointer("在 App 中发现更多内容"), PromotionButtonLabel: stringPointer("立即下载")}, "en-US": {Name: "English App", Introduction: "English introduction", PromotionTitle: stringPointer("Get English App"), PromotionDescription: stringPointer("Discover more in the app"), PromotionButtonLabel: stringPointer("Download now")}}, Stores: cfg.Stores}
	in.Stores[0].Platform = "ios"
	in.Stores[0].WebURL = "https://apps.apple.com/app/test"
	cfg, err = svc.UpdateAdminPublicWebConfig(ctx, "token", appID, in, "public-web-test")
	must(err)
	if cfg.LockVersion != 1 || !cfg.Enabled {
		t.Fatalf("config not saved %+v", cfg)
	}
	if _, err = svc.UpdateAdminPublicWebConfig(ctx, "token", appID, in, "stale"); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale update: %v", err)
	}
	// Both writers submit one version: exactly one can commit.
	in.LockVersion = cfg.LockVersion
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := svc.UpdateAdminPublicWebConfig(ctx, "token", appID, in, "parallel")
			results <- e
		}()
	}
	wg.Wait()
	close(results)
	successes, conflicts := 0, 0
	for e := range results {
		if e == nil {
			successes++
		} else if errors.Is(e, ErrConflict) {
			conflicts++
		} else {
			t.Fatal(e)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("writers=%d conflicts=%d", successes, conflicts)
	}
	var scheme string
	var priority int
	must(pool.QueryRow(ctx, `SELECT scheme,priority FROM app.application_store_listings WHERE id=$1`, storeID).Scan(&scheme, &priority))
	if scheme != "legacy://open" || priority != 7 {
		t.Fatal("legacy store fields overwritten")
	}
	cfg, err = svc.GetAdminPublicWebConfig(ctx, "token", appID)
	must(err)
	in.LockVersion = cfg.LockVersion
	in.Stores = []WebStore{{ID: uuid.New(), Platform: "ios", WebURL: "https://example.test"}}
	if _, err = svc.UpdateAdminPublicWebConfig(ctx, "token", appID, in, "invalid-store"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("foreign store accepted: %v", err)
	}
	unchanged, err := svc.GetAdminPublicWebConfig(ctx, "token", appID)
	must(err)
	if unchanged.LockVersion != cfg.LockVersion {
		t.Fatal("failed store update was not rolled back")
	}
	in.LockVersion = unchanged.LockVersion
	in.Stores = unchanged.Stores
	in.PromotionEnabled = nil // Old Admin clients omit this field and must preserve the stored value.
	for locale, translation := range in.Translations {
		translation.PromotionTitle = nil
		translation.PromotionDescription = nil
		translation.PromotionButtonLabel = nil
		in.Translations[locale] = translation
	}
	unchanged, err = svc.UpdateAdminPublicWebConfig(ctx, "token", appID, in, "legacy-client-save")
	must(err)
	if !unchanged.PromotionEnabled || unchanged.Translations["en-US"].PromotionTitle != "Get English App" || unchanged.Translations["en-US"].PromotionDescription != "Discover more in the app" || unchanged.Translations["en-US"].PromotionButtonLabel != "Download now" {
		t.Fatal("legacy request changed the stored promotion")
	}
	var audits int
	must(pool.QueryRow(ctx, `SELECT count(*) FROM audit.operation_logs WHERE tenant_id=$1 AND permission_code='app.public_web.update'`, tenant).Scan(&audits))
	if audits != 3 {
		t.Fatalf("audit count %d", audits)
	}
	auth.principal.Tenant.ID = uuid.New()
	if _, err = svc.GetAdminPublicWebConfig(ctx, "token", appID); !errors.Is(err, ErrAppNotFound) {
		t.Fatalf("cross tenant read: %v", err)
	}
	auth.principal.Tenant.ID = tenant
	auth.principal.Permissions = []string{"app.public_web.read"}
	if _, err = svc.UpdateAdminPublicWebConfig(ctx, "token", appID, in, "forbidden"); err == nil {
		t.Fatal("read permission authorized write")
	}
	public, err := svc.PublicWeb(ctx, appID, "en-US")
	must(err)
	if public.Name != "English App" || !public.PromotionEnabled || public.PromotionTitle != "Get English App" || public.PromotionDescription != "Discover more in the app" || public.PromotionButtonLabel != "Download now" {
		t.Fatal("locale not applied")
	}

	objects, e := storagerepo.NewLocalObjectStore(t.TempDir())
	must(e)
	svc.objects = objects
	file, page, revision := uuid.New(), uuid.New(), uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO storage.files(id,tenant_id,provider,bucket_name,object_key,original_name,media_type,extension,size_bytes,status,scan_status) VALUES($1,$2,'local','appkernia-local',$3,'test.png','image/png','png',4,'ready','clean')`, file, tenant, file.String())
	must(err)
	must(objects.Put(ctx, storage.ObjectRef{TenantID: tenant, Provider: "local", Bucket: "appkernia-local", Key: file.String()}, []byte("test")))
	_, err = pool.Exec(ctx, `INSERT INTO content.pages(id,app_id,tenant_id,slug,page_type) VALUES($1,$2,$3,'h5-image','custom')`, page, appID, tenant)
	must(err)
	_, err = pool.Exec(ctx, `INSERT INTO content.page_revisions(id,page_id,app_id,tenant_id,revision_number,content_hash,status) VALUES($1,$2,$3,$4,1,decode(repeat('ab',32),'hex'),'published')`, revision, page, appID, tenant)
	must(err)
	raw, _ := json.Marshal("![page image](/api/v1/public/content/assets/" + file.String() + ")")
	_, err = pool.Exec(ctx, `INSERT INTO content.page_revision_translations(revision_id,locale,title,body_format,body) VALUES($1,'zh-CN','Image','markdown',$2)`, revision, raw)
	must(err)
	_, err = pool.Exec(ctx, `UPDATE content.pages SET current_revision_id=$2,status='published' WHERE id=$1`, page, revision)
	must(err)
	_, _, reader, err := svc.OpenPublicWebAsset(ctx, appID, file)
	must(err)
	bytes, err := io.ReadAll(reader)
	must(err)
	must(reader.Close())
	if string(bytes) != "test" {
		t.Fatal("wrong image payload")
	}
	for _, sql := range []string{`UPDATE content.pages SET status='archived' WHERE id=$1`, `UPDATE content.pages SET status='draft' WHERE id=$1`} {
		_, err = pool.Exec(ctx, sql, page)
		must(err)
		if _, _, _, err = svc.OpenPublicWebAsset(ctx, appID, file); !errors.Is(err, ErrAppNotFound) {
			t.Fatalf("unpublished image: %v", err)
		}
	}
	_, err = pool.Exec(ctx, `UPDATE content.pages SET status='published' WHERE id=$1`, page)
	must(err)
	_, err = pool.Exec(ctx, `UPDATE storage.files SET scan_status='infected' WHERE id=$1`, file)
	must(err)
	if _, _, _, err = svc.OpenPublicWebAsset(ctx, appID, file); !errors.Is(err, ErrAppNotFound) {
		t.Fatalf("unsafe image: %v", err)
	}
	_, err = pool.Exec(ctx, `UPDATE app.applications SET status='disabled' WHERE id=$1`, appID)
	must(err)
	if _, err = svc.PublicWeb(ctx, appID, "en-US"); !errors.Is(err, ErrAppDisabled) {
		t.Fatalf("inactive app exposed: %v", err)
	}
}

func stringPointer(value string) *string { return &value }
