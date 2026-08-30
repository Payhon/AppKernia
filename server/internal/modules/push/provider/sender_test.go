package provider

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"testing"
	"time"

	push "github.com/appkernia/appkernia/server/internal/modules/push/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestMockSenderContract(t *testing.T) {
	sender := NewMockSender()
	payload := push.SendPayload{DeliveryID: uuid.MustParse("018f08d0-3b00-7000-8000-000000000001")}
	tests := []struct {
		token string
		class string
	}{
		{"mock-accepted-token", "accepted"},
		{"mock-invalid-token", "invalid_token"},
		{"mock-throttled-token", "throttled"},
		{"mock-transient-token", "transient"},
		{"mock-unknown-token", "unknown_after_write"},
	}
	for _, test := range tests {
		t.Run(test.class, func(t *testing.T) {
			result := sender.Send(context.Background(), uuid.Nil, uuid.Nil, push.ProviderFCM, test.token, payload)
			if result.Class != test.class {
				t.Fatalf("class=%q want %q", result.Class, test.class)
			}
			if test.class == "accepted" && result.ProviderMessageID == "" {
				t.Fatal("accepted result must include a provider message id")
			}
		})
	}
}

func TestHuaweiServiceAccountJWTUsesPS256(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: mustPKCS8(t, key)})
	raw, err := json.Marshal(map[string]string{
		"project_id": "project-1", "key_id": "kid-1", "sub_account": "service-account-1",
		"private_key": string(keyPEM), "token_uri": "https://oauth-login.cloud.huawei.com/oauth2/v3/token",
	})
	if err != nil {
		t.Fatal(err)
	}
	sender := &OfficialSender{clock: func() time.Time { return time.Unix(1_800_000_000, 0).UTC() }, tokens: map[string]cachedToken{}}
	signed, err := sender.huaweiServiceAccountJWT(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := jwt.Parse(signed, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodPS256.Alg() {
			t.Fatalf("algorithm=%s want PS256", token.Method.Alg())
		}
		if token.Header["kid"] != "kid-1" {
			t.Fatalf("kid=%v", token.Header["kid"])
		}
		return &key.PublicKey, nil
	}, jwt.WithAudience("https://oauth-login.cloud.huawei.com/oauth2/v3/token"), jwt.WithIssuer("service-account-1"), jwt.WithTimeFunc(sender.clock))
	if err != nil || !parsed.Valid {
		t.Fatalf("parse signed Huawei JWT: valid=%v err=%v", parsed.Valid, err)
	}
}

func mustPKCS8(t *testing.T, key *rsa.PrivateKey) []byte {
	t.Helper()
	value, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestHTTPClassification(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		header http.Header
		class  string
	}{
		{"throttled", http.StatusTooManyRequests, `{}`, http.Header{"Retry-After": {"17"}}, "throttled"},
		{"transient", http.StatusServiceUnavailable, `{}`, http.Header{}, "transient"},
		{"credentials", http.StatusUnauthorized, `{}`, http.Header{}, "auth_config_error"},
		{"invalid-token", http.StatusBadRequest, `{"error":{"status":"UNREGISTERED"}}`, http.Header{}, "invalid_token"},
		{"invalid-payload-not-token", http.StatusBadRequest, `{"error":{"status":"INVALID_ARGUMENT","message":"invalid ttl"}}`, http.Header{}, "permanent"},
		{"missing-route-not-token", http.StatusNotFound, `{"error":"project not found"}`, http.Header{}, "permanent"},
		{"permanent", http.StatusBadRequest, `{"error":"bad payload"}`, http.Header{}, "permanent"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := classifyHTTP("FCM", test.status, []byte(test.body), test.header)
			if result.Class != test.class {
				t.Fatalf("class=%q want %q", result.Class, test.class)
			}
			if test.name == "throttled" && result.RetryAfter != 17*time.Second {
				t.Fatalf("retry=%s", result.RetryAfter)
			}
			if result.SafeSummary == "" || result.ErrorCode == "" {
				t.Fatal("failure must expose a safe class summary and stable code")
			}
		})
	}
}

