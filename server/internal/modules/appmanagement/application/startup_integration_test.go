//go:build integration

package application

import (
	"context"
	"errors"
	"os"
	"testing"

	iam "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestStartupDraftPublishAndPublicProjection(t *testing.T) {
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
	var tenantID, userID uuid.UUID
	if err = pool.QueryRow(ctx, `INSERT INTO iam.tenants(code,name) VALUES($1,'Startup Integration') RETURNING id`, "startup-"+suffix).Scan(&tenantID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO iam.users(email,display_name,status) VALUES($1,'Startup Admin','active') RETURNING id`, "startup-"+suffix+"@example.test").Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO iam.tenant_members(tenant_id,user_id,status) VALUES($1,$2,'active')`, tenantID, userID); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM storage.file_usages WHERE tenant_id=$1`, tenantID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM app.applications WHERE tenant_id=$1`, tenantID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM storage.files WHERE tenant_id=$1`, tenantID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM iam.tenant_members WHERE tenant_id=$1`, tenantID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM iam.tenants WHERE id=$1`, tenantID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM iam.users WHERE id=$1`, userID)
	}()

	files := []uuid.UUID{uuid.New(), uuid.New()}
	for index, fileID := range files {
		if _, err = pool.Exec(ctx, `INSERT INTO storage.files(id,tenant_id,provider,bucket_name,object_key,original_name,media_type,extension,size_bytes,status,scan_status)
VALUES($1,$2,'local','startup-test',$3,$4,'image/png','png',128,'ready','clean')`, fileID, tenantID, "startup/"+fileID.String(), "slide-"+string(rune('a'+index))+".png"); err != nil {
			t.Fatal(err)
		}
	}
	auth := &adminIntegrationAuthenticator{principal: iam.AuthenticatedContext{AuthContext: iam.AuthContext{
		User: iam.User{ID: userID, Status: "active"}, Tenant: iam.Tenant{ID: tenantID, Status: "active"},
		Permissions: []string{"app.application.read", "app.application.create", "app.application.update", "app.onboarding.publish"},
	}}}
	service := NewService(pool, auth)
	input := AdminAppInput{
		AppID: "__UNI__STARTUP" + suffix[:8], AppType: "uni_app_x", Name: "Startup App", DefaultLocale: "zh-CN",
		RegistrationVerification: "none", OwnerType: "tenant", OwnerID: &tenantID,
		Startup: &StartupInput{OnboardingEnabled: true, Translations: map[string]StartupTranslation{
			"zh-CN": {DisplayName: "启动应用", Subtitle: "中文副标题"}, "en-US": {DisplayName: "Startup App", Subtitle: "English subtitle"},
		}, DraftSlides: []StartupSlide{{Assets: map[string]StartupSlideAsset{
			"zh-CN": {FileID: files[0], AccessibilityLabel: "中文启动介绍"}, "en-US": {FileID: files[1], AccessibilityLabel: "English onboarding"},
		}}}},
	}
	created, err := service.CreateAdminApp(ctx, "admin-token", input)
	if err != nil {
		t.Fatal(err)
	}
	if !created.Startup.OnboardingEnabled || len(created.Startup.DraftSlides) != 1 || !created.Startup.DraftChanged {
		t.Fatalf("startup draft=%#v", created.Startup)
	}
	beforePublish, err := service.PublicStartup(ctx, created, "en-US")
	if err != nil {
		t.Fatal(err)
	}
	if beforePublish.OnboardingEnabled || len(beforePublish.Slides) != 0 {
		t.Fatalf("draft leaked publicly: %#v", beforePublish)
	}

	published, err := service.PublishOnboarding(ctx, "admin-token", created.ID, 0, "startup-integration")
	if err != nil {
		t.Fatal(err)
	}
	if published.Startup.PublishedVersion != 1 || published.Startup.DraftChanged || published.Startup.PublishedAt == nil {
		t.Fatalf("published startup=%#v", published.Startup)
	}
	if _, err = service.PublishOnboarding(ctx, "admin-token", created.ID, 0, "stale-publish"); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale publish error=%v", err)
	}
	public, err := service.PublicStartup(ctx, published, "en-US")
	if err != nil {
		t.Fatal(err)
	}
	if !public.OnboardingEnabled || public.PublishedVersion != 1 || len(public.Slides) != 1 || public.Slides[0].AccessibilityLabel != "English onboarding" {
		t.Fatalf("public startup=%#v", public)
	}
	var usages int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM storage.file_usages WHERE tenant_id=$1 AND module_code='app' AND entity_type IN ('application_onboarding_draft','application_onboarding_revision')`, tenantID).Scan(&usages); err != nil {
		t.Fatal(err)
	}
	if usages != 4 {
		t.Fatalf("startup file usages=%d", usages)
	}
	if _, err = pool.Exec(ctx, `UPDATE app.application_startup_configs SET onboarding_enabled=false WHERE tenant_id=$1 AND app_id=$2`, tenantID, created.ID); err != nil {
		t.Fatal(err)
	}
	disabled, err := service.PublicStartup(ctx, published, "en-US")
	if err != nil || disabled.OnboardingEnabled || len(disabled.Slides) != 0 {
		t.Fatalf("disabled public startup=%#v err=%v", disabled, err)
	}
}
