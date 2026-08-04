//go:build integration

package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	iamrepo "github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	userdomain "github.com/appkernia/appkernia/server/internal/modules/useradmin/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestUserAdminRepositoryTenantAssignmentsStatusPasswordAndAudit(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	defer pool.Close()
	suffix := uuid.NewString()
	identityRepo := iamrepo.NewPostgres(pool)
	actor, tenant, err := identityRepo.CreateIdentity(ctx, iamdomain.CreateIdentity{TenantCode: "users-" + suffix, TenantName: "Users Integration", Email: "actor-" + suffix + "@example.test", DisplayName: "Actor", Locale: "zh-CN", PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$integration$integration"})
	if err != nil {
		t.Fatalf("CreateIdentity(actor) error = %v", err)
	}
	_, otherTenant, err := identityRepo.CreateIdentity(ctx, iamdomain.CreateIdentity{TenantCode: "users-other-" + suffix, TenantName: "Other Users", Email: "other-" + suffix + "@example.test", DisplayName: "Other", Locale: "en-US", PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$integration$integration"})
	if err != nil {
		t.Fatalf("CreateIdentity(other) error = %v", err)
	}
	session, err := db.New(pool).CreateSession(ctx, db.CreateSessionParams{UserID: actor.ID, TenantID: &tenant.ID, Audience: "ak-admin", AbsoluteExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true}, IdleExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(30 * time.Minute), Valid: true}})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	principal := userdomain.Principal{TenantID: tenant.ID, UserID: actor.ID, SessionID: session.ID, RequestID: "users-integration"}
	repository := NewPostgres(pool)
	hash, err := iamapp.HashPassword("Temporary!Password2026")
	if err != nil {
		t.Fatal(err)
	}
	managed, err := repository.CreateUser(ctx, principal, userdomain.CreateInput{Email: "managed-" + suffix + "@example.test", DisplayName: "Managed User", Locale: "en-US", TimeZone: "UTC"}, hash)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if _, err = repository.GetUser(ctx, otherTenant.ID, managed.ID); !errors.Is(err, userdomain.ErrNotFound) {
		t.Fatalf("GetUser(other tenant) error = %v", err)
	}
	page, err := repository.ListUsers(ctx, tenant.ID, userdomain.Filters{Query: "Managed", Status: "active", Page: 1, PageSize: 20, Sort: "name_asc"})
	if err != nil || page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("ListUsers() page=%#v error=%v", page, err)
	}
	var roleID uuid.UUID
	if err = pool.QueryRow(ctx, `INSERT INTO iam.roles(tenant_id,code,name,role_type,data_scope,status) VALUES($1,'member','Member','custom','self','active') RETURNING id`, tenant.ID).Scan(&roleID); err != nil {
		t.Fatalf("get role: %v", err)
	}
	managed, err = repository.ReplaceRoles(ctx, principal, managed.ID, []uuid.UUID{roleID})
	if err != nil || len(managed.Roles) != 1 {
		t.Fatalf("ReplaceRoles() user=%#v error=%v", managed, err)
	}
	var otherRoleID uuid.UUID
	if err = pool.QueryRow(ctx, `INSERT INTO iam.roles(tenant_id,code,name,role_type,data_scope,status) VALUES($1,'member','Member','custom','self','active') RETURNING id`, otherTenant.ID).Scan(&otherRoleID); err != nil {
		t.Fatal(err)
	}
	if _, err = repository.ReplaceRoles(ctx, principal, managed.ID, []uuid.UUID{otherRoleID}); !errors.Is(err, userdomain.ErrRoleInvalid) {
		t.Fatalf("ReplaceRoles(other tenant) error=%v", err)
	}
	var unitID, positionID uuid.UUID
	if err = pool.QueryRow(ctx, `INSERT INTO org.units(tenant_id,code,name,unit_type,status) VALUES($1,$2,'Managed Department','department','active') RETURNING id`, tenant.ID, "UNIT-"+suffix).Scan(&unitID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO org.positions(tenant_id,code,name,status) VALUES($1,$2,'Managed Position','active') RETURNING id`, tenant.ID, "POS-"+suffix).Scan(&positionID); err != nil {
		t.Fatal(err)
	}
	managed, err = repository.ReplaceAssignments(ctx, principal, managed.ID, userdomain.AssignmentInput{UnitIDs: []uuid.UUID{unitID}, PrimaryUnitID: &unitID, PositionIDs: []uuid.UUID{positionID}, PrimaryPositionID: &positionID})
	if err != nil || len(managed.Units) != 1 || len(managed.Positions) != 1 {
		t.Fatalf("ReplaceAssignments() user=%#v error=%v", managed, err)
	}
	managed, err = repository.SetMemberStatus(ctx, principal, managed.ID, "suspended")
	if err != nil || managed.Status != "disabled" {
		t.Fatalf("SetMemberStatus() user=%#v error=%v", managed, err)
	}
	managed, err = repository.SetMemberStatus(ctx, principal, managed.ID, "active")
	if err != nil || managed.Status != "active" {
		t.Fatalf("SetMemberStatus(enable) user=%#v error=%v", managed, err)
	}
	newHash, _ := iamapp.HashPassword("Temporary!Password2027")
	revoked, err := repository.ResetPassword(ctx, principal, managed.ID, newHash)
	if err != nil || revoked != 0 {
		t.Fatalf("ResetPassword() revoked=%d error=%v", revoked, err)
	}
	var forced bool
	if err = pool.QueryRow(ctx, `SELECT force_password_change FROM iam.user_credentials WHERE user_id=$1`, managed.ID).Scan(&forced); err != nil || !forced {
		t.Fatalf("force_password_change=%v error=%v", forced, err)
	}
	var auditCount int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit.operation_logs WHERE tenant_id=$1 AND request_id='users-integration'`, tenant.ID).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 6 {
		t.Fatalf("audit count=%d, want 6", auditCount)
	}
}
