package repository

import (
	"os"
	"strings"
	"testing"
)

func TestNotificationQueryScopesTenantUserAndDeliveryState(t *testing.T) {
	raw, err := os.ReadFile("../../../../db/queries/mobile_profile.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(raw)
	for _, name := range []string{"CountMobileUnreadNotifications", "ListMobileNotifications", "MarkAllMobileNotificationsRead"} {
		start := strings.Index(sql, "-- name: "+name)
		if start < 0 {
			t.Fatalf("%s query not found", name)
		}
		query := sql[start:]
		if end := strings.Index(query[1:], "-- name:"); end >= 0 {
			query = query[:1+end]
		}
		for _, required := range []string{"r.tenant_id = sqlc.arg(tenant_id)", "r.user_id = sqlc.arg(user_id)", "r.delivery_status = 'delivered'", "m.status = 'published'", "m.deleted_at IS NULL", "r.archived_at IS NULL"} {
			if !strings.Contains(query, required) {
				t.Fatalf("%s query missing %q", name, required)
			}
		}
	}
	bulkStart := strings.Index(sql, "-- name: MarkAllMobileNotificationsRead")
	if bulkStart < 0 || !strings.Contains(sql[bulkStart:], "r.read_at IS NULL") || !strings.Contains(sql[bulkStart:], "m.app_id = r.app_id") {
		t.Fatal("bulk notification read query must be idempotent and app scoped")
	}
}

func TestMobileProfileMigrationHasMatchedUpDownObjects(t *testing.T) {
	up, err := os.ReadFile("../../../../../blueprint/backend/db/migrations/000012_mobile_profile.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../../../../blueprint/backend/db/migrations/000012_mobile_profile.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"iam.user_preferences", "sys.mobile_releases"} {
		if !strings.Contains(string(up), "CREATE TABLE "+table) || !strings.Contains(string(down), "DROP TABLE IF EXISTS "+table) {
			t.Fatalf("migration pair missing %s", table)
		}
	}
	for _, constraint := range []string{
		"ck_mobile_release_versions",
		"ck_mobile_release_upgrade_url",
		"ck_mobile_release_active_upgrade_url",
		"ck_mobile_release_notes",
	} {
		if !strings.Contains(string(up), "CONSTRAINT "+constraint) {
			t.Fatalf("mobile release migration missing %s", constraint)
		}
	}
}
