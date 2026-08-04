package repository

import (
	"context"
	"io"
	"testing"
)

func TestLocalObjectStorePreventsTraversalAndRoundTripsPrivateObject(t *testing.T) {
	store, err := NewLocalObjectStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalObjectStore() error = %v", err)
	}
	if err = store.Put(context.Background(), "../escape", []byte("secret")); err == nil {
		t.Fatal("Put() expected traversal rejection")
	}
	if err = store.Put(context.Background(), "avatars/tenant/user/avatar.png", []byte("content")); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	reader, err := store.Open(context.Background(), "avatars/tenant/user/avatar.png")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = reader.Close() }()
	content, err := io.ReadAll(reader)
	if err != nil || string(content) != "content" {
		t.Fatalf("round trip = %q, %v", content, err)
	}
}
