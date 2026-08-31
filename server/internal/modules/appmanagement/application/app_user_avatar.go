package application

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	storagedomain "github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type AdminUserAvatar struct {
	FileID    uuid.UUID
	Provider  string
	Bucket    string
	ObjectKey string
	MediaType string
	SizeBytes int64
	SHA256    []byte
	UpdatedAt time.Time
}

func (s *Service) OpenAdminUserAvatar(ctx context.Context, token string, appID, userID uuid.UUID) (AdminUserAvatar, io.ReadCloser, error) {
	principal, err := s.authorizeAdmin(ctx, token, "app.user.read")
	if err != nil {
		return AdminUserAvatar{}, nil, err
	}
	if appID == uuid.Nil || userID == uuid.Nil || s.objects == nil {
		return AdminUserAvatar{}, nil, ErrAppNotFound
	}
	if err = s.requireAdminApp(ctx, appID, principal.Tenant.ID); err != nil {
		return AdminUserAvatar{}, nil, err
	}
	var out AdminUserAvatar
	var mediaType *string
	err = s.pool.QueryRow(ctx, `SELECT f.id,f.provider,f.bucket_name,f.object_key,f.media_type,f.size_bytes,f.sha256,f.updated_at
		FROM app.user_memberships m
		JOIN iam.users u ON u.id=m.user_id
		JOIN storage.files f ON f.id=u.avatar_file_id
		WHERE m.app_id=$1 AND m.tenant_id=$2 AND m.user_id=$3
		AND f.tenant_id=$2 AND f.owner_user_id=$3 AND f.status='ready'
		AND f.scan_status IN ('clean','skipped') AND f.deleted_at IS NULL
		AND lower(COALESCE(f.media_type,'')) IN ('image/jpeg','image/png','image/webp')`, appID, principal.Tenant.ID, userID).
		Scan(&out.FileID, &out.Provider, &out.Bucket, &out.ObjectKey, &mediaType, &out.SizeBytes, &out.SHA256, &out.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminUserAvatar{}, nil, ErrAppNotFound
	}
	if err != nil {
		return AdminUserAvatar{}, nil, err
	}
	if mediaType == nil {
		return AdminUserAvatar{}, nil, ErrAppNotFound
	}
	out.MediaType = strings.ToLower(strings.TrimSpace(*mediaType))
	reader, err := s.objects.Open(ctx, storagedomain.ObjectRef{TenantID: principal.Tenant.ID, Provider: out.Provider, Bucket: out.Bucket, Key: out.ObjectKey})
	if errors.Is(err, storagedomain.ErrObjectNotFound) {
		return AdminUserAvatar{}, nil, ErrAppNotFound
	}
	if err != nil {
		return AdminUserAvatar{}, nil, err
	}
	return out, reader, nil
}
