package repository

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	webhooks "github.com/appkernia/appkernia/server/internal/modules/webhookadmin/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct{ pool *pgxpool.Pool }

func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

type scanner interface{ Scan(...any) error }

const selectEndpoint = `SELECT id,name,endpoint_url,event_types,max_attempts,timeout_seconds,status,created_at,updated_at FROM sys.webhook_endpoints`

func scanEndpoint(row scanner) (webhooks.Endpoint, error) {
	var v webhooks.Endpoint
	err := row.Scan(&v.ID, &v.Name, &v.EndpointURL, &v.EventTypes, &v.MaxAttempts, &v.TimeoutSeconds, &v.Status, &v.CreatedAt, &v.UpdatedAt)
	if v.EventTypes == nil {
		v.EventTypes = []string{}
	}
	return v, err
}
func scanDelivery(row scanner) (webhooks.Delivery, error) {
	var v webhooks.Delivery
	var payload []byte
	err := row.Scan(&v.ID, &v.EndpointID, &v.EventID, &v.EventType, &payload, &v.Status, &v.AttemptCount, &v.NextAttemptAt, &v.ResponseStatus, &v.ResponseBody, &v.LastError, &v.DeliveredAt, &v.CreatedAt, &v.UpdatedAt)
	if err == nil {
		err = json.Unmarshal(payload, &v.Payload)
	}
	if v.Payload == nil {
		v.Payload = map[string]any{}
	}
	return v, err
}

const selectDelivery = `SELECT d.id,d.endpoint_id,d.event_id,d.event_type,d.payload,d.status,d.attempt_count,d.next_attempt_at,d.response_status,COALESCE(d.response_body,''),COALESCE(d.last_error,''),d.delivered_at,d.created_at,d.updated_at FROM sys.webhook_deliveries d`

