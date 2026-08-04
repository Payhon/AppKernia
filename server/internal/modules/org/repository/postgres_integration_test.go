//go:build integration

package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	iamrepo "github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	orgdomain "github.com/appkernia/appkernia/server/internal/modules/org/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOrganizationRepositoryTenantCycleOccupancyAndAudit(t *testing.T) {
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
	suffix := uuid.New().String()
	identityRepo := iamrepo.NewPostgres(pool)
	user, tenant, err := identityRepo.CreateIdentity(ctx, iamdomain.CreateIdentity{TenantCode: "org-" + suffix, TenantName: "Org Integration", Email: "org-" + suffix + "@example.test", DisplayName: "Org User", Locale: "zh-CN", PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$integration$integration"})
	if err != nil {
		t.Fatalf("CreateIdentity() error = %v", err)
	}
	_, otherTenant, err := identityRepo.CreateIdentity(ctx, iamdomain.CreateIdentity{TenantCode: "org-other-" + suffix, TenantName: "Other Org", Email: "org-other-" + suffix + "@example.test", DisplayName: "Other User", Locale: "en-US", PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$integration$integration"})
	if err != nil {
		t.Fatalf("CreateIdentity(other) error = %v", err)
	}
	session, err := db.New(pool).CreateSession(ctx, db.CreateSessionParams{UserID: user.ID, TenantID: &tenant.ID, Audience: "ak-admin", AbsoluteExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true}, IdleExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(30 * time.Minute), Valid: true}})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	principal := orgdomain.Principal{TenantID: tenant.ID, UserID: user.ID, SessionID: session.ID, RequestID: "org-integration"}
	repository := NewPostgres(pool)
	root, err := repository.CreateUnit(ctx, principal, orgdomain.UnitInput{Code: "ROOT", Name: "Root", UnitType: "company", Status: "active"})
	if err != nil {
		t.Fatalf("CreateUnit(root) error = %v", err)
	}
	child, err := repository.CreateUnit(ctx, principal, orgdomain.UnitInput{ParentID: &root.ID, Code: "CHILD", Name: "Child", UnitType: "department", Status: "active"})
	if err != nil {
		t.Fatalf("CreateUnit(child) error = %v", err)
	}
	if _, err = repository.MoveUnit(ctx, principal, root.ID, orgdomain.UnitMove{ParentID: &child.ID}); !errors.Is(err, orgdomain.ErrUnitCycle) {
		t.Fatalf("MoveUnit(cycle) error = %v", err)
	}
	occupancy, err := repository.DeleteUnit(ctx, principal, root.ID)
	if !errors.Is(err, orgdomain.ErrUnitOccupied) || occupancy.ChildCount != 1 {
		t.Fatalf("DeleteUnit(parent) occupancy=%#v error=%v", occupancy, err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO org.user_units (tenant_id,user_id,unit_id,is_primary) VALUES ($1,$2,$3,true)`, tenant.ID, user.ID, child.ID); err != nil {
		t.Fatalf("insert user unit: %v", err)
	}
	occupancy, err = repository.DeleteUnit(ctx, principal, child.ID)
	if !errors.Is(err, orgdomain.ErrUnitOccupied) || occupancy.MemberCount != 1 {
		t.Fatalf("DeleteUnit(member) occupancy=%#v error=%v", occupancy, err)
	}
	units, err := repository.ListUnits(ctx, otherTenant.ID)
	if err != nil || len(units) != 0 {
		t.Fatalf("ListUnits(other tenant) len=%d error=%v", len(units), err)
	}
	position, err := repository.CreatePosition(ctx, principal, orgdomain.PositionInput{Code: "ENGINEER", Name: "Engineer", Status: "active"})
	if err != nil {
		t.Fatalf("CreatePosition() error = %v", err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO org.user_positions (tenant_id,user_id,position_id,unit_id,is_primary) VALUES ($1,$2,$3,$4,true)`, tenant.ID, user.ID, position.ID, child.ID); err != nil {
		t.Fatalf("insert user position: %v", err)
	}
	count, err := repository.DeletePosition(ctx, principal, position.ID)
	if !errors.Is(err, orgdomain.ErrPositionOccupied) || count != 1 {
		t.Fatalf("DeletePosition() count=%d error=%v", count, err)
	}
	positions, err := repository.ListPositions(ctx, tenant.ID, "engi", "active", &child.ID)
	if err != nil || len(positions) != 1 || positions[0].MemberCount != 1 {
		t.Fatalf("ListPositions() values=%#v error=%v", positions, err)
	}
	var auditCount int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit.operation_logs WHERE tenant_id=$1 AND user_id=$2 AND module_code='org' AND request_id='org-integration'`, tenant.ID, user.ID).Scan(&auditCount); err != nil {
		t.Fatalf("count audit: %v", err)
	}
	if auditCount != 3 {
		t.Fatalf("audit count=%d, want 3 successful writes", auditCount)
	}
}
