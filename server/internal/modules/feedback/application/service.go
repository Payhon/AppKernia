package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	f "github.com/appkernia/appkernia/server/internal/modules/feedback/domain"
	iam "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	storage "github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	"github.com/google/uuid"
	_ "golang.org/x/image/webp"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type Authenticator interface {
	Authenticate(context.Context, string, string) (iam.AuthenticatedContext, error)
}
type Scanner interface {
	Scan(context.Context, []byte) error
}

type Service struct {
	scanner Scanner
	auth    Authenticator
	repo    f.Repository
	objects storage.ObjectStore
}

func NewService(a Authenticator, r f.Repository, o storage.ObjectStore, scanners ...Scanner) *Service {
	s := &Service{auth: a, repo: r, objects: o}
	if len(scanners) > 0 {
		s.scanner = scanners[0]
	}
	return s
}
func (s *Service) Scope(ctx context.Context, token string, appID uuid.UUID, permission, requestID string) (f.Scope, error) {
	audience := "ak-mobile"
	if permission != "" {
		audience = "ak-admin"
	}
	a, e := s.auth.Authenticate(ctx, token, audience)
	if e != nil {
		return f.Scope{}, e
	}
	if appID == uuid.Nil {
		return f.Scope{}, f.ErrInvalid
	}
	if permission != "" && !slices.Contains(a.Permissions, permission) {
		return f.Scope{}, f.ErrForbidden
	}
	if permission == "" && (a.AppID == nil || *a.AppID != appID) {
		return f.Scope{}, f.ErrForbidden
	}
	p := f.Scope{TenantID: a.Tenant.ID, AppID: appID, UserID: a.User.ID, SessionID: a.SessionID, Admin: permission != "", RequestID: requestID}
	return p, s.repo.CheckScope(ctx, p)
}
func (s *Service) List(ctx context.Context, p f.Scope, q f.Filter) (f.Page, error) {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}
	if q.Page < 1 || q.PageSize < 1 || q.PageSize > 100 || len([]rune(q.Query)) > 160 || (q.Status != "" && !f.ValidStatus(q.Status)) || (q.From != nil && q.To != nil && q.From.After(*q.To)) {
		return f.Page{}, f.ErrInvalid
	}
	return s.repo.List(ctx, p, q)
}
func (s *Service) Get(ctx context.Context, p f.Scope, id uuid.UUID) (f.Feedback, error) {
	return s.repo.Get(ctx, p, id, false)
}
func hash(x any) []byte { raw, _ := json.Marshal(x); v := sha256.Sum256(raw); return v[:] }
func ValidateInput(x f.Input) error {
	if len([]rune(x.Description)) < 1 || len([]rune(x.Description)) > 2000 || len([]rune(x.Contact)) > 200 || len(x.AppVersion) > 64 || !slices.Contains([]string{"android", "ios", "harmony", "unknown"}, x.Platform) || len(x.FileIDs) > 3 {
		return f.ErrInvalid
	}
	ids := map[uuid.UUID]bool{}
	for _, id := range x.FileIDs {
		if id == uuid.Nil || ids[id] {
			return f.ErrInvalid
		}
		ids[id] = true
	}
	return nil
}
func (s *Service) Create(ctx context.Context, p f.Scope, x f.Input, key uuid.UUID) (f.Feedback, error) {
	x.Description = strings.TrimSpace(x.Description)
	x.Contact = strings.TrimSpace(x.Contact)
	if ValidateInput(x) != nil || key == uuid.Nil || p.Admin {
		return f.Feedback{}, f.ErrInvalid
	}
	var id uuid.UUID
	e := s.repo.Transact(ctx, func(r f.Repository) error {
		existing, e := r.FindRequest(ctx, p, key, hash(x))
		if e != nil {
			return e
		}
		if existing != uuid.Nil {
			id = existing
			return nil
		}
		id, e = r.Create(ctx, p, x, key, hash(x))
		if e != nil {
			return e
		}
		if e = r.Attach(ctx, p, id, x.FileIDs); e != nil {
			return e
		}
		return r.Audit(ctx, p, id, "create")
	})
	if e != nil {
		return f.Feedback{}, e
	}
	return s.Get(ctx, p, id)
}
func (s *Service) Change(ctx context.Context, p f.Scope, id uuid.UUID, x f.StatusInput) (f.Feedback, error) {
	if !p.Admin || !f.ValidStatus(x.Status) || x.LockVersion < 1 {
		return f.Feedback{}, f.ErrInvalid
	}
	e := s.repo.Transact(ctx, func(r f.Repository) error {
		if e := r.Change(ctx, p, id, x.Status, x.LockVersion); e != nil {
			return e
		}
		return r.Audit(ctx, p, id, "update")
	})
	if e != nil {
		return f.Feedback{}, e
	}
	return s.Get(ctx, p, id)
}
func (s *Service) Reply(ctx context.Context, p f.Scope, id uuid.UUID, x f.ReplyInput, key uuid.UUID) (f.Feedback, error) {
	x.Body = strings.TrimSpace(x.Body)
	if !p.Admin || key == uuid.Nil || x.LockVersion < 1 || len([]rune(x.Body)) < 1 || len([]rune(x.Body)) > 2000 || !f.ValidStatus(x.Status) {
		return f.Feedback{}, f.ErrInvalid
	}
	e := s.repo.Transact(ctx, func(r f.Repository) error {
		if _, e := r.Get(ctx, p, id, true); e != nil {
			return e
		}
		found, e := r.FindReply(ctx, p, id, key, hash(x))
		if e != nil || found {
			return e
		}
		if e = r.Change(ctx, p, id, x.Status, x.LockVersion); e != nil {
			return e
		}
		if e = r.Reply(ctx, p, id, x, key, hash(x)); e != nil {
			return e
		}
		return r.Audit(ctx, p, id, "reply")
	})
	if e != nil {
		return f.Feedback{}, e
	}
	return s.Get(ctx, p, id)
}
func (s *Service) CreateUpload(ctx context.Context, p f.Scope, x f.UploadInput) (f.Upload, error) {
	if s.objects == nil || s.scanner == nil {
		return f.Upload{}, f.ErrStorage
	}
	policy, e := s.objects.ResolvePolicy(ctx, p.TenantID)
	if e != nil {
		return f.Upload{}, e
	}
	x.OriginalName = strings.TrimSpace(filepath.Base(x.OriginalName))
	if p.Admin || x.SizeBytes <= 0 || x.SizeBytes > f.MaxImageBytes || x.SizeBytes > policy.MaxImageBytes || !slices.Contains(policy.ImageMediaTypes, x.MediaType) || !slices.Contains([]string{"image/jpeg", "image/png", "image/webp"}, x.MediaType) || x.OriginalName == "" || len(x.OriginalName) > 240 {
		return f.Upload{}, f.ErrInvalid
	}
	u := f.Upload{ID: uuid.New(), ExpiresAt: time.Now().UTC().Add(24 * time.Hour), Name: x.OriginalName, MediaType: x.MediaType, Size: x.SizeBytes, Status: "initiated"}
	u.Object = storage.ObjectRef{TenantID: p.TenantID, Provider: policy.Provider, Bucket: policy.Bucket, Key: strings.Trim(policy.PathPrefix, "/") + "/feedback/" + p.AppID.String() + "/" + u.ID.String()}
	u.Object.Key = strings.TrimPrefix(u.Object.Key, "/")
	u.UploadURL = "/me/feedback-uploads/" + u.ID.String() + "/content"
	return s.repo.CreateUpload(ctx, p, u)
}
func (s *Service) Upload(ctx context.Context, p f.Scope, id uuid.UUID, data []byte) (uuid.UUID, error) {
	if s.objects == nil || s.scanner == nil {
		return uuid.Nil, f.ErrStorage
	}
	if p.Admin || len(data) == 0 || int64(len(data)) > f.MaxImageBytes {
		return uuid.Nil, f.ErrInvalid
	}
	var out uuid.UUID
	e := s.repo.Transact(ctx, func(r f.Repository) error {
		u, e := r.GetUpload(ctx, p, id, true)
		if e != nil {
			return e
		}
		if u.Size != int64(len(data)) {
			return f.ErrInvalid
		}
		decoded, format, e := image.DecodeConfig(bytes.NewReader(data))
		if e != nil || decoded.Width <= 0 || decoded.Height <= 0 || decoded.Width > 8192 || decoded.Height > 8192 || int64(decoded.Width)*int64(decoded.Height) > 32000000 {
			return f.ErrInvalid
		}
		if map[string]string{"jpeg": "image/jpeg", "png": "image/png", "webp": "image/webp"}[format] != u.MediaType {
			return f.ErrInvalid
		}
		if _, _, e := image.Decode(bytes.NewReader(data)); e != nil {
			return f.ErrInvalid
		}
		digest := sha256.Sum256(data)
		if u.FileID != nil {
			file, e := r.File(ctx, p, uuid.Nil, *u.FileID)
			if e != nil {
				return e
			}
			if !bytes.Equal(file.SHA256, digest[:]) {
				return f.ErrConflict
			}
			out = *u.FileID
			return nil
		}
		policy, e := s.objects.ResolvePolicy(ctx, p.TenantID)
		if e != nil {
			return e
		}
		if u.Size > policy.MaxImageBytes || !slices.Contains(policy.ImageMediaTypes, u.MediaType) {
			return f.ErrInvalid
		}
		if e = s.scanner.Scan(ctx, data); e != nil {
			return e
		}
		if e = s.objects.Put(ctx, u.Object, data); e != nil {
			return e
		}
		out, e = r.CompleteUpload(ctx, p, u, digest[:])
		return e
	})
	return out, e
}
func (s *Service) OpenFile(ctx context.Context, p f.Scope, feedbackID, fileID uuid.UUID) (f.File, io.ReadCloser, error) {
	if s.objects == nil {
		return f.File{}, nil, f.ErrStorage
	}
	x, e := s.repo.File(ctx, p, feedbackID, fileID)
	if e != nil {
		return x, nil, e
	}
	r, e := s.objects.Open(ctx, x.Object)
	return x, r, e
}
func (s *Service) CancelUpload(ctx context.Context, p f.Scope, id uuid.UUID) error {
	return s.repo.CancelUpload(ctx, p, id)
}
