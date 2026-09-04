package captcha

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func TestCodecRoundTripAndTamperRejection(t *testing.T) {
	key := bytes.Repeat([]byte{0x31}, 32)
	codec, err := NewCodec(key)
	if err != nil {
		t.Fatalf("create codec: %v", err)
	}
	now := time.Unix(1_800_000_000, 0).UTC()
	point := Point{X: 42, Y: 17}
	proof, err := NewProof("018f0000-0000-7000-8000-000000000001", bytes.Repeat([]byte{0x22}, 32), now, now.Add(5*time.Minute), Solution{
		Type: TypeSlide, CanvasWidth: 320, CanvasHeight: 180, Point: &point,
	})
	if err != nil {
		t.Fatalf("create proof: %v", err)
	}
	token, err := codec.Seal(proof)
	if err != nil {
		t.Fatalf("seal proof: %v", err)
	}
	opened, err := codec.Open(token)
	if err != nil {
		t.Fatalf("open proof: %v", err)
	}
	if opened.ChallengeID != proof.ChallengeID || !bytes.Equal(opened.ScopeHash, proof.ScopeHash) || opened.Solution.Point == nil || *opened.Solution.Point != point {
		t.Fatalf("proof changed during round trip: %+v", opened)
	}
	if len(TokenHash(token)) != 32 || !bytes.Equal(TokenHash(token), TokenHash(token)) {
		t.Fatal("token hash must be a stable SHA-256 digest")
	}
	tampered := []byte(token)
	if tampered[0] == 'A' {
		tampered[0] = 'B'
	} else {
		tampered[0] = 'A'
	}
	if _, err = codec.Open(string(tampered)); !errors.Is(err, ErrInvalidProof) {
		t.Fatalf("tampered token must fail authentication, got %v", err)
	}
}

func TestCodecRejectsWrongKeyAndInvalidProofs(t *testing.T) {
	if _, err := NewCodec(make([]byte, 31)); !errors.Is(err, ErrInvalidProof) {
		t.Fatalf("short key must fail, got %v", err)
	}
	first, err := NewCodec(bytes.Repeat([]byte{0x11}, 32))
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewCodec(bytes.Repeat([]byte{0x12}, 32))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_000, 0).UTC()
	point := Point{X: 1, Y: 1}
	proof, err := NewProof("challenge", bytes.Repeat([]byte{0x33}, 32), now, now.Add(time.Minute), Solution{
		Type: TypeDrag, CanvasWidth: 10, CanvasHeight: 10, Point: &point,
	})
	if err != nil {
		t.Fatal(err)
	}
	token, err := first.Seal(proof)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = second.Open(token); !errors.Is(err, ErrInvalidProof) {
		t.Fatalf("wrong key must fail, got %v", err)
	}
	if _, err = NewProof("challenge", []byte("short"), now, now.Add(time.Minute), proof.Solution); !errors.Is(err, ErrInvalidProof) {
		t.Fatalf("invalid scope must fail, got %v", err)
	}
}
