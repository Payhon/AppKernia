package repository

import (
	"context"
	"errors"
	"os"
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
	var tenantID, otherTenantID, userID, otherUserID uuid.UUID
	if err = pool.QueryRow(ctx, `INSERT INTO iam.tenants(code,name) VALUES($1,'Mobile Test') RETURNING id`, "mobile-"+suffix).Scan(&tenantID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO iam.tenants(code,name) VALUES($1,'Other Test') RETURNING id`, "other-"+suffix).Scan(&otherTenantID); err != nil {
		t.Fatal(err)
	}
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
	for _, row := range []struct{ tenant, message, user uuid.UUID }{{tenantID, ownMessageID, userID}, {otherTenantID, otherMessageID, otherUserID}} {
		if _, err = pool.Exec(ctx, `INSERT INTO notify.recipients(tenant_id,message_id,user_id,delivery_status,delivered_at) VALUES($1,$2,$3,'delivered',now())`, row.tenant, row.message, row.user); err != nil {
			t.Fatal(err)
		}
	}
	page, err := repo.Notifications(ctx, userID, tenantID, "", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != ownMessageID {
		t.Fatalf("tenant scope leaked: %#v", page.Items)
	}
	unread, err := repo.UnreadCount(ctx, userID, tenantID)
	if err != nil || unread != 1 {
		t.Fatalf("unread count=%d err=%v", unread, err)
	}
	if err = repo.MarkNotificationRead(ctx, userID, tenantID, uuid.Nil, otherMessageID, "integration-test"); !errors.Is(err, domain.ErrNotificationNotFound) {
		t.Fatalf("cross-tenant read error=%v", err)
	}
	created, err := repo.CreateRelease(ctx, domain.Release{Platform: "ios", CurrentVersion: "2.0.0", MinimumVersion: "1.0.0", UpgradeURL: ptr("https://example.test/ios"), ReleaseNotes: map[string]string{"zh-CN": "说明", "en-US": "Notes"}, Active: true}, userID, "integration-test")
	if err != nil {
		t.Fatal(err)
	}
	active, err := repo.ActiveRelease(ctx, "ios")
	if err != nil || active.ID != created.ID {
		t.Fatalf("active release=%#v err=%v", active, err)
	}
	created.Platform = "android"
	created.LockVersion = active.LockVersion
	if _, err = repo.UpdateRelease(ctx, created, userID, "integration-test"); !errors.Is(err, domain.ErrReleaseConflict) {
		t.Fatalf("platform mutation error=%v", err)
	}
	if _, err = repo.ActiveRelease(ctx, "harmony"); !errors.Is(err, domain.ErrReleaseNotFound) {
		t.Fatalf("missing release error=%v", err)
	}
	invalidReleases := []struct {
		name    string
		current string
		minimum string
		url     *string
		notes   string
		active  bool
	}{
		{name: "invalid semver", current: "1.0", minimum: "1.0.0", notes: `{"zh-CN":"说明","en-US":"Notes"}`},
		{name: "insecure URL", current: "1.0.0", minimum: "1.0.0", url: ptr("http://example.test/app"), notes: `{"zh-CN":"说明","en-US":"Notes"}`},
		{name: "active without URL", current: "1.0.0", minimum: "1.0.0", notes: `{"zh-CN":"说明","en-US":"Notes"}`, active: true},
		{name: "missing localized notes", current: "1.0.0", minimum: "1.0.0", notes: `{"zh-CN":"","en-US":"Notes"}`},
	}
	for _, test := range invalidReleases {
		if _, insertErr := pool.Exec(ctx, `INSERT INTO sys.mobile_releases(platform,current_version,minimum_version,upgrade_url,release_notes,active) VALUES('harmony',$1,$2,$3,$4,$5)`, test.current, test.minimum, test.url, test.notes, test.active); insertErr == nil {
			t.Fatalf("database accepted %s", test.name)
		}
	}
}
func ptr(value string) *string { return &value }
