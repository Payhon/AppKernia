package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	"github.com/google/uuid"
	_ "golang.org/x/image/webp"
)

const uploadLifetime = 10 * time.Minute

type Service struct {
	repository domain.Repository
	objects    domain.ObjectStore
	enabled    bool
	clock      func() time.Time
}

type CreateAvatarUploadInput struct {
	OriginalName string
	MediaType    string
	SizeBytes    int64
}

type UploadTarget struct {
	ID        uuid.UUID
	UploadURL string
	Method    string
	ExpiresAt time.Time
}

func NewService(repository domain.Repository, objects domain.ObjectStore, enabled bool) *Service {
	return &Service{repository: repository, objects: objects, enabled: enabled, clock: time.Now}
}

func (service *Service) CreateAvatarUpload(
	ctx context.Context,
	principal domain.Principal,
	input CreateAvatarUploadInput,
) (UploadTarget, error) {
	if !service.enabled || service.objects == nil {
		return UploadTarget{}, domain.ErrFeatureDisabled
	}
	policy, err := service.objects.ResolvePolicy(ctx, principal.TenantID)
	if err != nil {
		return UploadTarget{}, err
	}
	name := strings.TrimSpace(filepath.Base(input.OriginalName))
	mediaType, extension, ok := normalizeImageType(input.MediaType)
	if !ok || !contains(policy.ImageMediaTypes, mediaType) || name == "" || name == "." || len(name) > 240 || input.SizeBytes <= 0 || input.SizeBytes > policy.MaxImageBytes || input.SizeBytes > domain.MaxAvatarBytes {
		return UploadTarget{}, domain.ErrUploadInvalid
	}
	expiresAt := service.clock().UTC().Add(uploadLifetime)
	objectKey := withPrefix(policy.PathPrefix, fmt.Sprintf("avatars/%s/%s/%s%s", principal.TenantID, principal.UserID, uuid.New(), extension))
	session, err := service.repository.CreateAvatarUpload(ctx, domain.CreateAvatarUpload{
		Principal: principal, OriginalName: name, MediaType: mediaType,
		ExpectedSize: input.SizeBytes, ObjectKey: objectKey, Provider: policy.Provider, Bucket: policy.Bucket, ExpiresAt: expiresAt,
	})
	if err != nil {
		return UploadTarget{}, err
	}
	return UploadTarget{
		ID: session.ID, Method: "PUT", ExpiresAt: session.ExpiresAt,
		UploadURL: "/me/avatar/upload-sessions/" + session.ID.String() + "/content",
	}, nil
}

func (service *Service) UploadAvatar(
	ctx context.Context,
	principal domain.Principal,
	uploadID uuid.UUID,
	content []byte,
	client domain.ClientMetadata,
) (uuid.UUID, error) {
	if !service.enabled || service.objects == nil {
		return uuid.Nil, domain.ErrFeatureDisabled
	}
	if uploadID == uuid.Nil || len(content) == 0 || int64(len(content)) > domain.MaxAvatarBytes || strings.TrimSpace(client.RequestID) == "" {
		return uuid.Nil, domain.ErrUploadInvalid
	}
	session, err := service.repository.GetAvatarUpload(ctx, principal, uploadID)
	if err != nil {
		return uuid.Nil, err
	}
	if session.ExpectedSize != int64(len(content)) || service.clock().UTC().After(session.ExpiresAt) {
		return uuid.Nil, domain.ErrUploadInvalid
	}
	configuredType, extension, ok := normalizeImageType(session.MediaType)
	if !ok {
		return uuid.Nil, domain.ErrUploadInvalid
	}
	decoded, format, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil || decoded.Width <= 0 || decoded.Height <= 0 || decoded.Width > 4096 || decoded.Height > 4096 || decoded.Width*decoded.Height > 16_000_000 {
		return uuid.Nil, domain.ErrUploadInvalid
	}
	actualType := map[string]string{"jpeg": "image/jpeg", "png": "image/png", "webp": "image/webp"}[format]
	if actualType == "" || actualType != configuredType {
		return uuid.Nil, domain.ErrUploadInvalid
	}
	digest := sha256.Sum256(content)
	ref := domain.ObjectRef{TenantID: principal.TenantID, Provider: session.Provider, Bucket: session.Bucket, Key: session.ObjectKey}
	if err = service.objects.Put(ctx, ref, content); err != nil {
		return uuid.Nil, err
	}
	completion, err := service.repository.CompleteAvatarUpload(ctx, domain.CompleteAvatarUpload{
		Principal: principal, ClientMetadata: client, UploadSessionID: uploadID,
		ObjectKey: session.ObjectKey, Provider: session.Provider, Bucket: session.Bucket, OriginalName: session.OriginalName,
		MediaType: configuredType, Extension: strings.TrimPrefix(extension, "."),
		SizeBytes: int64(len(content)), SHA256: digest[:],
	})
	if err != nil {
		_ = service.objects.Delete(ctx, ref)
		return uuid.Nil, err
	}
	if completion.ObjectKey != session.ObjectKey {
		_ = service.objects.Delete(ctx, ref)
	}
	return completion.FileID, nil
}

func (service *Service) OpenAvatar(ctx context.Context, principal domain.Principal) (domain.AvatarObject, io.ReadCloser, error) {
	if !service.enabled || service.objects == nil {
		return domain.AvatarObject{}, nil, domain.ErrFeatureDisabled
	}
	object, err := service.repository.GetAvatarObject(ctx, principal)
	if err != nil {
		return domain.AvatarObject{}, nil, err
	}
	reader, err := service.objects.Open(ctx, domain.ObjectRef{TenantID: principal.TenantID, Provider: object.Provider, Bucket: object.Bucket, Key: object.ObjectKey})
	if err != nil {
		return domain.AvatarObject{}, nil, err
	}
	return object, reader, nil
}

func withPrefix(prefix, key string) string {
	prefix = strings.Trim(strings.TrimSpace(prefix), "/")
	if prefix == "" {
		return key
	}
	return prefix + "/" + key
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func normalizeImageType(raw string) (string, string, bool) {
	switch strings.ToLower(strings.TrimSpace(strings.Split(raw, ";")[0])) {
	case "image/jpeg":
		return "image/jpeg", ".jpg", true
	case "image/png":
		return "image/png", ".png", true
	case "image/webp":
		return "image/webp", ".webp", true
	default:
		return "", "", false
	}
}
