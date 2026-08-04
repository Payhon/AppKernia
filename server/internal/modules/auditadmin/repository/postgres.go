package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	auditdomain "github.com/appkernia/appkernia/server/internal/modules/auditadmin/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const redactedValue = "[REDACTED]"

type Postgres struct{ pool *pgxpool.Pool }

func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

func (r *Postgres) ListOperations(ctx context.Context, tenantID uuid.UUID, f auditdomain.OperationFilter) (auditdomain.Page[auditdomain.Operation], error) {
	q := db.New(r.pool)
	params := db.AuditCountOperationsParams{TenantID: &tenantID, FromAt: timeValue(f.FromAt), ToAt: timeValue(f.ToAt), Query: f.Query, ModuleCode: f.ModuleCode, Result: f.Result}
	total, err := q.AuditCountOperations(ctx, params)
	if err != nil {
		return auditdomain.Page[auditdomain.Operation]{}, fmt.Errorf("count operation audits: %w", err)
	}
	rows, err := q.AuditListOperations(ctx, db.AuditListOperationsParams{TenantID: params.TenantID, FromAt: params.FromAt, ToAt: params.ToAt, Query: params.Query, ModuleCode: params.ModuleCode, Result: params.Result, PageOffset: (f.Page - 1) * f.PageSize, PageSize: f.PageSize})
	if err != nil {
		return auditdomain.Page[auditdomain.Operation]{}, fmt.Errorf("list operation audits: %w", err)
	}
	items := make([]auditdomain.Operation, 0, len(rows))
	for _, row := range rows {
		items = append(items, auditdomain.Operation{
			ID: row.ID, UserID: row.UserID, SessionID: row.SessionID, RequestID: row.RequestID, TraceID: row.TraceID,
			ModuleCode: row.ModuleCode, ActionName: row.ActionName, PermissionCode: row.PermissionCode,
			ResourceType: row.ResourceType, ResourceID: row.ResourceID, HTTPMethod: row.HttpMethod,
			RequestPath: row.RequestPath, ResponseStatus: row.ResponseStatus, ClientIP: interfaceString(row.ClientIp),
			RequestSummary: decodeRedacted(row.RequestSummary), BeforeData: decodeRedacted(row.BeforeData), AfterData: decodeRedacted(row.AfterData),
			DurationMS: row.DurationMs, Succeeded: row.Succeeded, ErrorCode: row.ErrorCode, OccurredAt: row.OccurredAt.Time.UTC(),
		})
	}
	return auditdomain.Page[auditdomain.Operation]{Items: items, Total: total, Page: f.Page, PageSize: f.PageSize}, nil
}

func (r *Postgres) ListLogins(ctx context.Context, tenantID uuid.UUID, f auditdomain.LoginFilter) (auditdomain.Page[auditdomain.Login], error) {
	q := db.New(r.pool)
	params := db.AuditCountLoginsParams{TenantID: &tenantID, FromAt: timeValue(f.FromAt), ToAt: timeValue(f.ToAt), Result: f.Result, Audience: f.Audience, AuthMethod: f.AuthMethod, Query: f.Query}
	total, err := q.AuditCountLogins(ctx, params)
	if err != nil {
		return auditdomain.Page[auditdomain.Login]{}, fmt.Errorf("count login audits: %w", err)
	}
	rows, err := q.AuditListLogins(ctx, db.AuditListLoginsParams{TenantID: params.TenantID, FromAt: params.FromAt, ToAt: params.ToAt, Result: params.Result, Audience: params.Audience, AuthMethod: params.AuthMethod, Query: params.Query, PageOffset: (f.Page - 1) * f.PageSize, PageSize: f.PageSize})
	if err != nil {
		return auditdomain.Page[auditdomain.Login]{}, fmt.Errorf("list login audits: %w", err)
	}
	items := make([]auditdomain.Login, 0, len(rows))
	for _, row := range rows {
		items = append(items, auditdomain.Login{ID: row.ID, UserID: row.UserID, SessionID: row.SessionID, RequestID: row.RequestID, LoginIdentifierHint: row.LoginIdentifierHint, AuthMethod: row.AuthMethod, Audience: row.Audience, Result: row.Result, FailureReason: row.FailureReason, ClientIP: interfaceString(row.ClientIp), OccurredAt: row.OccurredAt.Time.UTC()})
	}
	return auditdomain.Page[auditdomain.Login]{Items: items, Total: total, Page: f.Page, PageSize: f.PageSize}, nil
}

func (r *Postgres) ListSecurityEvents(ctx context.Context, tenantID uuid.UUID, f auditdomain.SecurityFilter) (auditdomain.Page[auditdomain.SecurityEvent], error) {
	q := db.New(r.pool)
	params := db.AuditCountSecurityEventsParams{TenantID: &tenantID, FromAt: timeValue(f.FromAt), ToAt: timeValue(f.ToAt), Query: f.Query, Severity: f.Severity, Source: f.Source, Status: f.Status}
	total, err := q.AuditCountSecurityEvents(ctx, params)
	if err != nil {
		return auditdomain.Page[auditdomain.SecurityEvent]{}, fmt.Errorf("count security events: %w", err)
	}
	rows, err := q.AuditListSecurityEvents(ctx, db.AuditListSecurityEventsParams{TenantID: params.TenantID, FromAt: params.FromAt, ToAt: params.ToAt, Query: params.Query, Severity: params.Severity, Source: params.Source, Status: params.Status, PageOffset: (f.Page - 1) * f.PageSize, PageSize: f.PageSize})
	if err != nil {
		return auditdomain.Page[auditdomain.SecurityEvent]{}, fmt.Errorf("list security events: %w", err)
	}
	items := make([]auditdomain.SecurityEvent, 0, len(rows))
	for _, row := range rows {
		items = append(items, auditdomain.SecurityEvent{ID: row.ID, UserID: row.UserID, SessionID: row.SessionID, EventType: row.EventType, Severity: row.Severity, Source: row.Source, ClientIP: interfaceString(row.ClientIp), ResolvedAt: timePointer(row.ResolvedAt), ResolvedBy: row.ResolvedBy, OccurredAt: row.OccurredAt.Time.UTC(), Details: map[string]any{}})
	}
	return auditdomain.Page[auditdomain.SecurityEvent]{Items: items, Total: total, Page: f.Page, PageSize: f.PageSize}, nil
}

