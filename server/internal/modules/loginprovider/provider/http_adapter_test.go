package provider

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	login "github.com/appkernia/appkernia/server/internal/modules/loginprovider/domain"
)

func jwkForTest(key *rsa.PublicKey, kid string) map[string]string {
	exponent := big.NewInt(int64(key.E)).Bytes()
	return map[string]string{"kty": "RSA", "kid": kid, "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(exponent)}
}

func TestUnknownCachedKIDRefreshesJWKSOnce(t *testing.T) {
	first, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	second, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		requests++
		key, kid := &first.PublicKey, "old"
		if requests > 1 {
			key, kid = &second.PublicKey, "new"
		}
		_ = json.NewEncoder(response).Encode(map[string]any{"keys": []any{jwkForTest(key, kid)}})
	}))
	defer server.Close()
	adapter := newHTTPAdapter(&http.Client{Timeout: time.Second}, "https://api.example.test/callback", endpoints{})
	if _, err = adapter.key(context.Background(), server.URL, "old"); err != nil {
		t.Fatal(err)
	}
	if _, err = adapter.key(context.Background(), server.URL, "new"); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("JWKS requests = %d, want exactly one refresh", requests)
	}
}

func TestWechatVerificationUsesFixedShapeAndReturnsScopedIssuer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("appid") != "wx1234567890ABCDEF" || request.URL.Query().Get("secret") != "provider-secret" {
			t.Fatal("missing provider exchange parameters")
		}
		_ = json.NewEncoder(response).Encode(map[string]string{"openid": "openid-1", "unionid": "union-1"})
	}))
	defer server.Close()
	adapter := newHTTPAdapter(&http.Client{Timeout: time.Second}, "https://api.example.test/api/v1/auth/oauth/github/browser-callback", endpoints{wechatToken: server.URL})
	identity, err := adapter.Verify(context.Background(), login.VerificationRequest{
		ProviderCode: login.ProviderWechat, ExternalClientID: "wx1234567890ABCDEF",
		AuthorizationCode: "one-time-code", Secrets: map[string]string{"app_secret": "provider-secret"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if identity.Issuer != "wechat:wx1234567890ABCDEF" || identity.Subject != "openid-1" || identity.UnionSubject != "union-1" {
		t.Fatalf("unexpected identity: %+v", identity)
	}
}

func TestGitHubAuthorizationURLUsesStateAndS256PKCE(t *testing.T) {
	adapter := newHTTPAdapter(&http.Client{Timeout: time.Second}, "https://api.example.test/api/v1/auth/oauth/github/browser-callback", endpoints{githubAuth: "https://github.example.test/authorize"})
	descriptor, _ := login.Descriptor(login.ProviderGitHub)
	result, err := adapter.AuthorizationURL(context.Background(), descriptor, login.RuntimeProvider{ExternalClientID: "client-123"}, "state-123", "challenge-123", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"client_id=client-123", "state=state-123", "code_challenge=challenge-123", "code_challenge_method=S256"} {
		if !contains(result, expected) {
			t.Fatalf("authorization URL %q missing %q", result, expected)
		}
	}
}

func contains(value, fragment string) bool {
	for index := 0; index+len(fragment) <= len(value); index++ {
		if value[index:index+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
