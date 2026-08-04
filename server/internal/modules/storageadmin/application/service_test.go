package application

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	files "github.com/appkernia/appkernia/server/internal/modules/storageadmin/domain"
	"github.com/google/uuid"
)

type fakeAuth struct{ permissions []string }

func (a fakeAuth) Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error) {
	return iamdomain.AuthenticatedContext{AuthContext: iamdomain.AuthContext{User: iamdomain.User{ID: uuid.MustParse("00000000-0000-0000-0000-000000000101")}, Tenant: iamdomain.Tenant{ID: uuid.MustParse("00000000-0000-0000-0000-000000000201")}, Permissions: a.permissions}, SessionID: uuid.MustParse("00000000-0000-0000-0000-000000000301")}, nil
}

type fakeRepo struct {
	session   files.UploadSession
	completed files.File
	file      files.File
}

func (r *fakeRepo) CreateUpload(_ context.Context, in files.CreateUpload) (files.UploadSession, error) {
	r.session = files.UploadSession{ID: uuid.New(), OriginalName: in.OriginalName, MediaType: in.MediaType, ExpectedSize: in.ExpectedSize, PartSize: files.PartSize, Status: "initiated", ObjectKey: in.ObjectKey, ExpiresAt: in.ExpiresAt, UploadedParts: []files.Part{}}
	return r.session, nil
}
func (r *fakeRepo) GetUpload(context.Context, uuid.UUID, uuid.UUID) (files.UploadSession, error) {
	return r.session, nil
}
func (r *fakeRepo) UpsertPart(_ context.Context, _ uuid.UUID, _ uuid.UUID, p files.Part) error {
	for i, x := range r.session.UploadedParts {
		if x.PartNumber == p.PartNumber {
			r.session.UploadedParts[i] = p
			return nil
		}
	}
	r.session.UploadedParts = append(r.session.UploadedParts, p)
	return nil
}
func (r *fakeRepo) AbortUpload(context.Context, files.Principal, uuid.UUID) (files.UploadSession, error) {
	return r.session, nil
}
func (r *fakeRepo) CompleteUpload(_ context.Context, in files.CompleteUpload) (files.File, error) {
	r.completed = files.File{ID: uuid.New(), OriginalName: r.session.OriginalName, MediaType: in.MediaType, SizeBytes: in.SizeBytes, Status: "ready", ScanStatus: in.ScanStatus, ObjectKey: in.ObjectKey}
	return r.completed, nil
}
func (r *fakeRepo) ListFiles(context.Context, uuid.UUID, files.FileFilter) (files.FilePage, error) {
	return files.FilePage{}, nil
}
func (r *fakeRepo) GetFile(context.Context, uuid.UUID, uuid.UUID) (files.File, error) {
	return r.file, nil
}
func (r *fakeRepo) ListUsages(context.Context, uuid.UUID, uuid.UUID) ([]files.Usage, error) {
	return nil, nil
}
func (r *fakeRepo) DeleteFile(context.Context, files.Principal, uuid.UUID) (files.File, error) {
	return r.file, nil
}

type memoryObjects struct{ values map[string][]byte }

func (m *memoryObjects) Put(_ context.Context, key string, value []byte) error {
	m.values[key] = bytes.Clone(value)
	return nil
}
func (m *memoryObjects) Open(_ context.Context, key string) (io.ReadCloser, error) {
	value, ok := m.values[key]
	if !ok {
		return nil, files.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(value)), nil
}
func (m *memoryObjects) Delete(_ context.Context, key string) error {
	delete(m.values, key)
	return nil
}

func TestMultipartUploadResumeAndComplete(t *testing.T) {
	repo := &fakeRepo{}
	objects := &memoryObjects{values: map[string][]byte{}}
	service := NewService(fakeAuth{permissions: []string{"storage.file.upload"}}, repo, objects, true)
	service.clock = func() time.Time { return time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC) }
	content := bytes.Repeat([]byte("a"), int(files.PartSize+1024))
	session, err := service.CreateUpload(context.Background(), "token", "request-1", "report.txt", "text/plain", int64(len(content)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.UploadPart(context.Background(), "token", session.ID, 1, content[:files.PartSize]); err != nil {
		t.Fatal(err)
	}
	resumed, err := service.GetUpload(context.Background(), "token", session.ID)
	if err != nil || len(resumed.UploadedParts) != 1 {
		t.Fatalf("resume state=%#v err=%v", resumed, err)
	}
	if _, err = service.UploadPart(context.Background(), "token", session.ID, 2, content[files.PartSize:]); err != nil {
		t.Fatal(err)
	}
	completed, err := service.CompleteUpload(context.Background(), "token", "request-2", "127.0.0.1", "test", session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != "ready" || completed.ScanStatus != "skipped" || completed.SizeBytes != int64(len(content)) {
		t.Fatalf("unexpected completion %#v", completed)
	}
	if !bytes.Equal(objects.values[session.ObjectKey], content) {
		t.Fatal("assembled object differs")
	}
}

func TestMultipartRejectsWrongPartAndPermission(t *testing.T) {
	repo := &fakeRepo{}
	objects := &memoryObjects{values: map[string][]byte{}}
	denied := NewService(fakeAuth{}, repo, objects, true)
	if _, err := denied.CreateUpload(context.Background(), "token", "request", "a.txt", "text/plain", 1); err != files.ErrForbidden {
		t.Fatalf("permission error=%v", err)
	}
	allowed := NewService(fakeAuth{permissions: []string{"storage.file.upload"}}, repo, objects, true)
	session, err := allowed.CreateUpload(context.Background(), "token", "request", "a.txt", "text/plain", 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = allowed.UploadPart(context.Background(), "token", session.ID, 1, []byte("x")); err != files.ErrInvalid {
		t.Fatalf("wrong part size error=%v", err)
	}
}

func TestDownloadScanGate(t *testing.T) {
	repo := &fakeRepo{file: files.File{ID: uuid.New(), Status: "ready", ScanStatus: "pending", ObjectKey: "pending"}}
	service := NewService(fakeAuth{permissions: []string{"storage.file.download"}}, repo, &memoryObjects{values: map[string][]byte{}}, true)
	if _, _, err := service.OpenDownload(context.Background(), "token", repo.file.ID); err != files.ErrScanBlocked {
		t.Fatalf("scan gate error=%v", err)
	}
}

func TestMediaCompatibleAllowsUnknownBrowserTypeButRejectsMismatch(t *testing.T) {
	if !mediaCompatible("application/octet-stream", "text/plain") {
		t.Fatal("unknown browser media type should defer to server detection")
	}
	if mediaCompatible("image/png", "text/plain") {
		t.Fatal("specific mismatched media types must be rejected")
	}
}
