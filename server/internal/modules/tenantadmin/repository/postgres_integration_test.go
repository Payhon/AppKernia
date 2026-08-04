//go:build integration

package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	iamrepo "github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	tenantdomain "github.com/appkernia/appkernia/server/internal/modules/tenantadmin/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresTenantLifecycleIsScopedAndAudited(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	defer pool.Close()

	suffix := uuid.NewString()
	hash, err := iamapp.HashPassword("tenant integration password 2026!")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	identities := iamrepo.NewPostgres(pool)
	owner, sourceTenant, err := identities.CreateIdentity(ctx, iamdomain.CreateIdentity{TenantCode: "tenant-source-" + suffix, TenantName: "Source Tenant", Email: "tenant-owner-" + suffix + "@example.test", DisplayName: "Tenant Owner", Locale: "zh-CN", PasswordHash: hash})
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	member, _, err := identities.CreateIdentity(ctx, iamdomain.CreateIdentity{TenantCode: "tenant-foreign-" + suffix, TenantName: "Foreign Tenant", Email: "tenant-member-" + suffix + "@example.test", DisplayName: "Tenant Member", Locale: "en-US", PasswordHash: hash})
	if err != nil {
		t.Fatalf("create member identity: %v", err)
	}

	repository := NewPostgres(pool)
	principal := tenantdomain.Principal{TenantID: sourceTenant.ID, UserID: owner.ID, RequestID: "tenant-create-" + suffix}
	created, err := repository.Create(ctx, principal, tenantdomain.CreateInput{Code: "created-" + suffix, Name: "Created Tenant"})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	if _, err = repository.Get(ctx, sourceTenant.ID, created.ID); !errors.Is(err, tenantdomain.ErrNotFound) {
		t.Fatalf("cross-tenant Get() error = %v", err)
	}
	page, err := repository.List(ctx, sourceTenant.ID, tenantdomain.Filters{Page: 1, PageSize: 20})
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != sourceTenant.ID {
		t.Fatalf("source scoped List() = %#v, error = %v", page, err)
	}

	createdPrincipal := tenantdomain.Principal{TenantID: created.ID, UserID: owner.ID, RequestID: "tenant-member-" + suffix}
	added, err := repository.AddMember(ctx, createdPrincipal, created.ID, tenantdomain.AddMemberInput{Email: "tenant-member-" + suffix + "@example.test"})
	if err != nil || added.UserID != member.ID {
		t.Fatalf("add member = %#v, error = %v", added, err)
	}
	if _, err = repository.SetMemberStatus(ctx, createdPrincipal, created.ID, owner.ID, "suspended"); !errors.Is(err, tenantdomain.ErrLastAdmin) {
		t.Fatalf("last admin status error = %v", err)
	}
	updated, err := repository.SetMemberStatus(ctx, createdPrincipal, created.ID, member.ID, "suspended")
	if err != nil || updated.Status != "suspended" {
		t.Fatalf("suspend member = %#v, error = %v", updated, err)
	}
	if _, err = repository.Members(ctx, sourceTenant.ID, created.ID); !errors.Is(err, tenantdomain.ErrNotFound) {
		t.Fatalf("cross-tenant Members() error = %v", err)
	}

	var audits int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit.operation_logs WHERE request_id IN ($1,$2) AND succeeded`, principal.RequestID, createdPrincipal.RequestID).Scan(&audits); err != nil {
		t.Fatalf("count audits: %v", err)
	}
	if audits != 3 {
		t.Fatalf("audit count = %d, want 3", audits)
	}
}
