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
	for _, name := range []string{"CountMobileUnreadNotifications", "ListMobileNotifications"} {
		start := strings.Index(sql, "-- name: "+name)
		end := strings.Index(sql[start+1:], "-- name:")
		if start < 0 || end < 0 {
			t.Fatalf("%s query not found", name)
		}
		query := sql[start : start+1+end]
		for _, required := range []string{"r.tenant_id = sqlc.arg(tenant_id)", "r.user_id = sqlc.arg(user_id)", "r.delivery_status = 'delivered'", "m.status = 'published'", "m.deleted_at IS NULL", "r.archived_at IS NULL"} {
			if !strings.Contains(query, required) {
				t.Fatalf("%s query missing %q", name, required)
			}
		}
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
