package application

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"testing"
	"time"

	"github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	"github.com/google/uuid"
)

type fakeRepository struct {
	session         domain.AvatarUploadSession
	completed       *domain.CompleteAvatarUpload
	storedObjectKey string
}

func (repository *fakeRepository) CreateAvatarUpload(_ context.Context, input domain.CreateAvatarUpload) (domain.AvatarUploadSession, error) {
	repository.session = domain.AvatarUploadSession{
		ID: uuid.New(), Provider: input.Provider, Bucket: input.Bucket, ObjectKey: input.ObjectKey, OriginalName: input.OriginalName,
		MediaType: input.MediaType, ExpectedSize: input.ExpectedSize, ExpiresAt: input.ExpiresAt,
	}
	return repository.session, nil
}
func (repository *fakeRepository) GetAvatarUpload(_ context.Context, _ domain.Principal, _ uuid.UUID) (domain.AvatarUploadSession, error) {
	return repository.session, nil
}
func (repository *fakeRepository) CompleteAvatarUpload(_ context.Context, input domain.CompleteAvatarUpload) (domain.AvatarCompletion, error) {
	repository.completed = &input
	objectKey := repository.storedObjectKey
	if objectKey == "" {
		objectKey = input.ObjectKey
	}
	return domain.AvatarCompletion{FileID: uuid.New(), ObjectKey: objectKey}, nil
}
func (repository *fakeRepository) GetAvatarObject(context.Context, domain.Principal) (domain.AvatarObject, error) {
	return domain.AvatarObject{}, domain.ErrAvatarNotFound
}

type memoryStore struct{ values map[string][]byte }

func (store *memoryStore) ResolvePolicy(context.Context, uuid.UUID) (domain.UploadPolicy, error) {
	return domain.UploadPolicy{Provider: "local", Bucket: "appkernia-local", MaxImageBytes: domain.MaxAvatarBytes, MaxFileBytes: domain.MaxFileBytes, ImageMediaTypes: []string{"image/jpeg", "image/png", "image/webp"}, FileMediaTypes: []string{"application/octet-stream"}, ConfigurationSafe: true}, nil
}
func (store *memoryStore) Put(_ context.Context, ref domain.ObjectRef, content []byte) error {
	store.values[ref.Key] = bytes.Clone(content)
	return nil
}
func (store *memoryStore) Open(_ context.Context, ref domain.ObjectRef) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(store.values[ref.Key])), nil
}
func (store *memoryStore) Delete(_ context.Context, ref domain.ObjectRef) error {
	delete(store.values, ref.Key)
	return nil
}

func TestAvatarUploadValidatesImageAndCompletes(t *testing.T) {
	content := testPNG(t)
	repository := &fakeRepository{}
	objects := &memoryStore{values: map[string][]byte{}}
	service := NewService(repository, objects, true)
	service.clock = func() time.Time { return time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC) }
	principal := domain.Principal{UserID: uuid.New(), TenantID: uuid.New(), SessionID: uuid.New()}
	target, err := service.CreateAvatarUpload(context.Background(), principal, CreateAvatarUploadInput{
		OriginalName: "avatar.png", MediaType: "image/png", SizeBytes: int64(len(content)),
	})
	if err != nil {
		t.Fatalf("CreateAvatarUpload() error = %v", err)
	}
	if target.Method != "PUT" || target.ID == uuid.Nil || target.UploadURL == "" {
		t.Fatalf("unexpected upload target: %#v", target)
	}
	if _, err = service.UploadAvatar(context.Background(), principal, target.ID, content, domain.ClientMetadata{RequestID: "request-1"}); err != nil {
		t.Fatalf("UploadAvatar() error = %v", err)
	}
	if repository.completed == nil || repository.completed.UserID != principal.UserID || repository.completed.TenantID != principal.TenantID {
		t.Fatalf("completion did not preserve the authenticated scope: %#v", repository.completed)
	}
	if !bytes.Equal(objects.values[repository.session.ObjectKey], content) {
		t.Fatal("object content was not stored")
	}
}

func TestAvatarUploadRejectsDisabledInvalidAndSpoofedContent(t *testing.T) {
	principal := domain.Principal{UserID: uuid.New(), TenantID: uuid.New(), SessionID: uuid.New()}
	repository := &fakeRepository{}
	objects := &memoryStore{values: map[string][]byte{}}
	disabled := NewService(repository, objects, false)
	if _, err := disabled.CreateAvatarUpload(context.Background(), principal, CreateAvatarUploadInput{OriginalName: "a.png", MediaType: "image/png", SizeBytes: 1}); err != domain.ErrFeatureDisabled {
		t.Fatalf("disabled CreateAvatarUpload() error = %v", err)
	}
	enabled := NewService(repository, objects, true)
	if _, err := enabled.CreateAvatarUpload(context.Background(), principal, CreateAvatarUploadInput{OriginalName: "a.svg", MediaType: "image/svg+xml", SizeBytes: 20}); err != domain.ErrUploadInvalid {
		t.Fatalf("invalid CreateAvatarUpload() error = %v", err)
	}
	repository.session = domain.AvatarUploadSession{
		ID: uuid.New(), Provider: "local", Bucket: "appkernia-local", ObjectKey: "avatars/test.png", OriginalName: "test.png",
		MediaType: "image/png", ExpectedSize: 8, ExpiresAt: time.Now().Add(time.Minute),
	}
	if _, err := enabled.UploadAvatar(context.Background(), principal, repository.session.ID, []byte("notimage"), domain.ClientMetadata{RequestID: "request-2"}); err != domain.ErrUploadInvalid {
		t.Fatalf("spoofed UploadAvatar() error = %v", err)
	}
	if len(objects.values) != 0 {
		t.Fatal("invalid image must not reach object storage")
	}
}

func TestAvatarUploadDeletesUnreferencedObjectWhenDatabaseReusesDeduplicatedFile(t *testing.T) {
	content := testPNG(t)
	repository := &fakeRepository{storedObjectKey: "avatars/existing.png"}
	objects := &memoryStore{values: map[string][]byte{}}
	service := NewService(repository, objects, true)
	principal := domain.Principal{UserID: uuid.New(), TenantID: uuid.New(), SessionID: uuid.New()}
	target, err := service.CreateAvatarUpload(context.Background(), principal, CreateAvatarUploadInput{
		OriginalName: "avatar.png", MediaType: "image/png", SizeBytes: int64(len(content)),
	})
	if err != nil {
		t.Fatalf("CreateAvatarUpload() error = %v", err)
	}
	if _, err = service.UploadAvatar(context.Background(), principal, target.ID, content, domain.ClientMetadata{RequestID: "request-dedup"}); err != nil {
		t.Fatalf("UploadAvatar() error = %v", err)
	}
	if _, exists := objects.values[repository.session.ObjectKey]; exists {
		t.Fatal("deduplicated upload object was not deleted")
	}
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	value := image.NewRGBA(image.Rect(0, 0, 4, 4))
	value.Set(0, 0, color.RGBA{R: 37, G: 99, B: 235, A: 255})
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, value); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}
	return buffer.Bytes()
}
