//go:build integration

package repository

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	iamrepo "github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	sessiondomain "github.com/appkernia/appkernia/server/internal/modules/sessionadmin/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresSessionTenantMaskingAndTransactionalRevoke(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	suffix := uuid.NewString()
	hash, err := iamapp.HashPassword("session integration password 2026!")
	if err != nil {
		t.Fatal(err)
	}
	identities := iamrepo.NewPostgres(pool)
	owner, tenant, err := identities.CreateIdentity(ctx, iamdomain.CreateIdentity{TenantCode: "session-source-" + suffix, TenantName: "Session Source", Email: "session-owner-" + suffix + "@example.test", DisplayName: "Session Owner", Locale: "zh-CN", PasswordHash: hash})
	if err != nil {
		t.Fatal(err)
	}
	other, otherTenant, err := identities.CreateIdentity(ctx, iamdomain.CreateIdentity{TenantCode: "session-other-" + suffix, TenantName: "Session Other", Email: "session-other-" + suffix + "@example.test", DisplayName: "Other", Locale: "en-US", PasswordHash: hash})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	var currentID, targetID, otherID uuid.UUID
	if err = pool.QueryRow(ctx, `INSERT INTO iam.sessions(user_id,tenant_id,audience,ip_address,user_agent,last_seen_at,absolute_expires_at) VALUES($1,$2,'ak-admin','203.0.113.45','raw-current-agent',$3,$4) RETURNING id`, owner.ID, tenant.ID, now, now.Add(time.Hour)).Scan(&currentID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO iam.sessions(user_id,tenant_id,audience,ip_address,user_agent,last_seen_at,absolute_expires_at) VALUES($1,$2,'ak-admin','198.51.100.27','raw-target-agent',$3,$4) RETURNING id`, owner.ID, tenant.ID, now.Add(-time.Minute), now.Add(time.Hour)).Scan(&targetID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO iam.sessions(user_id,tenant_id,audience,ip_address,user_agent,last_seen_at,absolute_expires_at) VALUES($1,$2,'ak-admin','192.0.2.99','raw-other-agent',$3,$4) RETURNING id`, other.ID, otherTenant.ID, now, now.Add(time.Hour)).Scan(&otherID); err != nil {
		t.Fatal(err)
	}
	tokenHash := sha256.Sum256([]byte(suffix))
	if _, err = pool.Exec(ctx, `INSERT INTO iam.refresh_tokens(session_id,token_hash,expires_at) VALUES($1,$2,$3)`, targetID, tokenHash[:], now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	repo := NewPostgres(pool)
	page, err := repo.List(ctx, tenant.ID, currentID, sessiondomain.Filter{FromAt: now.Add(-time.Hour), ToAt: now.Add(time.Hour), Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 {
		t.Fatalf("total=%d items=%#v", page.Total, page.Items)
	}
	for _, item := range page.Items {
		if item.ID == otherID || strings.Contains(item.UserHint, "session-owner-") || strings.Contains(item.IPHint, ".45") || strings.Contains(item.IPHint, ".27") {
			t.Fatalf("unsafe or cross-tenant item=%#v", item)
		}
		if item.ID == currentID && !item.Current {
			t.Fatalf("current session not marked: %#v", item)
		}
	}
	principal := sessiondomain.Principal{TenantID: tenant.ID, UserID: owner.ID, SessionID: currentID, RequestID: "session-revoke-" + suffix}
	if _, err = repo.Revoke(ctx, principal, otherID); !errors.Is(err, sessiondomain.ErrSessionAbsent) {
		t.Fatalf("cross-tenant revoke error=%v", err)
	}
	result, err := repo.Revoke(ctx, principal, targetID)
	if err != nil || !result.Revoked || result.Current {
		t.Fatalf("result=%#v error=%v", result, err)
	}
	var sessionStatus string
	var refreshRevoked *time.Time
	if err = pool.QueryRow(ctx, `SELECT s.status,rt.revoked_at FROM iam.sessions s JOIN iam.refresh_tokens rt ON rt.session_id=s.id WHERE s.id=$1`, targetID).Scan(&sessionStatus, &refreshRevoked); err != nil || sessionStatus != "revoked" || refreshRevoked == nil {
		t.Fatalf("status=%s refresh=%v error=%v", sessionStatus, refreshRevoked, err)
	}
	var auditCount int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit.operation_logs WHERE tenant_id=$1 AND request_id=$2 AND action_name='iam.session.revoke' AND after_data->>'revoked'='true'`, tenant.ID, principal.RequestID).Scan(&auditCount); err != nil || auditCount != 1 {
		t.Fatalf("audit=%d error=%v", auditCount, err)
	}
	if _, err = repo.Revoke(ctx, principal, targetID); !errors.Is(err, sessiondomain.ErrSessionAbsent) {
		t.Fatalf("second revoke error=%v", err)
	}
}
