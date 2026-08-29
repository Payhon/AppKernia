package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	files "github.com/appkernia/appkernia/server/internal/modules/storageadmin/domain"
	"github.com/google/uuid"
)

const uploadLifetime = 24 * time.Hour

type Authenticator interface {
	Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error)
}

type Service struct {
	auth    Authenticator
	repo    files.Repository
	objects files.ObjectStore
	enabled bool
	clock   func() time.Time
}

func NewService(auth Authenticator, repo files.Repository, objects files.ObjectStore, enabled bool) *Service {
	return &Service{auth: auth, repo: repo, objects: objects, enabled: enabled, clock: time.Now}
}

func (s *Service) authorize(ctx context.Context, token, permission string) (iamdomain.AuthenticatedContext, error) {
	auth, err := s.auth.Authenticate(ctx, token, "ak-admin")
	if err != nil {
		return iamdomain.AuthenticatedContext{}, err
	}
	for _, candidate := range auth.Permissions {
		if candidate == permission {
			return auth, nil
		}
	}
	return iamdomain.AuthenticatedContext{}, files.ErrForbidden
}

func principal(auth iamdomain.AuthenticatedContext, requestID, ipAddress, userAgent string) files.Principal {
	return files.Principal{TenantID: auth.Tenant.ID, UserID: auth.User.ID, SessionID: auth.SessionID, RequestID: strings.TrimSpace(requestID), IPAddress: strings.TrimSpace(ipAddress), UserAgent: strings.TrimSpace(userAgent)}
}

func (s *Service) CreateUpload(ctx context.Context, token, requestID string, name, mediaType string, size int64) (files.UploadSession, error) {
	if !s.enabled || s.objects == nil {
		return files.UploadSession{}, files.ErrFeatureDisabled
	}
	auth, err := s.authorize(ctx, token, "storage.file.upload")
	if err != nil {
		return files.UploadSession{}, err
	}
	policy, err := s.objects.ResolvePolicy(ctx, auth.Tenant.ID)
	if err != nil {
		return files.UploadSession{}, err
	}
	name = strings.TrimSpace(filepath.Base(name))
	mediaType = normalizeMediaType(mediaType)
	if name == "" || name == "." || len(name) > 1000 || mediaType == "" || !contains(policy.FileMediaTypes, mediaType) || size <= 0 || size > policy.MaxFileBytes || size > files.MaxFileBytes || strings.TrimSpace(requestID) == "" {
		return files.UploadSession{}, files.ErrInvalid
	}
	objectKey := withPrefix(policy.PathPrefix, "files/"+auth.Tenant.ID.String()+"/"+uuid.New().String())
	return s.repo.CreateUpload(ctx, files.CreateUpload{Principal: principal(auth, requestID, "", ""), OriginalName: name, MediaType: mediaType, ExpectedSize: size, ObjectKey: objectKey, Provider: policy.Provider, Bucket: policy.Bucket, ExpiresAt: s.clock().UTC().Add(uploadLifetime)})
}

func (s *Service) UploadPolicy(ctx context.Context, token string) (files.UploadPolicy, error) {
	if !s.enabled || s.objects == nil {
		return files.UploadPolicy{}, files.ErrFeatureDisabled
	}
	auth, err := s.authorize(ctx, token, "storage.file.upload")
	if err != nil {
		return files.UploadPolicy{}, err
	}
	return s.objects.ResolvePolicy(ctx, auth.Tenant.ID)
}

func (s *Service) GetUpload(ctx context.Context, token string, id uuid.UUID) (files.UploadSession, error) {
	auth, err := s.authorize(ctx, token, "storage.file.upload")
	if err != nil {
		return files.UploadSession{}, err
	}
	if id == uuid.Nil {
		return files.UploadSession{}, files.ErrInvalid
	}
	return s.repo.GetUpload(ctx, auth.Tenant.ID, id)
}

func partKey(session files.UploadSession, number int32) string {
	return session.ObjectKey + ".parts/" + leftPad(number)
}

func objectRef(tenantID uuid.UUID, session files.UploadSession, key string) files.ObjectRef {
	return files.ObjectRef{TenantID: tenantID, Provider: session.Provider, Bucket: session.Bucket, Key: key}
}

func leftPad(value int32) string {
	return fmt.Sprintf("%06d", value)
}

