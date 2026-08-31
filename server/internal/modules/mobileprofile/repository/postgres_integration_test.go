package repository

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/appkernia/appkernia/server/internal/modules/mobileprofile/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresMobileReleaseAndNotificationTenantScope(t *testing.T) {
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
	repo := NewPostgres(pool)
	suffix := uuid.NewString()
	var tenantID, otherTenantID, appID, otherAppID, userID, otherUserID uuid.UUID
	if err = pool.QueryRow(ctx, `INSERT INTO iam.tenants(code,name) VALUES($1,'Mobile Test') RETURNING id`, "mobile-"+suffix).Scan(&tenantID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO iam.tenants(code,name) VALUES($1,'Other Test') RETURNING id`, "other-"+suffix).Scan(&otherTenantID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT id FROM app.applications WHERE tenant_id=$1 AND is_default`, tenantID).Scan(&appID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE app.applications SET appid=$2 WHERE id=$1`, appID, "__UNI__TEST"+strings.ReplaceAll(suffix, "-", "")); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT id FROM app.applications WHERE tenant_id=$1 AND is_default`, otherTenantID).Scan(&otherAppID); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM sys.mobile_release_publications WHERE app_id IN ($1,$2)`, appID, otherAppID)
		_, _ = pool.Exec(ctx, `DELETE FROM sys.mobile_releases WHERE app_id IN ($1,$2)`, appID, otherAppID)
	}()
	if err = pool.QueryRow(ctx, `INSERT INTO iam.users(email,display_name,status) VALUES($1,'Mobile User','active') RETURNING id`, "mobile-"+suffix+"@example.test").Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO iam.users(email,display_name,status) VALUES($1,'Other User','active') RETURNING id`, "other-"+suffix+"@example.test").Scan(&otherUserID); err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]uuid.UUID{{tenantID, userID}, {otherTenantID, otherUserID}} {
		if _, err = pool.Exec(ctx, `INSERT INTO iam.tenant_members(tenant_id,user_id,status) VALUES($1,$2,'active')`, pair[0], pair[1]); err != nil {
			t.Fatal(err)
		}
	}
	var ownMessageID, otherMessageID uuid.UUID
	if err = pool.QueryRow(ctx, `INSERT INTO notify.messages(tenant_id,title,body,body_format,status,published_at) VALUES($1,'Own','Own body','plain','published',now()) RETURNING id`, tenantID).Scan(&ownMessageID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO notify.messages(tenant_id,title,body,body_format,status,published_at) VALUES($1,'Other','Other body','plain','published',now()) RETURNING id`, otherTenantID).Scan(&otherMessageID); err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct{ tenant, app, message, user uuid.UUID }{{tenantID, appID, ownMessageID, userID}, {otherTenantID, otherAppID, otherMessageID, otherUserID}} {
		if _, err = pool.Exec(ctx, `INSERT INTO notify.recipients(tenant_id,app_id,message_id,user_id,delivery_status,delivered_at) VALUES($1,$2,$3,$4,'delivered',now())`, row.tenant, row.app, row.message, row.user); err != nil {
			t.Fatal(err)
		}
	}
	page, err := repo.Notifications(ctx, userID, tenantID, appID, "", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != ownMessageID {
		t.Fatalf("tenant scope leaked: %#v", page.Items)
	}
	unread, err := repo.UnreadCount(ctx, userID, tenantID, appID)
	if err != nil || unread != 1 {
		t.Fatalf("unread count=%d err=%v", unread, err)
	}
	if err = repo.MarkNotificationRead(ctx, userID, tenantID, appID, uuid.Nil, otherMessageID, "integration-test"); !errors.Is(err, domain.ErrNotificationNotFound) {
		t.Fatalf("cross-tenant read error=%v", err)
	}
	var sessionID uuid.UUID
	if err = pool.QueryRow(ctx, `INSERT INTO iam.sessions(user_id,tenant_id,audience,absolute_expires_at) VALUES($1,$2,'ak-mobile',now()+interval '1 hour') RETURNING id`, userID, tenantID).Scan(&sessionID); err != nil {
		t.Fatal(err)
	}
	updated, err := repo.MarkAllNotificationsRead(ctx, userID, tenantID, appID, sessionID, "integration-read-all")
	if err != nil || updated != 1 {
		t.Fatalf("mark all updated=%d err=%v", updated, err)
	}
	updated, err = repo.MarkAllNotificationsRead(ctx, userID, tenantID, appID, sessionID, "integration-read-all-idempotent")
	if err != nil || updated != 0 {
		t.Fatalf("idempotent mark all updated=%d err=%v", updated, err)
	}
	unread, err = repo.UnreadCount(ctx, userID, tenantID, appID)
	if err != nil || unread != 0 {
		t.Fatalf("unread count after mark all=%d err=%v", unread, err)
	}
	var otherRead bool
	if err = pool.QueryRow(ctx, `SELECT read_at IS NOT NULL FROM notify.recipients WHERE tenant_id=$1 AND app_id=$2 AND message_id=$3 AND user_id=$4`, otherTenantID, otherAppID, otherMessageID, otherUserID).Scan(&otherRead); err != nil || otherRead {
		t.Fatalf("cross-tenant notification changed=%t err=%v", otherRead, err)
	}
	created, err := repo.CreateRelease(ctx, appID, domain.Release{Platform: "ios", CurrentVersion: "2.0.0", MinimumVersion: "1.0.0", UpgradeURL: ptr("https://example.test/ios"), ReleaseNotes: map[string]string{"zh-CN": "说明", "en-US": "Notes"}, Active: true}, userID, "integration-test")
	if err != nil {
		t.Fatal(err)
	}
	active, err := repo.ActiveRelease(ctx, appID, "ios")
	if err != nil || active.ID != created.ID {
		t.Fatalf("active release=%#v err=%v", active, err)
	}
	created.Platform = "android"
	created.LockVersion = active.LockVersion
	if _, err = repo.UpdateRelease(ctx, appID, created, userID, "integration-test"); !errors.Is(err, domain.ErrReleaseConflict) {
		t.Fatalf("platform mutation error=%v", err)
	}
	if _, err = repo.ActiveRelease(ctx, appID, "harmony"); !errors.Is(err, domain.ErrReleaseNotFound) {
		t.Fatalf("missing release error=%v", err)
	}

	minimumNative := "1.0.0"
	wgtURL := "https://example.test/app.wgt"
	wgt, err := repo.CreateDraft(ctx, tenantID, appID, domain.Release{
		PackageType:          "wgt",
		Platforms:            []string{"android", "ios", "harmony"},
		Version:              "3.0.0",
		MinimumNativeVersion: &minimumNative,
		Titles:               map[string]string{"zh-CN": "WGT 更新", "en-US": "WGT update"},
		Contents:             map[string]string{"zh-CN": "三平台资源更新", "en-US": "Resource update for three platforms"},
		ExternalURL:          &wgtURL,
	}, userID, "integration-wgt-create")
	if err != nil || wgt.PublishStatus != "draft" {
		t.Fatalf("create WGT draft=%#v err=%v", wgt, err)
	}
	wgt, err = repo.Publish(ctx, tenantID, appID, wgt.ID, wgt.LockVersion, userID, "integration-wgt-publish")
	if err != nil || wgt.PublishStatus != "online" || len(wgt.PublishedPlatforms) != 3 {
		t.Fatalf("publish WGT=%#v err=%v", wgt, err)
	}
	if _, err = repo.UpdateDraft(ctx, tenantID, appID, wgt, userID, "integration-frozen-update"); !errors.Is(err, domain.ErrReleaseFrozen) {
		t.Fatalf("published WGT update error=%v", err)
	}
	if err = repo.Delete(ctx, tenantID, appID, []uuid.UUID{wgt.ID}, userID, "integration-frozen-delete"); !errors.Is(err, domain.ErrReleaseDeleteForbidden) {
		t.Fatalf("published WGT delete error=%v", err)
	}

	androidURL := "https://example.test/android.wgt"
	androidWGT, err := repo.CreateDraft(ctx, tenantID, appID, domain.Release{
		PackageType:          "wgt",
		Platforms:            []string{"android"},
		Version:              "4.0.0",
		MinimumNativeVersion: &minimumNative,
		Titles:               map[string]string{"zh-CN": "Android 更新", "en-US": "Android update"},
		Contents:             map[string]string{"zh-CN": "仅替换 Android", "en-US": "Replace Android only"},
		ExternalURL:          &androidURL,
	}, userID, "integration-partial-create")
	if err != nil {
		t.Fatal(err)
	}
	androidWGT, err = repo.Publish(ctx, tenantID, appID, androidWGT.ID, androidWGT.LockVersion, userID, "integration-partial-publish")
	if err != nil || androidWGT.PublishStatus != "online" {
		t.Fatalf("publish Android WGT=%#v err=%v", androidWGT, err)
	}
	wgt, err = repo.GetRelease(ctx, tenantID, appID, wgt.ID)
	if err != nil || wgt.PublishStatus != "partial" || len(wgt.PublishedPlatforms) != 2 {
		t.Fatalf("previous WGT partial state=%#v err=%v", wgt, err)
	}
	if _, err = repo.GetRelease(ctx, otherTenantID, otherAppID, wgt.ID); !errors.Is(err, domain.ErrReleaseNotFound) {
		t.Fatalf("cross-tenant release read error=%v", err)
	}

	lowerURL := "https://example.test/lower.wgt"
	lower, err := repo.CreateDraft(ctx, tenantID, appID, domain.Release{
		PackageType:          "wgt",
		Platforms:            []string{"android"},
		Version:              "3.5.0",
		MinimumNativeVersion: &minimumNative,
		Titles:               map[string]string{"zh-CN": "低版本", "en-US": "Lower version"},
		Contents:             map[string]string{"zh-CN": "不应发布", "en-US": "Must not publish"},
		ExternalURL:          &lowerURL,
	}, userID, "integration-lower-create")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Publish(ctx, tenantID, appID, lower.ID, lower.LockVersion, userID, "integration-lower-publish"); !errors.Is(err, domain.ErrReleaseVersionNotIncreasing) {
		t.Fatalf("lower WGT publish error=%v", err)
	}
	if err = repo.Delete(ctx, tenantID, appID, []uuid.UUID{lower.ID, wgt.ID}, userID, "integration-atomic-delete"); !errors.Is(err, domain.ErrReleaseDeleteForbidden) {
		t.Fatalf("mixed draft/published delete error=%v", err)
	}
	if _, err = repo.GetRelease(ctx, tenantID, appID, lower.ID); err != nil {
		t.Fatalf("atomic delete removed eligible draft: %v", err)
	}
	if err = repo.Delete(ctx, tenantID, appID, []uuid.UUID{lower.ID}, userID, "integration-draft-delete"); err != nil {
		t.Fatalf("delete draft: %v", err)
	}
	androidWGT, err = repo.Unpublish(ctx, tenantID, appID, androidWGT.ID, androidWGT.LockVersion, userID, "integration-unpublish")
	if err != nil || androidWGT.PublishStatus != "offline" {
		t.Fatalf("unpublish Android WGT=%#v err=%v", androidWGT, err)
	}
	androidWGT, err = repo.Publish(ctx, tenantID, appID, androidWGT.ID, androidWGT.LockVersion, userID, "integration-republish")
	if err != nil || androidWGT.PublishStatus != "online" {
		t.Fatalf("republish Android WGT=%#v err=%v", androidWGT, err)
	}
	invalidReleases := []struct {
		name          string
		version       string
		packageType   string
		minimumNative *string
		url           *string
		createEnv     string
	}{
		{name: "invalid semver", version: "1.0", packageType: "native_app", createEnv: "upgrade_center"},
		{name: "insecure URL", version: "1.0.0", packageType: "native_app", url: ptr("http://example.test/app"), createEnv: "upgrade_center"},
		{name: "invalid minimum native semver", version: "1.0.0", packageType: "wgt", minimumNative: ptr("1.0"), createEnv: "upgrade_center"},
		{name: "unsupported package type", version: "1.0.0", packageType: "archive", createEnv: "upgrade_center"},
		{name: "unsupported create environment", version: "1.0.0", packageType: "native_app", createEnv: "manual"},
	}
	for _, test := range invalidReleases {
		if _, insertErr := pool.Exec(ctx, `INSERT INTO sys.mobile_releases(
tenant_id,app_id,platform,current_version,minimum_version,upgrade_url,release_notes,active,
package_type,version,minimum_native_version,external_url,create_env)
VALUES($1,$2,'harmony',$3,'0.0.0',$4,'{}',false,$5,$3,$6,$4,$7)`, tenantID, appID,
			test.version, test.url, test.packageType, test.minimumNative, test.createEnv); insertErr == nil {
			t.Fatalf("database accepted %s", test.name)
		}
	}
}
func ptr(value string) *string { return &value }