func TestTransportFailureDoesNotReplayUnknownWrite(t *testing.T) {
	before := transportFailure("APNS", httpResult{wrote: false, err: context.DeadlineExceeded})
	after := transportFailure("APNS", httpResult{wrote: true, err: context.DeadlineExceeded})
	if before.Class != "transient" || after.Class != "unknown_after_write" || after.RetryAfter != 0 {
		t.Fatalf("unexpected transport classes: before=%+v after=%+v", before, after)
	}
}

func TestProviderBodyAcceptanceFixtures(t *testing.T) {
	fixtures := map[string]string{
		"HUAWEI":  `{"code":"80000000","requestId":"h"}`,
		"HARMONY": `{"code":"80000000","requestId":"a"}`,
		"HONOR":   `{"code":"12200000","requestId":"o"}`,
		"XIAOMI":  `{"result":"ok","code":"0"}`,
		"OPPO":    `{"code":0,"messageId":"p"}`,
		"VIVO":    `{"result":0,"requestId":"v"}`,
		"MEIZU":   `{"code":"200","msgId":"m"}`,
	}
	for provider, body := range fixtures {
		if !providerBodyAccepted(provider, []byte(body)) {
			t.Errorf("%s fixture should be accepted", provider)
		}
	}
	if providerBodyAccepted("XIAOMI", []byte(`{"code":"invalid"}`)) {
		t.Fatal("rejected provider body must not be accepted")
	}
	if providerBodyAccepted("XIAOMI", []byte(`{"error":"bad request"}`)) {
		t.Fatal("ambiguous provider body must not be accepted")
	}
}

func TestAPNSConnectionsAreIsolatedByCredentialTopicAndEnvironment(t *testing.T) {
	sender := NewOfficialSender(nil, nil, "development")
	first := providerConfig{public: map[string]string{
		"team_id": "team-1", "key_id": "key-1", "bundle_id": "com.example.first", "apns_environment": "sandbox",
	}}
	second := providerConfig{public: map[string]string{
		"team_id": "team-1", "key_id": "key-1", "bundle_id": "com.example.second", "apns_environment": "sandbox",
	}}
	if sender.apnsClient(first) != sender.apnsClient(first) {
		t.Fatal("the same APNs configuration must reuse its HTTP/2 connection pool")
	}
	if sender.apnsClient(first) == sender.apnsClient(second) {
		t.Fatal("different APNs topics must not share a connection pool")
	}
}

func TestProviderCategoryTracksMessageCategory(t *testing.T) {
	config := map[string]string{"service_category": "ACCOUNT", "operations_category": "MARKETING"}
	if got := categoryForPayload(config, push.CategoryServiceSecurity); got != "ACCOUNT" {
		t.Fatalf("service category=%q", got)
	}
	if got := categoryForPayload(config, push.CategoryNewsOperations); got != "MARKETING" {
		t.Fatalf("operations category=%q", got)
	}
}

func TestOfficialProviderEndpointsMatchCurrentVendorDocumentation(t *testing.T) {
	if honorAuthEndpoint != "https://iam.developer.honor.com/auth/token" {
		t.Fatalf("HONOR auth endpoint=%q", honorAuthEndpoint)
	}
	wantXiaomiHosts := map[string]string{
		"china":     "api.xmpush.xiaomi.com",
		"singapore": "sgp-api.xmpush.global.xiaomi.com",
		"europe":    "fr-api.xmpush.global.xiaomi.com",
		"india":     "idmb-api.xmpush.global.xiaomi.com",
		"russia":    "ru-api.xmpush.global.xiaomi.com",
	}
	for region, want := range wantXiaomiHosts {
		if got := xiaomiAPIHosts[region]; got != want {
			t.Errorf("Xiaomi host for %s=%q want %q", region, got, want)
		}
	}
}
