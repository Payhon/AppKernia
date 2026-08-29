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
	filter    files.FileFilter
}

func (r *fakeRepo) CreateUpload(_ context.Context, in files.CreateUpload) (files.UploadSession, error) {
	r.session = files.UploadSession{ID: uuid.New(), OriginalName: in.OriginalName, MediaType: in.MediaType, ExpectedSize: in.ExpectedSize, PartSize: files.PartSize, Status: "initiated", Provider: in.Provider, Bucket: in.Bucket, ObjectKey: in.ObjectKey, ExpiresAt: in.ExpiresAt, UploadedParts: []files.Part{}}
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
func (r *fakeRepo) ListFiles(_ context.Context, _ uuid.UUID, filter files.FileFilter) (files.FilePage, error) {
	r.filter = filter
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

func (m *memoryObjects) ResolvePolicy(context.Context, uuid.UUID) (files.UploadPolicy, error) {
	return files.UploadPolicy{Provider: "local", Bucket: "appkernia-local", MaxImageBytes: 5 * 1024 * 1024, MaxFileBytes: files.MaxFileBytes, ImageMediaTypes: []string{"image/jpeg", "image/png", "image/webp"}, FileMediaTypes: []string{"text/plain", "application/octet-stream"}, ConfigurationSafe: true}, nil
}
func (m *memoryObjects) Put(_ context.Context, ref files.ObjectRef, value []byte) error {
	m.values[ref.Key] = bytes.Clone(value)
	return nil
}
func (m *memoryObjects) Open(_ context.Context, ref files.ObjectRef) (io.ReadCloser, error) {
	value, ok := m.values[ref.Key]
	if !ok {
		return nil, files.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(value)), nil
}
func (m *memoryObjects) Delete(_ context.Context, ref files.ObjectRef) error {
	delete(m.values, ref.Key)
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

func TestListFilesValidatesCreatedRange(t *testing.T) {
	repo := &fakeRepo{}
	service := NewService(fakeAuth{permissions: []string{"storage.file.read"}}, repo, &memoryObjects{values: map[string][]byte{}}, true)
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 31, 23, 59, 59, 0, time.UTC)
	if _, err := service.ListFiles(context.Background(), "token", files.FileFilter{CreatedFrom: &from, CreatedTo: &to}); err != nil {
		t.Fatal(err)
	}
	if repo.filter.CreatedFrom == nil || !repo.filter.CreatedFrom.Equal(from) || repo.filter.CreatedTo == nil || !repo.filter.CreatedTo.Equal(to) || repo.filter.Page != 1 || repo.filter.PageSize != 20 {
		t.Fatalf("unexpected normalized filter %#v", repo.filter)
	}
	if _, err := service.ListFiles(context.Background(), "token", files.FileFilter{CreatedFrom: &to, CreatedTo: &from}); err != files.ErrInvalid {
		t.Fatalf("reversed range error=%v", err)
	}
}

func TestMediaCompatibleAllowsUnknownBrowserTypeButRejectsMismatch(t *testing.T) {
	if !mediaCompatible("application/octet-stream", "text/plain") {
		t.Fatal("unknown browser media type should defer to server detection")
	}
	if mediaCompatible("image/png", "text/plain") {
		t.Fatal("specific mismatched media types must be rejected")
	}
	if !mediaCompatible("application/vnd.android.package-archive", "application/zip") {
		t.Fatal("APK browser MIME should accept the server-detected ZIP container")
	}
}