func (s *Service) UploadPart(ctx context.Context, token string, uploadID uuid.UUID, number int32, content []byte) (files.Part, error) {
	if !s.enabled || s.objects == nil {
		return files.Part{}, files.ErrFeatureDisabled
	}
	auth, err := s.authorize(ctx, token, "storage.file.upload")
	if err != nil {
		return files.Part{}, err
	}
	session, err := s.repo.GetUpload(ctx, auth.Tenant.ID, uploadID)
	if err != nil {
		return files.Part{}, err
	}
	partCount := int32((session.ExpectedSize + session.PartSize - 1) / session.PartSize)
	expected := session.PartSize
	if number == partCount {
		expected = session.ExpectedSize - int64(partCount-1)*session.PartSize
	}
	if number < 1 || number > partCount || int64(len(content)) != expected || s.clock().UTC().After(session.ExpiresAt) {
		return files.Part{}, files.ErrInvalid
	}
	digest := sha256.Sum256(content)
	part := files.Part{PartNumber: number, SizeBytes: int64(len(content)), ETag: hex.EncodeToString(digest[:]), Checksum: digest[:]}
	partRef := objectRef(auth.Tenant.ID, session, partKey(session, number))
	if err = s.objects.Put(ctx, partRef, content); err != nil {
		return files.Part{}, err
	}
	if err = s.repo.UpsertPart(ctx, auth.Tenant.ID, uploadID, part); err != nil {
		_ = s.objects.Delete(ctx, partRef)
		return files.Part{}, err
	}
	return part, nil
}

func (s *Service) CancelUpload(ctx context.Context, token, requestID, ipAddress, userAgent string, id uuid.UUID) error {
	if !s.enabled || s.objects == nil {
		return files.ErrFeatureDisabled
	}
	auth, err := s.authorize(ctx, token, "storage.file.upload")
	if err != nil {
		return err
	}
	session, err := s.repo.AbortUpload(ctx, principal(auth, requestID, ipAddress, userAgent), id)
	if err != nil {
		return err
	}
	for _, part := range session.UploadedParts {
		_ = s.objects.Delete(ctx, objectRef(auth.Tenant.ID, session, partKey(session, part.PartNumber)))
	}
	_ = s.objects.Delete(ctx, objectRef(auth.Tenant.ID, session, session.ObjectKey))
	return nil
}

func (s *Service) CompleteUpload(ctx context.Context, token, requestID, ipAddress, userAgent string, id uuid.UUID) (files.File, error) {
	if !s.enabled || s.objects == nil {
		return files.File{}, files.ErrFeatureDisabled
	}
	auth, err := s.authorize(ctx, token, "storage.file.upload")
	if err != nil {
		return files.File{}, err
	}
	session, err := s.repo.GetUpload(ctx, auth.Tenant.ID, id)
	if err != nil {
		return files.File{}, err
	}
	partCount := int32((session.ExpectedSize + session.PartSize - 1) / session.PartSize)
	if int32(len(session.UploadedParts)) != partCount {
		return files.File{}, files.ErrUploadIncomplete
	}
	var assembled bytes.Buffer
	hasher := sha256.New()
	for number := int32(1); number <= partCount; number++ {
		part := session.UploadedParts[number-1]
		if part.PartNumber != number {
			return files.File{}, files.ErrUploadIncomplete
		}
		reader, openErr := s.objects.Open(ctx, objectRef(auth.Tenant.ID, session, partKey(session, number)))
		if openErr != nil {
			return files.File{}, files.ErrUploadIncomplete
		}
		_, copyErr := io.Copy(io.MultiWriter(&assembled, hasher), io.LimitReader(reader, session.PartSize+1))
		_ = reader.Close()
		if copyErr != nil {
			return files.File{}, copyErr
		}
	}
	content := assembled.Bytes()
	if int64(len(content)) != session.ExpectedSize {
		return files.File{}, files.ErrUploadIncomplete
	}
	actual := normalizeMediaType(http.DetectContentType(content[:min(len(content), 512)]))
	if !mediaCompatible(session.MediaType, actual) {
		return files.File{}, files.ErrInvalid
	}
	finalRef := objectRef(auth.Tenant.ID, session, session.ObjectKey)
	if err = s.objects.Put(ctx, finalRef, content); err != nil {
		return files.File{}, err
	}
	extension := strings.TrimPrefix(strings.ToLower(filepath.Ext(session.OriginalName)), ".")
	file, err := s.repo.CompleteUpload(ctx, files.CompleteUpload{Principal: principal(auth, requestID, ipAddress, userAgent), UploadID: id, ObjectKey: session.ObjectKey, Provider: session.Provider, Bucket: session.Bucket, MediaType: actual, Extension: extension, SizeBytes: int64(len(content)), SHA256: hasher.Sum(nil), ScanStatus: "skipped"})
	if err != nil {
		_ = s.objects.Delete(ctx, finalRef)
		return files.File{}, err
	}
	if file.ObjectKey != session.ObjectKey {
		_ = s.objects.Delete(ctx, finalRef)
	}
	for _, part := range session.UploadedParts {
		_ = s.objects.Delete(ctx, objectRef(auth.Tenant.ID, session, partKey(session, part.PartNumber)))
	}
	return file, nil
}