func (r *Postgres) List(ctx context.Context, tenant uuid.UUID, f webhooks.Filter) (webhooks.EndpointPage, error) {
	args := []any{tenant}
	where := []string{"tenant_id=$1"}
	if f.Query != "" {
		args = append(args, f.Query)
		where = append(where, fmt.Sprintf("(name ILIKE '%%'||$%d||'%%' OR endpoint_url ILIKE '%%'||$%d||'%%')", len(args), len(args)))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		where = append(where, fmt.Sprintf("status=$%d", len(args)))
	}
	if f.EventType != "" {
		args = append(args, f.EventType)
		where = append(where, fmt.Sprintf("$%d=ANY(event_types)", len(args)))
	}
	cond := strings.Join(where, " AND ")
	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM sys.webhook_endpoints WHERE "+cond, args...).Scan(&total); err != nil {
		return webhooks.EndpointPage{}, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.pool.Query(ctx, selectEndpoint+" WHERE "+cond+fmt.Sprintf(" ORDER BY updated_at DESC,id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return webhooks.EndpointPage{}, err
	}
	defer rows.Close()
	items := []webhooks.Endpoint{}
	for rows.Next() {
		v, e := scanEndpoint(rows)
		if e != nil {
			return webhooks.EndpointPage{}, e
		}
		items = append(items, v)
	}
	return webhooks.EndpointPage{Items: items, Page: f.Page, PageSize: f.PageSize, Total: total}, rows.Err()
}
func (r *Postgres) Create(ctx context.Context, p webhooks.Principal, in webhooks.Input, cipher []byte, version int32) (webhooks.Endpoint, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return webhooks.Endpoint{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO sys.webhook_endpoints(tenant_id,name,endpoint_url,event_types,secret_ciphertext,secret_key_version,max_attempts,timeout_seconds,status,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`, p.TenantID, in.Name, in.EndpointURL, in.EventTypes, cipher, version, in.MaxAttempts, in.TimeoutSeconds, in.Status, p.UserID).Scan(&id)
	if err != nil {
		return webhooks.Endpoint{}, err
	}
	if err = audit(ctx, tx, p, "sys.webhook.create", id, "POST", map[string]any{"name": in.Name, "endpoint_url": in.EndpointURL, "event_types": in.EventTypes, "status": in.Status}); err != nil {
		return webhooks.Endpoint{}, err
	}
	v, err := scanEndpoint(tx.QueryRow(ctx, selectEndpoint+` WHERE tenant_id=$1 AND id=$2`, p.TenantID, id))
	if err != nil {
		return v, err
	}
	return v, tx.Commit(ctx)
}
func (r *Postgres) Update(ctx context.Context, p webhooks.Principal, id uuid.UUID, in webhooks.Input) (webhooks.Endpoint, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return webhooks.Endpoint{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE sys.webhook_endpoints SET name=$1,endpoint_url=$2,event_types=$3,max_attempts=$4,timeout_seconds=$5,status=$6,updated_at=now() WHERE tenant_id=$7 AND id=$8`, in.Name, in.EndpointURL, in.EventTypes, in.MaxAttempts, in.TimeoutSeconds, in.Status, p.TenantID, id)
	if err != nil {
		return webhooks.Endpoint{}, err
	}
	if tag.RowsAffected() != 1 {
		return webhooks.Endpoint{}, webhooks.ErrNotFound
	}
	if err = audit(ctx, tx, p, "sys.webhook.update", id, "PATCH", map[string]any{"endpoint_url": in.EndpointURL, "event_types": in.EventTypes, "status": in.Status}); err != nil {
		return webhooks.Endpoint{}, err
	}
	v, err := scanEndpoint(tx.QueryRow(ctx, selectEndpoint+` WHERE tenant_id=$1 AND id=$2`, p.TenantID, id))
	if err != nil {
		return v, err
	}
	return v, tx.Commit(ctx)
}
func (r *Postgres) GetStored(ctx context.Context, tenant, id uuid.UUID) (webhooks.StoredEndpoint, error) {
	var v webhooks.StoredEndpoint
	err := r.pool.QueryRow(ctx, `SELECT id,name,endpoint_url,event_types,max_attempts,timeout_seconds,status,created_at,updated_at,tenant_id,secret_ciphertext,secret_key_version FROM sys.webhook_endpoints WHERE tenant_id=$1 AND id=$2`, tenant, id).Scan(&v.ID, &v.Name, &v.EndpointURL, &v.EventTypes, &v.MaxAttempts, &v.TimeoutSeconds, &v.Status, &v.CreatedAt, &v.UpdatedAt, &v.TenantID, &v.SecretCiphertext, &v.SecretKeyVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		err = webhooks.ErrNotFound
	}
	return v, err
}
func (r *Postgres) CreateTestDelivery(ctx context.Context, p webhooks.Principal, endpoint uuid.UUID, idem string, eventID uuid.UUID, eventType string, payload map[string]any) (webhooks.Delivery, bool, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return webhooks.Delivery{}, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	raw, err := json.Marshal(payload)
	if err != nil {
		return webhooks.Delivery{}, false, err
	}
	var id uuid.UUID
	created := true
	err = tx.QueryRow(ctx, `INSERT INTO sys.webhook_deliveries(endpoint_id,event_id,event_type,payload,status) SELECT id,$3,$4,$5,'processing' FROM sys.webhook_endpoints WHERE tenant_id=$1 AND id=$2 ON CONFLICT(endpoint_id,event_id) DO NOTHING RETURNING id`, p.TenantID, endpoint, eventID, eventType, raw).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		created = false
		err = tx.QueryRow(ctx, `SELECT d.id FROM sys.webhook_deliveries d JOIN sys.webhook_endpoints e ON e.id=d.endpoint_id WHERE e.tenant_id=$1 AND e.id=$2 AND d.event_id=$3`, p.TenantID, endpoint, eventID).Scan(&id)
	}
	if err != nil {
		return webhooks.Delivery{}, false, err
	}
	if created {
		if err = audit(ctx, tx, p, "sys.webhook.test", endpoint, "POST", map[string]any{"delivery_id": id, "event_id": eventID, "event_type": eventType, "idempotency_key_sha256": fmt.Sprintf("%x", sha256Bytes(idem))}); err != nil {
			return webhooks.Delivery{}, false, err
		}
	}
	v, err := scanDelivery(tx.QueryRow(ctx, selectDelivery+` JOIN sys.webhook_endpoints e ON e.id=d.endpoint_id WHERE e.tenant_id=$1 AND d.id=$2`, p.TenantID, id))
	if err != nil {
		return v, false, err
	}
	return v, created, tx.Commit(ctx)
}
func (r *Postgres) CompleteDelivery(ctx context.Context, p webhooks.Principal, endpoint, delivery uuid.UUID, result webhooks.DeliveryResult, deliverErr error) (webhooks.Delivery, error) {
	status := "succeeded"
	var errorText string
	if deliverErr != nil {
		status = "failed"
		errorText = bounded(deliverErr.Error(), 2000)
	}
	body := bounded(result.Body, 4000)
	var responseStatus any
	if result.StatusCode >= 100 && result.StatusCode <= 599 {
		responseStatus = result.StatusCode
	}
	tag, err := r.pool.Exec(ctx, `UPDATE sys.webhook_deliveries d SET status=$1::varchar,attempt_count=attempt_count+1,response_status=$2,response_body=NULLIF($3,''),last_error=NULLIF($4,''),delivered_at=CASE WHEN $1::varchar='succeeded' THEN now() ELSE NULL END,next_attempt_at=CASE WHEN $1::varchar='failed' THEN now()+interval '1 minute' ELSE NULL END,updated_at=now() FROM sys.webhook_endpoints e WHERE e.id=d.endpoint_id AND e.tenant_id=$5 AND e.id=$6 AND d.id=$7`, status, responseStatus, body, errorText, p.TenantID, endpoint, delivery)
	if err != nil {
		return webhooks.Delivery{}, err
	}
	if tag.RowsAffected() != 1 {
		return webhooks.Delivery{}, webhooks.ErrNotFound
	}
	v, err := scanDelivery(r.pool.QueryRow(ctx, selectDelivery+` JOIN sys.webhook_endpoints e ON e.id=d.endpoint_id WHERE e.tenant_id=$1 AND e.id=$2 AND d.id=$3`, p.TenantID, endpoint, delivery))
	if err != nil {
		return v, err
	}
	if deliverErr != nil {
		return v, nil
	}
	return v, nil
}
func (r *Postgres) Deliveries(ctx context.Context, tenant, endpoint uuid.UUID, page, pageSize int32) (webhooks.DeliveryPage, error) {
	var exists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM sys.webhook_endpoints WHERE tenant_id=$1 AND id=$2)`, tenant, endpoint).Scan(&exists); err != nil {
		return webhooks.DeliveryPage{}, err
	}
	if !exists {
		return webhooks.DeliveryPage{}, webhooks.ErrNotFound
	}
	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM sys.webhook_deliveries WHERE endpoint_id=$1`, endpoint).Scan(&total); err != nil {
		return webhooks.DeliveryPage{}, err
	}
	rows, err := r.pool.Query(ctx, selectDelivery+` WHERE d.endpoint_id=$1 ORDER BY d.created_at DESC,d.id DESC LIMIT $2 OFFSET $3`, endpoint, pageSize, (page-1)*pageSize)
	if err != nil {
		return webhooks.DeliveryPage{}, err
	}
	defer rows.Close()
	items := []webhooks.Delivery{}
	for rows.Next() {
		v, e := scanDelivery(rows)
		if e != nil {
			return webhooks.DeliveryPage{}, e
		}
		items = append(items, v)
	}
	return webhooks.DeliveryPage{Items: items, Page: page, PageSize: pageSize, Total: total}, rows.Err()
}
func bounded(v string, n int) string {
	v = strings.TrimSpace(v)
	if len(v) > n {
		return v[:n]
	}
	return v
}
func sha256Bytes(v string) []byte { sum := sha256.Sum256([]byte(v)); return sum[:] }
func audit(ctx context.Context, tx pgx.Tx, p webhooks.Principal, action string, id uuid.UUID, method string, after any) error {
	raw, err := json.Marshal(after)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit.operation_logs(tenant_id,user_id,session_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,http_method,request_path,response_status,user_agent,after_data,succeeded) VALUES($1,$2,NULLIF($3,'00000000-0000-0000-0000-000000000000'::uuid),$4,'sys',$5,$5,'webhook_endpoint',$6,$7,$8,200,NULLIF($9,''),$10,true)`, p.TenantID, p.UserID, p.SessionID, p.RequestID, action, id.String(), method, "/admin-api/v1/webhooks/"+id.String(), strings.TrimSpace(p.UserAgent), raw)
	return err
}
