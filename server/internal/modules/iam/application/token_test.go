package application

import (
	"testing"

	"github.com/google/uuid"
)

func TestAccessTokenAudienceIsolation(t *testing.T) {
	issuer, err := NewDevelopmentTokenIssuer()
	if err != nil {
		t.Fatalf("create issuer: %v", err)
	}
	token, _, err := issuer.Issue(uuid.New(), uuid.New(), uuid.New(), "ak-admin", 1)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	if _, err = issuer.Verify(token, "ak-admin"); err != nil {
		t.Fatalf("verify intended audience: %v", err)
	}
	if _, err = issuer.Verify(token, "ak-mobile"); err == nil {
		t.Fatal("admin token must not verify for mobile audience")
	}
}

func TestOpaqueTokenStoresOnlyHash(t *testing.T) {
	plainText, hash, err := NewOpaqueToken()
	if err != nil {
		t.Fatalf("generate opaque token: %v", err)
	}
	if plainText == "" || len(hash) != 32 {
		t.Fatal("unexpected opaque token material")
	}
	if string(hash) == plainText {
		t.Fatal("stored token material must not equal plaintext")
	}
	if got := HashOpaqueToken(plainText); string(got) != string(hash) {
		t.Fatal("opaque token hash is not deterministic")
	}
}