func (r *Postgres) GetSecurityEvent(ctx context.Context, tenantID, id uuid.UUID) (auditdomain.SecurityEvent, error) {
	row, err := db.New(r.pool).AuditGetSecurityEvent(ctx, db.AuditGetSecurityEventParams{TenantID: &tenantID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return auditdomain.SecurityEvent{}, auditdomain.ErrNotFound
	}
	if err != nil {
		return auditdomain.SecurityEvent{}, fmt.Errorf("get security event: %w", err)
	}
	return securityEvent(row), nil
}

func (r *Postgres) ResolveSecurityEvent(ctx context.Context, p auditdomain.Principal, id uuid.UUID) (auditdomain.SecurityEvent, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return auditdomain.SecurityEvent{}, fmt.Errorf("begin resolve security event: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var eventType, severity, source string
	var resolvedAt pgtype.Timestamptz
	err = tx.QueryRow(ctx, `SELECT event_type,severity,source,resolved_at FROM audit.security_events WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, p.TenantID, id).Scan(&eventType, &severity, &source, &resolvedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return auditdomain.SecurityEvent{}, auditdomain.ErrNotFound
	}
	if err != nil {
		return auditdomain.SecurityEvent{}, fmt.Errorf("lock security event: %w", err)
	}
	if resolvedAt.Valid {
		return auditdomain.SecurityEvent{}, auditdomain.ErrAlreadyResolved
	}
	if _, err = tx.Exec(ctx, `UPDATE audit.security_events SET resolved_at=now(), resolved_by=$3 WHERE tenant_id=$1 AND id=$2`, p.TenantID, id, p.UserID); err != nil {
		return auditdomain.SecurityEvent{}, fmt.Errorf("resolve security event: %w", err)
	}
	before, _ := json.Marshal(map[string]any{"event_type": eventType, "severity": severity, "source": source, "resolved": false})
	after, _ := json.Marshal(map[string]any{"event_type": eventType, "severity": severity, "source": source, "resolved": true, "resolved_by": p.UserID})
	if _, err = tx.Exec(ctx, `INSERT INTO audit.operation_logs(tenant_id,user_id,session_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,http_method,request_path,response_status,client_ip,user_agent,before_data,after_data,succeeded) VALUES($1,$2,nullif($3,'00000000-0000-0000-0000-000000000000'::uuid),nullif($4,''),'audit','audit.security.resolve','audit.security.resolve','audit.security_event',$5,'POST',$6,200,$7,nullif($8,''),$9,$10,true)`, p.TenantID, p.UserID, p.SessionID, p.RequestID, id.String(), "/admin-api/v1/audit/security-events/"+id.String()+"/resolve", p.IPAddress, p.UserAgent, before, after); err != nil {
		return auditdomain.SecurityEvent{}, fmt.Errorf("audit security resolution: %w", err)
	}
	row, err := db.New(tx).AuditGetSecurityEvent(ctx, db.AuditGetSecurityEventParams{TenantID: &p.TenantID, ID: id})
	if err != nil {
		return auditdomain.SecurityEvent{}, fmt.Errorf("read resolved security event: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return auditdomain.SecurityEvent{}, fmt.Errorf("commit resolved security event: %w", err)
	}
	return securityEvent(row), nil
}

func securityEvent(row db.AuditGetSecurityEventRow) auditdomain.SecurityEvent {
	return auditdomain.SecurityEvent{ID: row.ID, UserID: row.UserID, SessionID: row.SessionID, EventType: row.EventType, Severity: row.Severity, Source: row.Source, ClientIP: interfaceString(row.ClientIp), Details: decodeRedacted(row.Details), ResolvedAt: timePointer(row.ResolvedAt), ResolvedBy: row.ResolvedBy, OccurredAt: row.OccurredAt.Time.UTC()}
}

func timeValue(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func timePointer(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time.UTC()
	return &result
}

func interfaceString(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func decodeRedacted(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return map[string]any{}
	}
	redacted := redactValue(value)
	if object, ok := redacted.(map[string]any); ok {
		return object
	}
	return map[string]any{"value": redacted}
}

func redactValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, child := range typed {
			if sensitiveKey(key) {
				result[key] = redactedValue
			} else {
				result[key] = redactValue(child)
			}
		}
		return result
	case []any:
		result := make([]any, len(typed))
		for index, child := range typed {
			result[index] = redactValue(child)
		}
		return result
	case string:
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(typed)), "bearer ") {
			return redactedValue
		}
	}
	return value
}

func sensitiveKey(key string) bool {
	normalized := strings.NewReplacer("_", "", "-", "", ".", "", " ", "").Replace(strings.ToLower(key))
	for _, marker := range []string{"password", "passwd", "token", "secret", "authorization", "cookie", "otp", "mfa", "apikey", "privatekey", "credential"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}
