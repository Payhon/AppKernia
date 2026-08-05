package repository

import (
	"context"
	"io"
	"testing"

	"github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	"github.com/google/uuid"
)

func TestLocalObjectStorePreventsTraversalAndRoundTripsPrivateObject(t *testing.T) {
	store, err := NewLocalObjectStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalObjectStore() error = %v", err)
	}
	tenantID := uuid.New()
	ref := func(key string) domain.ObjectRef {
		return domain.ObjectRef{TenantID: tenantID, Provider: "local", Bucket: "appkernia-local", Key: key}
	}
	if err = store.Put(context.Background(), ref("../escape"), []byte("secret")); err == nil {
		t.Fatal("Put() expected traversal rejection")
	}
	if err = store.Put(context.Background(), ref("avatars/tenant/user/avatar.png"), []byte("content")); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	reader, err := store.Open(context.Background(), ref("avatars/tenant/user/avatar.png"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = reader.Close() }()
	content, err := io.ReadAll(reader)
	if err != nil || string(content) != "content" {
		t.Fatalf("round trip = %q, %v", content, err)
	}
}