func (s *Service) ListFiles(ctx context.Context, token string, filter files.FileFilter) (files.FilePage, error) {
	auth, err := s.authorize(ctx, token, "storage.file.read")
	if err != nil {
		return files.FilePage{}, err
	}
	filter.Query, filter.Status, filter.ScanStatus, filter.MediaType, filter.Provider = strings.TrimSpace(filter.Query), strings.TrimSpace(filter.Status), strings.TrimSpace(filter.ScanStatus), strings.TrimSpace(filter.MediaType), strings.TrimSpace(filter.Provider)
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.PageSize == 0 {
		filter.PageSize = 20
	}
	if filter.Page < 1 || filter.PageSize < 1 || filter.PageSize > 100 || len(filter.Query) > 160 || !oneOf(filter.Status, "", "pending", "ready", "quarantined") || !oneOf(filter.ScanStatus, "", "pending", "clean", "infected", "failed", "skipped") || !oneOf(filter.Provider, "", "local", "s3", "minio") || (filter.CreatedFrom != nil && filter.CreatedTo != nil && filter.CreatedFrom.After(*filter.CreatedTo)) {
		return files.FilePage{}, files.ErrInvalid
	}
	return s.repo.ListFiles(ctx, auth.Tenant.ID, filter)
}

func (s *Service) GetFile(ctx context.Context, token string, id uuid.UUID) (files.File, error) {
	auth, err := s.authorize(ctx, token, "storage.file.read")
	if err != nil {
		return files.File{}, err
	}
	return s.repo.GetFile(ctx, auth.Tenant.ID, id)
}
func (s *Service) ListUsages(ctx context.Context, token string, id uuid.UUID) ([]files.Usage, error) {
	auth, err := s.authorize(ctx, token, "storage.file.read")
	if err != nil {
		return nil, err
	}
	return s.repo.ListUsages(ctx, auth.Tenant.ID, id)
}
func (s *Service) OpenDownload(ctx context.Context, token string, id uuid.UUID) (files.File, io.ReadCloser, error) {
	if !s.enabled || s.objects == nil {
		return files.File{}, nil, files.ErrFeatureDisabled
	}
	auth, err := s.authorize(ctx, token, "storage.file.download")
	if err != nil {
		return files.File{}, nil, err
	}
	file, err := s.repo.GetFile(ctx, auth.Tenant.ID, id)
	if err != nil {
		return files.File{}, nil, err
	}
	if file.Status != "ready" || !oneOf(file.ScanStatus, "clean", "skipped") {
		return files.File{}, nil, files.ErrScanBlocked
	}
	reader, err := s.objects.Open(ctx, files.ObjectRef{TenantID: auth.Tenant.ID, Provider: file.Provider, Bucket: file.Bucket, Key: file.ObjectKey})
	if err != nil {
		return files.File{}, nil, err
	}
	return file, reader, nil
}
func (s *Service) DeleteFile(ctx context.Context, token, requestID, ipAddress, userAgent string, id uuid.UUID) error {
	if !s.enabled || s.objects == nil {
		return files.ErrFeatureDisabled
	}
	auth, err := s.authorize(ctx, token, "storage.file.delete")
	if err != nil {
		return err
	}
	file, err := s.repo.DeleteFile(ctx, principal(auth, requestID, ipAddress, userAgent), id)
	if err != nil {
		return err
	}
	return s.objects.Delete(ctx, files.ObjectRef{TenantID: auth.Tenant.ID, Provider: file.Provider, Bucket: file.Bucket, Key: file.ObjectKey})
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

func normalizeMediaType(value string) string {
	value = strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0]))
	if _, _, err := mime.ParseMediaType(value); err != nil {
		return ""
	}
	return value
}
func mediaCompatible(expected, actual string) bool {
	if expected == actual {
		return true
	}
	// Browsers use application/octet-stream when they cannot determine a more
	// specific media type. The server still records the detected type and keeps
	// the scan gate in front of every download/selection path.
	if expected == "application/octet-stream" {
		return true
	}
	archiveMedia := func(value string) bool {
		return value == "application/zip" || value == "application/x-zip-compressed" || value == "application/vnd.android.package-archive"
	}
	if archiveMedia(expected) && archiveMedia(actual) {
		return true
	}
	return (expected == "text/csv" || expected == "application/json") && actual == "text/plain"
}
func oneOf(value string, candidates ...string) bool {
	for _, c := range candidates {
		if value == c {
			return true
		}
	}
	return false
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
