//go:build integration

package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	auditdomain "github.com/appkernia/appkernia/server/internal/modules/auditadmin/domain"
	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	iamrepo "github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresAuditTenantIsolationRedactionAndResolution(t *testing.T) {
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
	hash, err := iamapp.HashPassword("audit integration password 2026!")
	if err != nil {
		t.Fatal(err)
	}
	identities := iamrepo.NewPostgres(pool)
	owner, tenant, err := identities.CreateIdentity(ctx, iamdomain.CreateIdentity{TenantCode: "audit-source-" + suffix, TenantName: "Audit Source", Email: "audit-owner-" + suffix + "@example.test", DisplayName: "Audit Owner", Locale: "zh-CN", PasswordHash: hash})
	if err != nil {
		t.Fatalf("create identity: %v", err)
	}
	_, otherTenant, err := identities.CreateIdentity(ctx, iamdomain.CreateIdentity{TenantCode: "audit-other-" + suffix, TenantName: "Audit Other", Email: "audit-other-" + suffix + "@example.test", DisplayName: "Other", Locale: "en-US", PasswordHash: hash})
	if err != nil {
		t.Fatalf("create other identity: %v", err)
	}
	now := time.Now().UTC()
	requestID := "audit-integration-" + suffix
	if _, err = pool.Exec(ctx, `INSERT INTO audit.operation_logs(tenant_id,user_id,request_id,module_code,action_name,request_summary,before_data,after_data,succeeded) VALUES($1,$2,$3,'audit','audit.fixture',$4,$5,$6,true)`, tenant.ID, owner.ID, requestID, `{"Authorization":"Bearer raw","safe":"visible"}`, `{"password":"plain","state":"before"}`, `{"nested":{"access_token":"raw"},"state":"after"}`); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO audit.login_events(tenant_id,user_id,request_id,login_identifier_hint,auth_method,audience,result,failure_reason) VALUES($1,$2,$3,'a***@example.test','password','ak-admin','failure','invalid_credentials')`, tenant.ID, owner.ID, requestID); err != nil {
		t.Fatal(err)
	}
	var securityID, otherSecurityID uuid.UUID
	if err = pool.QueryRow(ctx, `INSERT INTO audit.security_events(tenant_id,user_id,event_type,severity,source,details) VALUES($1,$2,'audit.fixture','critical','integration',$3) RETURNING id`, tenant.ID, owner.ID, `{"credential":"plain","safe":"visible","nested":{"api_key":"raw"}}`).Scan(&securityID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO audit.security_events(tenant_id,event_type,severity,source) VALUES($1,'audit.other','low','integration') RETURNING id`, otherTenant.ID).Scan(&otherSecurityID); err != nil {
		t.Fatal(err)
	}
	repository := NewPostgres(pool)
	base := auditdomain.PageFilter{FromAt: now.Add(-time.Hour), ToAt: now.Add(time.Hour), Page: 1, PageSize: 20}
	operations, err := repository.ListOperations(ctx, tenant.ID, auditdomain.OperationFilter{PageFilter: base, ModuleCode: "audit"})
	if err != nil || operations.Total < 1 {
		t.Fatalf("operations=%#v error=%v", operations, err)
	}
	var fixture *auditdomain.Operation
	for index := range operations.Items {
		if operations.Items[index].RequestID == requestID {
			fixture = &operations.Items[index]
		}
	}
	if fixture == nil || fixture.RequestSummary["Authorization"] != redactedValue || fixture.BeforeData["password"] != redactedValue || fixture.AfterData["nested"].(map[string]any)["access_token"] != redactedValue {
		t.Fatalf("operation redaction=%#v", fixture)
	}
	logins, err := repository.ListLogins(ctx, tenant.ID, auditdomain.LoginFilter{PageFilter: base, Result: "failure"})
	if err != nil || len(logins.Items) == 0 || logins.Items[0].LoginIdentifierHint == "" {
		t.Fatalf("logins=%#v error=%v", logins, err)
	}
	if _, err = repository.GetSecurityEvent(ctx, tenant.ID, otherSecurityID); !errors.Is(err, auditdomain.ErrNotFound) {
		t.Fatalf("cross tenant detail error=%v", err)
	}
	event, err := repository.GetSecurityEvent(ctx, tenant.ID, securityID)
	if err != nil || event.Details["credential"] != redactedValue || event.Details["nested"].(map[string]any)["api_key"] != redactedValue {
		t.Fatalf("security event=%#v error=%v", event, err)
	}
	principal := auditdomain.Principal{TenantID: tenant.ID, UserID: owner.ID, RequestID: requestID + "-resolve"}
	event, err = repository.ResolveSecurityEvent(ctx, principal, securityID)
	if err != nil || event.ResolvedAt == nil || event.ResolvedBy == nil || *event.ResolvedBy != owner.ID {
		t.Fatalf("resolved event=%#v error=%v", event, err)
	}
	if _, err = repository.ResolveSecurityEvent(ctx, principal, securityID); !errors.Is(err, auditdomain.ErrAlreadyResolved) {
		t.Fatalf("second resolve error=%v", err)
	}
	var auditCount int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit.operation_logs WHERE tenant_id=$1 AND request_id=$2 AND action_name='audit.security.resolve' AND succeeded`, tenant.ID, principal.RequestID).Scan(&auditCount); err != nil || auditCount != 1 {
		t.Fatalf("resolve audit count=%d error=%v", auditCount, err)
	}
}
