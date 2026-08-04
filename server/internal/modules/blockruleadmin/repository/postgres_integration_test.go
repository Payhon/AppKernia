//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	blocks "github.com/appkernia/appkernia/server/internal/modules/blockruleadmin/domain"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	iamrepo "github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestBlockRuleLifecycleIsTenantScopedRedactedAndAudited(t *testing.T) {
	dsn := os.Getenv("AK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, e := pgxpool.New(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer pool.Close()
	suffix := uuid.NewString()
	user, tenant, e := iamrepo.NewPostgres(pool).CreateIdentity(ctx, iamdomain.CreateIdentity{TenantCode: "block-" + suffix, TenantName: "Block Integration", Email: "block-" + suffix + "@example.test", DisplayName: "Block Owner", Locale: "en-US", PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$integration$integration"})
	if e != nil {
		t.Fatal(e)
	}
	session, e := db.New(pool).CreateSession(ctx, db.CreateSessionParams{UserID: user.ID, TenantID: &tenant.ID, Audience: "ak-admin", AbsoluteExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true}, IdleExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true}})
	if e != nil {
		t.Fatal(e)
	}
	repo := NewPostgres(pool)
	p := blocks.Principal{TenantID: tenant.ID, UserID: user.ID, SessionID: session.ID, RequestID: "block-integration", UserAgent: "integration"}
	start := time.Now().UTC()
	rawSubject := "203.0.113.91"
	created, e := repo.Create(ctx, p, blocks.CreateInput{SubjectType: "ip", SubjectValue: rawSubject, Action: "deny", Reason: "integration", StartsAt: &start, Status: "active"})
	if e != nil {
		t.Fatal(e)
	}
	encoded, _ := json.Marshal(created)
	if created.SubjectHint == rawSubject || strings.Contains(string(encoded), rawSubject) {
		t.Fatalf("raw subject leaked: %s", encoded)
	}
	page, e := repo.List(ctx, tenant.ID, blocks.Filter{SubjectHint: created.SubjectHint, Page: 1, PageSize: 20})
	if e != nil || page.Total != 1 {
		t.Fatalf("page=%#v err=%v", page, e)
	}
	other, e := repo.List(ctx, uuid.New(), blocks.Filter{Page: 1, PageSize: 20})
	if e != nil || other.Total != 0 {
		t.Fatalf("cross tenant page=%#v err=%v", other, e)
	}
	updated, e := repo.Update(ctx, p, created.ID, blocks.UpdateInput{Action: "challenge", Reason: "updated", StartsAt: start, Status: "disabled"})
	if e != nil || updated.Status != "disabled" {
		t.Fatalf("updated=%#v err=%v", updated, e)
	}
	if _, e = repo.Revoke(ctx, p, created.ID); e != nil {
		t.Fatal(e)
	}
	var audits int
	if e = pool.QueryRow(ctx, `SELECT count(*) FROM audit.operation_logs WHERE tenant_id=$1 AND resource_id=$2 AND action_name IN ('iam.block_rule.create','iam.block_rule.update','iam.block_rule.delete') AND after_data::text NOT LIKE '%'||$3||'%'`, tenant.ID, created.ID.String(), rawSubject).Scan(&audits); e != nil {
		t.Fatal(e)
	}
	if audits != 3 {
		t.Fatalf("safe audits=%d", audits)
	}
}
