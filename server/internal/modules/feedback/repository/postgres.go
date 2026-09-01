package repository

import (
	"bytes"
	"context"
	"errors"
	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	f "github.com/appkernia/appkernia/server/internal/modules/feedback/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Postgres struct {
	pool *pgxpool.Pool
	q    db.DBTX
}

func NewPostgres(p *pgxpool.Pool) *Postgres { return &Postgres{p, p} }
func (r *Postgres) Transact(ctx context.Context, fn func(f.Repository) error) error {
	tx, e := r.pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if e = fn(&Postgres{r.pool, tx}); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func result(e error) error {
	if errors.Is(e, pgx.ErrNoRows) {
		return f.ErrNotFound
	}
	return e
}
func (r *Postgres) CheckScope(ctx context.Context, p f.Scope) error {
	var ok bool
	e := r.q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.applications a WHERE a.id=$1 AND a.tenant_id=$2 AND a.deleted_at IS NULL AND a.status='active' AND ($4 OR EXISTS(SELECT 1 FROM app.user_memberships m WHERE m.app_id=a.id AND m.tenant_id=a.tenant_id AND m.user_id=$3 AND m.status='active')))`, p.AppID, p.TenantID, p.UserID, p.Admin).Scan(&ok)
	if e == nil && !ok {
		return f.ErrNotFound
	}
	return e
}
func timestamp(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
func (r *Postgres) List(ctx context.Context, p f.Scope, q f.Filter) (f.Page, error) {
	params := db.ListFeedbacksParams{TenantID: p.TenantID, AppID: p.AppID, UserID: p.UserID, Admin: p.Admin, Query: q.Query, Status: q.Status, CreatedFrom: timestamp(q.From), CreatedTo: timestamp(q.To), PageSize: q.PageSize, PageOffset: int64(q.Page-1) * int64(q.PageSize)}
	rows, e := db.New(r.q).ListFeedbacks(ctx, params)
	if e != nil {
		return f.Page{}, e
	}
	count, e := db.New(r.q).CountFeedbacks(ctx, db.CountFeedbacksParams{TenantID: p.TenantID, AppID: p.AppID, UserID: p.UserID, Admin: p.Admin, Query: q.Query, Status: q.Status, CreatedFrom: params.CreatedFrom, CreatedTo: params.CreatedTo})
	if e != nil {
		return f.Page{}, e
	}
	out := f.Page{Items: []f.Feedback{}, Total: count, Page: q.Page, PageSize: q.PageSize}
	for _, x := range rows {
		out.Items = append(out.Items, f.Feedback{ID: x.ID, UserID: x.UserID, Description: x.Description, Platform: x.Platform, AppVersion: x.AppVersion, Status: x.Status, LockVersion: x.LockVersion, CreatedAt: x.CreatedAt.Time, UpdatedAt: x.UpdatedAt.Time, Attachments: []f.Attachment{}, Replies: []f.Reply{}, Events: []f.Event{}})
	}
	return out, nil
}
func (r *Postgres) Get(ctx context.Context, p f.Scope, id uuid.UUID, lock bool) (f.Feedback, error) {
	x := f.Feedback{Attachments: []f.Attachment{}, Replies: []f.Reply{}, Events: []f.Event{}}
	query := `SELECT id,user_id,description,contact,platform,app_version,status,lock_version,created_at,updated_at FROM app.feedbacks WHERE tenant_id=$1 AND app_id=$2 AND id=$3 AND ($4 OR user_id=$5)`
	if lock {
		query += " FOR UPDATE"
	}
	e := r.q.QueryRow(ctx, query, p.TenantID, p.AppID, id, p.Admin, p.UserID).Scan(&x.ID, &x.UserID, &x.Description, &x.Contact, &x.Platform, &x.AppVersion, &x.Status, &x.LockVersion, &x.CreatedAt, &x.UpdatedAt)
	if e != nil {
		return x, result(e)
	}
	rows, e := r.q.Query(ctx, `SELECT a.file_id,COALESCE(s.media_type,''),s.size_bytes FROM app.feedback_attachments a JOIN storage.files s ON s.id=a.file_id AND s.tenant_id=a.tenant_id WHERE a.tenant_id=$1 AND a.app_id=$2 AND a.feedback_id=$3 ORDER BY a.position`, p.TenantID, p.AppID, id)
	if e != nil {
		return x, e
	}
	for rows.Next() {
		var a f.Attachment
		if e = rows.Scan(&a.FileID, &a.MediaType, &a.SizeBytes); e != nil {
			rows.Close()
			return x, e
		}
		x.Attachments = append(x.Attachments, a)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return x, e
	}
	rows, e = r.q.Query(ctx, `SELECT id,body,created_at FROM app.feedback_replies WHERE tenant_id=$1 AND app_id=$2 AND feedback_id=$3 ORDER BY created_at,id`, p.TenantID, p.AppID, id)
	if e != nil {
		return x, e
	}
	for rows.Next() {
		var a f.Reply
		if e = rows.Scan(&a.ID, &a.Body, &a.CreatedAt); e != nil {
			rows.Close()
			return x, e
		}
		x.Replies = append(x.Replies, a)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return x, e
	}
	rows, e = r.q.Query(ctx, `SELECT id,from_status,to_status,created_at FROM app.feedback_events WHERE tenant_id=$1 AND app_id=$2 AND feedback_id=$3 ORDER BY created_at,id`, p.TenantID, p.AppID, id)
	if e != nil {
		return x, e
	}
	defer rows.Close()
	for rows.Next() {
		var a f.Event
		if e = rows.Scan(&a.ID, &a.From, &a.To, &a.CreatedAt); e != nil {
			return x, e
		}
		x.Events = append(x.Events, a)
	}
	return x, rows.Err()
}
func (r *Postgres) FindRequest(ctx context.Context, p f.Scope, key uuid.UUID, hash []byte) (uuid.UUID, error) {
	// Serializes retries without conflating request IDs with idempotency identities.
	if _, e := r.q.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, p.AppID.String()+p.UserID.String()+key.String()); e != nil {
		return uuid.Nil, e
	}
	var id uuid.UUID
	var existing []byte
	e := r.q.QueryRow(ctx, `SELECT id,request_hash FROM app.feedbacks WHERE tenant_id=$1 AND app_id=$2 AND user_id=$3 AND idempotency_key=$4`, p.TenantID, p.AppID, p.UserID, key).Scan(&id, &existing)
	if errors.Is(e, pgx.ErrNoRows) {
		return uuid.Nil, nil
	}
	if e == nil && !bytes.Equal(existing, hash) {
		return uuid.Nil, f.ErrConflict
	}
	return id, e
}
func (r *Postgres) Create(ctx context.Context, p f.Scope, x f.Input, key uuid.UUID, hash []byte) (uuid.UUID, error) {
	var member uuid.UUID
	if e := r.q.QueryRow(ctx, `SELECT user_id FROM app.user_memberships WHERE tenant_id=$1 AND app_id=$2 AND user_id=$3 AND status='active' FOR SHARE`, p.TenantID, p.AppID, p.UserID).Scan(&member); e != nil {
		return uuid.Nil, result(e)
	}
	var id uuid.UUID
	e := r.q.QueryRow(ctx, `INSERT INTO app.feedbacks(tenant_id,app_id,user_id,description,contact,platform,app_version,idempotency_key,request_hash) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`, p.TenantID, p.AppID, p.UserID, x.Description, x.Contact, x.Platform, x.AppVersion, key, hash).Scan(&id)
	return id, e
}
func (r *Postgres) Attach(ctx context.Context, p f.Scope, id uuid.UUID, ids []uuid.UUID) error {
	for i, file := range ids {
		var valid uuid.UUID
		e := r.q.QueryRow(ctx, `SELECT s.id FROM storage.files s JOIN storage.upload_sessions u ON u.file_id=s.id AND u.tenant_id=s.tenant_id AND u.app_id=s.app_id WHERE s.id=$1 AND s.tenant_id=$2 AND s.app_id=$3 AND s.owner_user_id=$4 AND s.metadata->>'purpose'='feedback' AND s.status='ready' AND s.scan_status='clean' AND s.deleted_at IS NULL AND u.purpose='feedback' AND u.status='completed' AND u.expires_at>now() AND NOT EXISTS(SELECT 1 FROM app.feedback_attachments a WHERE a.file_id=s.id) FOR UPDATE OF s,u`, file, p.TenantID, p.AppID, p.UserID).Scan(&valid)
		if e != nil {
			return result(e)
		}
		if _, e = r.q.Exec(ctx, `INSERT INTO app.feedback_attachments(tenant_id,app_id,feedback_id,file_id,position) VALUES($1,$2,$3,$4,$5)`, p.TenantID, p.AppID, id, file, i); e != nil {
			return e
		}
		if _, e = r.q.Exec(ctx, `INSERT INTO storage.file_usages(tenant_id,file_id,module_code,entity_type,entity_id,field_name) VALUES($1,$2,'app','app.feedback',$3,'screenshot')`, p.TenantID, file, id); e != nil {
			return e
		}
	}
	return nil
}
func (r *Postgres) Change(ctx context.Context, p f.Scope, id uuid.UUID, status string, version int32) error {
	current, e := r.Get(ctx, p, id, true)
	if e != nil {
		return e
	}
	if current.LockVersion != version {
		return f.ErrConflict
	}
	if _, e = r.q.Exec(ctx, `UPDATE app.feedbacks SET status=$4,lock_version=lock_version+1,updated_at=now() WHERE tenant_id=$1 AND app_id=$2 AND id=$3`, p.TenantID, p.AppID, id, status); e != nil {
		return e
	}
	if current.Status != status {
		_, e = r.q.Exec(ctx, `INSERT INTO app.feedback_events(tenant_id,app_id,feedback_id,actor_id,from_status,to_status) VALUES($1,$2,$3,$4,$5,$6)`, p.TenantID, p.AppID, id, p.UserID, current.Status, status)
	}
	return e
}
func (r *Postgres) FindReply(ctx context.Context, p f.Scope, id, key uuid.UUID, hash []byte) (bool, error) {
	var old []byte
	var author *uuid.UUID
	e := r.q.QueryRow(ctx, `SELECT request_hash,author_id FROM app.feedback_replies WHERE tenant_id=$1 AND app_id=$2 AND feedback_id=$3 AND idempotency_key=$4`, p.TenantID, p.AppID, id, key).Scan(&old, &author)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, nil
	}
	if e == nil && (!bytes.Equal(old, hash) || author == nil || *author != p.UserID) {
		return false, f.ErrConflict
	}
	return e == nil, e
}
func (r *Postgres) Reply(ctx context.Context, p f.Scope, id uuid.UUID, x f.ReplyInput, key uuid.UUID, hash []byte) error {
	_, e := r.q.Exec(ctx, `INSERT INTO app.feedback_replies(tenant_id,app_id,feedback_id,author_id,body,idempotency_key,request_hash) VALUES($1,$2,$3,$4,$5,$6,$7)`, p.TenantID, p.AppID, id, p.UserID, x.Body, key, hash)
	return e
}
func (r *Postgres) Audit(ctx context.Context, p f.Scope, id uuid.UUID, action string) error {
	_, e := r.q.Exec(ctx, `INSERT INTO audit.operation_logs(tenant_id,app_id,user_id,session_id,request_id,module_code,action_name,resource_type,resource_id,response_status,succeeded) VALUES($1,$2,$3,NULLIF($4::uuid,'00000000-0000-0000-0000-000000000000'::uuid),$5,'app',$6,'app.feedback',$7,200,true)`, p.TenantID, p.AppID, p.UserID, p.SessionID, p.RequestID, "app.feedback."+action, id.String())
	return e
}
func (r *Postgres) CreateUpload(ctx context.Context, p f.Scope, u f.Upload) (f.Upload, error) {
	_, e := r.q.Exec(ctx, `INSERT INTO storage.upload_sessions(id,tenant_id,app_id,user_id,provider,bucket_name,object_key,original_name,media_type,expected_size,expires_at,purpose) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'feedback')`, u.ID, p.TenantID, p.AppID, p.UserID, u.Object.Provider, u.Object.Bucket, u.Object.Key, u.Name, u.MediaType, u.Size, u.ExpiresAt)
	return u, e
}
func (r *Postgres) GetUpload(ctx context.Context, p f.Scope, id uuid.UUID, lock bool) (f.Upload, error) {
	var x f.Upload
	query := `SELECT id,file_id,status,provider,bucket_name,object_key,original_name,COALESCE(media_type,''),expected_size,expires_at FROM storage.upload_sessions WHERE id=$1 AND tenant_id=$2 AND app_id=$3 AND user_id=$4 AND purpose='feedback' AND status IN ('initiated','completed') AND expires_at>now()`
	if lock {
		query += " FOR UPDATE"
	}
	e := r.q.QueryRow(ctx, query, id, p.TenantID, p.AppID, p.UserID).Scan(&x.ID, &x.FileID, &x.Status, &x.Object.Provider, &x.Object.Bucket, &x.Object.Key, &x.Name, &x.MediaType, &x.Size, &x.ExpiresAt)
	x.Object.TenantID = p.TenantID
	return x, result(e)
}
func (r *Postgres) CompleteUpload(ctx context.Context, p f.Scope, u f.Upload, hash []byte) (uuid.UUID, error) {
	var id uuid.UUID
	e := r.q.QueryRow(ctx, `INSERT INTO storage.files(tenant_id,app_id,owner_user_id,provider,bucket_name,object_key,original_name,media_type,size_bytes,sha256,status,scan_status,metadata) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'ready','clean','{"purpose":"feedback"}'::jsonb) RETURNING id`, p.TenantID, p.AppID, p.UserID, u.Object.Provider, u.Object.Bucket, u.Object.Key, u.Name, u.MediaType, u.Size, hash).Scan(&id)
	if e != nil {
		return id, e
	}
	_, e = r.q.Exec(ctx, `UPDATE storage.upload_sessions SET file_id=$5,status='completed',completed_at=now() WHERE id=$1 AND tenant_id=$2 AND app_id=$3 AND user_id=$4 AND purpose='feedback'`, u.ID, p.TenantID, p.AppID, p.UserID, id)
	return id, e
}
func (r *Postgres) File(ctx context.Context, p f.Scope, feedbackID, fileID uuid.UUID) (f.File, error) {
	var x f.File
	e := r.q.QueryRow(ctx, `SELECT s.id,COALESCE(s.media_type,''),s.size_bytes,s.provider,s.bucket_name,s.object_key,s.sha256 FROM storage.files s WHERE s.tenant_id=$1 AND s.app_id=$2 AND s.id=$3 AND s.metadata->>'purpose'='feedback' AND s.status='ready' AND s.scan_status='clean' AND s.deleted_at IS NULL AND (($4::uuid='00000000-0000-0000-0000-000000000000' AND NOT $5 AND s.owner_user_id=$6 AND EXISTS(SELECT 1 FROM storage.upload_sessions u WHERE u.file_id=s.id AND u.tenant_id=$1 AND u.app_id=$2 AND u.user_id=$6 AND u.purpose='feedback' AND u.status='completed' AND u.expires_at>now())) OR EXISTS(SELECT 1 FROM app.feedback_attachments a JOIN app.feedbacks b ON b.id=a.feedback_id AND b.tenant_id=a.tenant_id AND b.app_id=a.app_id WHERE a.file_id=s.id AND a.tenant_id=$1 AND a.app_id=$2 AND b.id=$4 AND ($5 OR b.user_id=$6)))`, p.TenantID, p.AppID, fileID, feedbackID, p.Admin, p.UserID).Scan(&x.FileID, &x.MediaType, &x.SizeBytes, &x.Object.Provider, &x.Object.Bucket, &x.Object.Key, &x.SHA256)
	x.Object.TenantID = p.TenantID
	return x, result(e)
}
func (r *Postgres) CancelUpload(ctx context.Context, p f.Scope, id uuid.UUID) error {
	tag, e := r.q.Exec(ctx, `UPDATE storage.upload_sessions u SET expires_at=GREATEST(u.created_at+interval '1 microsecond',now()) WHERE u.id=$1 AND u.tenant_id=$2 AND u.app_id=$3 AND u.user_id=$4 AND u.purpose='feedback' AND NOT EXISTS(SELECT 1 FROM app.feedback_attachments a WHERE a.file_id=u.file_id)`, id, p.TenantID, p.AppID, p.UserID)
	if e == nil && tag.RowsAffected() == 0 {
		return f.ErrNotFound
	}
	return e
}
