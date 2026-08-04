package repository

import (
	"bytes"
	"testing"
)

func TestAESGCMSealerUsesRandomNonceAndDoesNotExposePlaintext(t *testing.T) {
	sealer, err := NewAESGCMSealer(bytes.Repeat([]byte{0x42}, 32), 7)
	if err != nil {
		t.Fatal(err)
	}
	plaintext := []byte("configuration-secret-value")
	first, version, err := sealer.Seal(plaintext, "tenant-a")
	if err != nil {
		t.Fatal(err)
	}
	second, secondVersion, err := sealer.Seal(plaintext, "tenant-a")
	if err != nil {
		t.Fatal(err)
	}
	if version != 7 || secondVersion != 7 {
		t.Fatalf("key versions = %d, %d", version, secondVersion)
	}
	if bytes.Equal(first, second) {
		t.Fatal("ciphertexts must differ because every seal uses a fresh nonce")
	}
	if bytes.Contains(first, plaintext) || bytes.Contains(second, plaintext) {
		t.Fatal("ciphertext contains plaintext")
	}
	opened, err := sealer.Open(first, "tenant-a")
	if err != nil || !bytes.Equal(opened, plaintext) {
		t.Fatalf("open sealed value: %v", err)
	}
	if _, err := sealer.Open(first, "tenant-b"); err == nil {
		t.Fatal("expected AAD mismatch to fail")
	}
}

func TestAESGCMSealerRejectsInvalidKeyMaterial(t *testing.T) {
	if _, err := NewAESGCMSealer([]byte("too-short"), 1); err == nil {
		t.Fatal("short key must be rejected")
	}
	if _, err := NewAESGCMSealer(make([]byte, 32), 0); err == nil {
		t.Fatal("non-positive key version must be rejected")
	}
}
