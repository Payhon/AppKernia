//go:build integration

package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	accessdomain "github.com/appkernia/appkernia/server/internal/modules/accessadmin/domain"
	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	iamrepo "github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresAccessControlLifecycleAndHierarchyGuards(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	defer pool.Close()
	suffix := uuid.NewString()
	hash, err := iamapp.HashPassword("access integration password 2026!")
	if err != nil {
		t.Fatal(err)
	}
	identities := iamrepo.NewPostgres(pool)
	owner, tenant, err := identities.CreateIdentity(ctx, iamdomain.CreateIdentity{TenantCode: "access-source-" + suffix, TenantName: "Access Source", Email: "access-owner-" + suffix + "@example.test", DisplayName: "Access Owner", Locale: "zh-CN", PasswordHash: hash})
	if err != nil {
		t.Fatalf("create identity: %v", err)
	}
	_, other, err := identities.CreateIdentity(ctx, iamdomain.CreateIdentity{TenantCode: "access-other-" + suffix, TenantName: "Access Other", Email: "access-other-" + suffix + "@example.test", DisplayName: "Other", Locale: "en-US", PasswordHash: hash})
	if err != nil {
		t.Fatalf("create other identity: %v", err)
	}
	repository := NewPostgres(pool)
	principal := accessdomain.Principal{TenantID: tenant.ID, UserID: owner.ID, RequestID: "access-" + suffix}
	role, err := repository.CreateRole(ctx, principal, accessdomain.RoleInput{Code: "support-" + suffix, Name: "Support", Status: "active"})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}
	if _, err = repository.UpdateRole(ctx, accessdomain.Principal{TenantID: other.ID, UserID: owner.ID}, role.ID, accessdomain.RoleInput{Code: role.Code, Name: "Cross", Status: "active"}); !errors.Is(err, accessdomain.ErrNotFound) {
		t.Fatalf("cross tenant role error=%v", err)
	}
	var permissionID, menuID, unitID uuid.UUID
	if err = pool.QueryRow(ctx, `SELECT id FROM iam.permissions WHERE code='iam.user.read'`).Scan(&permissionID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT id FROM sys.menus WHERE tenant_id IS NULL AND code='system.users.accounts' AND deleted_at IS NULL`).Scan(&menuID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO org.units(tenant_id,code,name,unit_type,status) VALUES($1,$2,'Support','department','active') RETURNING id`, tenant.ID, "support-"+suffix).Scan(&unitID); err != nil {
		t.Fatal(err)
	}
	role, err = repository.ReplaceRolePermissions(ctx, principal, role.ID, []uuid.UUID{permissionID})
	if err != nil || len(role.PermissionIDs) != 1 {
		t.Fatalf("replace permissions=%#v error=%v", role, err)
	}
	role, err = repository.ReplaceRoleMenus(ctx, principal, role.ID, []uuid.UUID{menuID})
	if err != nil || len(role.MenuIDs) != 1 {
		t.Fatalf("replace menus=%#v error=%v", role, err)
	}
	role, err = repository.ReplaceRoleDataScope(ctx, principal, role.ID, "custom", []uuid.UUID{unitID})
	if err != nil || role.DataScope != "custom" || len(role.ScopeUnitIDs) != 1 {
		t.Fatalf("replace scope=%#v error=%v", role, err)
	}
	root, err := repository.CreateMenu(ctx, principal, menuInput("custom.root."+suffix, "Root", nil))
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	child, err := repository.CreateMenu(ctx, principal, menuInput("custom.child."+suffix, "Child", &root.ID))
	if err != nil {
		t.Fatalf("create child: %v", err)
	}
	grandchild, err := repository.CreateMenu(ctx, principal, menuInput("custom.grandchild."+suffix, "Grandchild", &child.ID))
	if err != nil {
		t.Fatalf("create grandchild: %v", err)
	}
	if _, err = repository.CreateMenu(ctx, principal, menuInput("custom.too-deep."+suffix, "Too deep", &grandchild.ID)); !errors.Is(err, accessdomain.ErrMenuDepth) {
		t.Fatalf("depth error=%v", err)
	}
	if _, err = repository.MoveMenu(ctx, principal, root.ID, accessdomain.MenuMove{ParentID: &child.ID}); !errors.Is(err, accessdomain.ErrMenuCycle) {
		t.Fatalf("cycle error=%v", err)
	}
	bad := menuInput("custom.bad-component."+suffix, "Bad component", nil)
	bad.Type = "page"
	bad.Path = "/bad"
	bad.ComponentKey = "arbitrary.dynamic.component"
	if _, err = repository.CreateMenu(ctx, principal, bad); !errors.Is(err, accessdomain.ErrComponentKey) {
		t.Fatalf("component key error=%v", err)
	}
	menus, err := repository.ListMenus(ctx, tenant.ID)
	if err != nil || len(menus) == 0 {
		t.Fatalf("list menus count=%d error=%v", len(menus), err)
	}
	var audits int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit.operation_logs WHERE request_id=$1 AND succeeded`, principal.RequestID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if audits != 7 {
		t.Fatalf("audit count=%d want=7", audits)
	}
}

func menuInput(code, title string, parent *uuid.UUID) accessdomain.MenuInput {
	return accessdomain.MenuInput{ParentID: parent, Code: code, Title: title, I18nKey: "menu.custom", Type: "directory", OpenMode: "same_tab", Status: "active"}
}
