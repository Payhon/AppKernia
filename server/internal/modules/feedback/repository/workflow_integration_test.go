//go:build integration

package repository_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	cms "github.com/appkernia/appkernia/server/internal/modules/appmanagement/application"
	app "github.com/appkernia/appkernia/server/internal/modules/feedback/application"
	f "github.com/appkernia/appkernia/server/internal/modules/feedback/domain"
	repo "github.com/appkernia/appkernia/server/internal/modules/feedback/repository"
	cleanup "github.com/appkernia/appkernia/server/internal/modules/feedback/worker"
	iam "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	store "github.com/appkernia/appkernia/server/internal/modules/storage/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"image"
	"image/png"
	"io"
	"os"
	"sync"
	"testing"
	"time"
)

type cmsAuth struct{ principal iam.AuthenticatedContext }

func (a cmsAuth) Authenticate(context.Context, string, string) (iam.AuthenticatedContext, error) {
	return a.principal, nil
}

type testScanner struct{ err error }

func (s testScanner) Scan(context.Context, []byte) error { return s.err }

func TestFeedbackWorkflow(t *testing.T) {
	dsn := os.Getenv("AK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, e := pgxpool.New(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(pool.Close)
	p := f.Scope{TenantID: uuid.New(), AppID: uuid.New(), UserID: uuid.New(), RequestID: uuid.NewString()}
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, e := pool.Exec(ctx, sql, args...); e != nil {
			t.Fatal(e)
		}
	}
	exec(`INSERT INTO iam.tenants(id,code,name) VALUES($1,$2,'Feedback Test')`, p.TenantID, "fb-"+uuid.NewString())
	exec(`INSERT INTO iam.users(id,email,display_name,status) VALUES($1,$2,'Feedback Test','active')`, p.UserID, uuid.NewString()+"@example.test")
	exec(`INSERT INTO iam.tenant_members(tenant_id,user_id,status) VALUES($1,$2,'active')`, p.TenantID, p.UserID)
	exec(`INSERT INTO app.applications(id,tenant_id,appid,code,name,app_type,owner_type,owner_user_id) VALUES($1,$2,$3,$4,'Feedback Test','uni_app_x','user',$5)`, p.AppID, p.TenantID, "__UNI__"+uuid.NewString()[:8], "fb-"+uuid.NewString(), p.UserID)
	exec(`INSERT INTO app.user_memberships(app_id,tenant_id,user_id,status,source) VALUES($1,$2,$3,'active','admin_created')`, p.AppID, p.TenantID, p.UserID)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM storage.file_usages WHERE tenant_id=$1`, p.TenantID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.feedbacks WHERE tenant_id=$1`, p.TenantID)
		_, _ = pool.Exec(ctx, `DELETE FROM storage.upload_sessions WHERE tenant_id=$1`, p.TenantID)
		_, _ = pool.Exec(ctx, `DELETE FROM storage.files WHERE tenant_id=$1`, p.TenantID)
		_, _ = pool.Exec(ctx, `DELETE FROM audit.operation_logs WHERE tenant_id=$1`, p.TenantID)
		_, _ = pool.Exec(ctx, `UPDATE content.pages SET page_type='custom' WHERE tenant_id=$1`, p.TenantID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.applications WHERE tenant_id=$1`, p.TenantID)
		_, _ = pool.Exec(ctx, `DELETE FROM iam.tenant_members WHERE tenant_id=$1`, p.TenantID)
		_, _ = pool.Exec(ctx, `DELETE FROM iam.users WHERE id=$1`, p.UserID)
		_, _ = pool.Exec(ctx, `DELETE FROM iam.tenants WHERE id=$1`, p.TenantID)
	})
	objects, e := store.NewLocalObjectStore(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	r := repo.NewPostgres(pool)
	s := app.NewService(nil, r, objects, testScanner{})
	if e = r.CheckScope(ctx, p); e != nil {
		t.Fatal(e)
	}
	var data bytes.Buffer
	if e = png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 4, 4))); e != nil {
		t.Fatal(e)
	}
	upload := func() (f.Upload, uuid.UUID) {
		t.Helper()
		u, e := s.CreateUpload(ctx, p, f.UploadInput{OriginalName: "screenshot.png", MediaType: "image/png", SizeBytes: int64(data.Len())})
		if e != nil {
			t.Fatal(e)
		}
		id, e := s.Upload(ctx, p, u.ID, data.Bytes())
		if e != nil {
			t.Fatal(e)
		}
		return u, id
	}
	t.Run("scanner is required and rejected bytes never become a file", func(t *testing.T) {
		disabled := app.NewService(nil, r, objects)
		input := f.UploadInput{OriginalName: "screenshot.png", MediaType: "image/png", SizeBytes: int64(data.Len())}
		if _, err := disabled.CreateUpload(ctx, p, input); !errors.Is(err, f.ErrStorage) {
			t.Fatal("scanner absence must disable image upload")
		}
		for _, failure := range []error{f.ErrInvalid, f.ErrStorage} {
			blocked := app.NewService(nil, r, objects, testScanner{err: failure})
			u, err := blocked.CreateUpload(ctx, p, input)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = blocked.Upload(ctx, p, u.ID, data.Bytes()); !errors.Is(err, failure) {
				t.Fatal("scanner rejection lost")
			}
			var count int
			if err = pool.QueryRow(ctx, `SELECT count(*) FROM storage.files WHERE tenant_id=$1 AND object_key=$2`, p.TenantID, u.Object.Key).Scan(&count); err != nil || count != 0 {
				t.Fatal("unscanned file persisted")
			}
		}
	})
	t.Run("reserved help draft publishing preserves revisions", func(t *testing.T) {
		service := cms.NewService(pool, cmsAuth{iam.AuthenticatedContext{AuthContext: iam.AuthContext{User: iam.User{ID: p.UserID}, Tenant: iam.Tenant{ID: p.TenantID}, Permissions: []string{"app.content.read", "app.content.update", "app.content.publish", "app.content.delete"}}}})
		for _, slug := range []string{"faq", "contact-support", "about-us"} {
			body, _ := json.Marshal("# Private draft")
			input := cms.PageInput{Slug: slug, PageType: slug, Translations: map[string]cms.PageTranslation{"zh-CN": {Title: "帮助说明", BodyFormat: "markdown", Body: body}, "en-US": {Title: "Help", BodyFormat: "markdown", Body: body}}}
			first, err := service.SaveAdminPage(ctx, "admin", p.AppID, input, false)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = service.PublicPage(ctx, p.AppID, slug, "zh-CN"); err == nil {
				t.Fatal("draft leaked")
			}
			published, err := service.PublishAdminPage(ctx, "admin", p.AppID, slug, first.LockVersion)
			if err != nil {
				t.Fatal(err)
			}
			for locale, title := range map[string]string{"zh-CN": "帮助说明", "en-US": "Help"} {
				page, err := service.PublicPage(ctx, p.AppID, slug, locale)
				if err != nil || page.Title != title {
					t.Fatal(locale, err)
				}
			}
			input.LockVersion = published.LockVersion
			input.Translations["en-US"] = cms.PageTranslation{Title: "Unpublished new draft", BodyFormat: "markdown", Body: body}
			draft, err := service.SaveAdminPage(ctx, "admin", p.AppID, input, false)
			if err != nil {
				t.Fatal(err)
			}
			current, err := service.PublicPage(ctx, p.AppID, slug, "en-US")
			if err != nil || current.Title != "Help" {
				t.Fatal("new draft replaced published revision", err)
			}
			if _, err = service.PublishAdminPage(ctx, "admin", p.AppID, slug, draft.LockVersion); err != nil {
				t.Fatal(err)
			}
			current, err = service.PublicPage(ctx, p.AppID, slug, "en-US")
			if err != nil || current.Title != "Unpublished new draft" {
				t.Fatal(err)
			}
			if err = service.DeleteAdminPage(ctx, "admin", p.AppID, slug, draft.LockVersion+1); err == nil {
				t.Fatal("reserved help deleted")
			}
		}
	})
	u, fileID := upload()
	t.Run("upload replay and no avatar binding", func(t *testing.T) {
		again, e := s.Upload(ctx, p, u.ID, data.Bytes())
		if e != nil || again != fileID {
			t.Fatal("upload replay", e)
		}
		var avatar *uuid.UUID
		if e = pool.QueryRow(ctx, `SELECT avatar_file_id FROM iam.users WHERE id=$1`, p.UserID).Scan(&avatar); e != nil || avatar != nil {
			t.Fatal("avatar modified", e)
		}
	})
	t.Run("mime and size validation", func(t *testing.T) {
		truncated := data.Bytes()[:33]
		broken, err := s.CreateUpload(ctx, p, f.UploadInput{OriginalName: "header.png", MediaType: "image/png", SizeBytes: int64(len(truncated))})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.Upload(ctx, p, broken.ID, truncated); !errors.Is(err, f.ErrInvalid) {
			t.Fatal("truncated image accepted", err)
		}
		bad := bytes.Repeat([]byte{0}, data.Len())
		if _, e := s.Upload(ctx, p, u.ID, bad); !errors.Is(e, f.ErrInvalid) {
			t.Fatal(e)
		}
		if _, e := s.CreateUpload(ctx, p, f.UploadInput{OriginalName: "x.png", MediaType: "image/png", SizeBytes: f.MaxImageBytes + 1}); !errors.Is(e, f.ErrInvalid) {
			t.Fatal(e)
		}
	})
	x := f.Input{Description: "Private test feedback", Contact: "contact@example.test", Platform: "ios", AppVersion: "0.2.0", FileIDs: []uuid.UUID{fileID}}
	key := uuid.New()
	item, e := s.Create(ctx, p, x, key)
	if e != nil {
		t.Fatal(e)
	}
	t.Run("create replay and conflict", func(t *testing.T) {
		again, e := s.Create(ctx, p, x, key)
		if e != nil || again.ID != item.ID {
			t.Fatal(e)
		}
		changed := x
		changed.Description = "Different"
		if _, e = s.Create(ctx, p, changed, key); !errors.Is(e, f.ErrConflict) {
			t.Fatal(e)
		}
	})
	t.Run("concurrent submission", func(t *testing.T) {
		var wg sync.WaitGroup
		ids := make(chan uuid.UUID, 6)
		errs := make(chan error, 6)
		input := x
		input.FileIDs = []uuid.UUID{}
		idem := uuid.New()
		for i := 0; i < 6; i++ {
			wg.Add(1)
			go func() { defer wg.Done(); v, e := s.Create(ctx, p, input, idem); ids <- v.ID; errs <- e }()
		}
		wg.Wait()
		close(ids)
		close(errs)
		for e := range errs {
			if e != nil {
				t.Fatal(e)
			}
		}
		var first uuid.UUID
		for id := range ids {
			if first == uuid.Nil {
				first = id
			}
			if first != id {
				t.Fatal("duplicate rows")
			}
		}
	})
	t.Run("scope isolation", func(t *testing.T) {
		for _, other := range []f.Scope{{TenantID: uuid.New(), AppID: p.AppID, UserID: p.UserID}, {TenantID: p.TenantID, AppID: uuid.New(), UserID: p.UserID}, {TenantID: p.TenantID, AppID: p.AppID, UserID: uuid.New()}} {
			if _, e := s.Get(ctx, other, item.ID); !errors.Is(e, f.ErrNotFound) {
				t.Fatal(e)
			}
			if _, _, e := s.OpenFile(ctx, other, item.ID, fileID); !errors.Is(e, f.ErrNotFound) {
				t.Fatal(e)
			}
		}
	})
	t.Run("scan gate", func(t *testing.T) {
		_, waitingID := upload()
		for _, status := range []string{"pending", "infected", "failed", "skipped"} {
			exec(`UPDATE storage.files SET scan_status=$2 WHERE id IN ($1,$3)`, fileID, status, waitingID)
			if _, _, e := s.OpenFile(ctx, p, item.ID, fileID); !errors.Is(e, f.ErrNotFound) {
				t.Fatal(status, e)
			}
			draft := x
			draft.FileIDs = []uuid.UUID{waitingID}
			if _, e := s.Create(ctx, p, draft, uuid.New()); !errors.Is(e, f.ErrNotFound) {
				t.Fatal("unscanned attachment accepted", status, e)
			}
		}
		exec(`UPDATE storage.files SET scan_status='clean' WHERE id IN ($1,$2)`, fileID, waitingID)
	})
	t.Run("private files excluded from generic admin storage", func(t *testing.T) {
		_, err := db.New(pool).GetAdminFile(ctx, db.GetAdminFileParams{ID: fileID, TenantID: p.TenantID})
		if !errors.Is(err, pgx.ErrNoRows) {
			t.Fatal("generic storage exposes feedback", err)
		}
	})
	t.Run("inactive membership cannot submit", func(t *testing.T) {
		exec(`UPDATE app.user_memberships SET status='disabled' WHERE app_id=$1 AND user_id=$2`, p.AppID, p.UserID)
		draft := x
		draft.FileIDs = []uuid.UUID{}
		_, err := s.Create(ctx, p, draft, uuid.New())
		exec(`UPDATE app.user_memberships SET status='active' WHERE app_id=$1 AND user_id=$2`, p.AppID, p.UserID)
		if !errors.Is(err, f.ErrNotFound) {
			t.Fatal("inactive member submitted", err)
		}
	})
	admin := p
	admin.Admin = true
	t.Run("reply replay and optimistic concurrency", func(t *testing.T) {
		reply := f.ReplyInput{Body: "We fixed it", Status: "resolved", LockVersion: 1}
		k := uuid.New()
		got, e := s.Reply(ctx, admin, item.ID, reply, k)
		if e != nil || got.Status != "resolved" || len(got.Replies) != 1 {
			t.Fatal(e)
		}
		got, e = s.Reply(ctx, admin, item.ID, reply, k)
		if e != nil || len(got.Replies) != 1 {
			t.Fatal(e)
		}
		if _, e = s.Change(ctx, admin, item.ID, f.StatusInput{Status: "processing", LockVersion: 1}); !errors.Is(e, f.ErrConflict) {
			t.Fatal(e)
		}
	})
	t.Run("user reads staff reply and private image", func(t *testing.T) {
		got, e := s.Get(ctx, p, item.ID)
		if e != nil || len(got.Replies) != 1 || len(got.Attachments) != 1 {
			t.Fatal(e)
		}
		_, reader, e := s.OpenFile(ctx, p, item.ID, fileID)
		if e != nil {
			t.Fatal(e)
		}
		raw, _ := io.ReadAll(reader)
		_ = reader.Close()
		if !bytes.Equal(raw, data.Bytes()) {
			t.Fatal("image corrupted")
		}
	})
	t.Run("expired unattached cleanup preserves attachments", func(t *testing.T) {
		orphan, id := upload()
		exec(`UPDATE storage.upload_sessions SET created_at=now()-interval '2 days',expires_at=now()-interval '1 day' WHERE id IN ($1,$2)`, orphan.ID, u.ID)
		if e := cleanup.NewCleanup(pool, objects).Sweep(ctx); e != nil {
			t.Fatal(e)
		}
		var status string
		if e = pool.QueryRow(ctx, `SELECT status FROM storage.files WHERE id=$1`, id).Scan(&status); e != nil || status != "deleted" {
			t.Fatal(e, status)
		}
		if _, reader, e := s.OpenFile(ctx, p, item.ID, fileID); e != nil {
			t.Fatal(e)
		} else {
			_ = reader.Close()
		}
	})
	t.Run("list filters and sanitized audit", func(t *testing.T) {
		from := time.Now().Add(-time.Hour)
		page, e := s.List(ctx, admin, f.Filter{Status: "resolved", From: &from, Page: 1, PageSize: 20})
		if e != nil || page.Total != 1 || page.Items[0].Contact != "" {
			t.Fatal(e)
		}
		var leaked bool
		e = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM audit.operation_logs WHERE tenant_id=$1 AND (COALESCE(request_summary::text,'') LIKE '%contact@example.test%' OR COALESCE(after_data::text,'') LIKE '%Private test feedback%'))`, p.TenantID).Scan(&leaked)
		if e != nil || leaked {
			t.Fatal("audit leaked content", e)
		}
	})
}
