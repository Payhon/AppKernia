package provider

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	login "github.com/appkernia/appkernia/server/internal/modules/loginprovider/domain"
	"github.com/golang-jwt/jwt/v5"
)

const maxProviderResponse = 1 << 20

type endpoints struct {
	wechatToken string
	githubToken string
	githubUser  string
	githubEmail string
	githubAuth  string
	appleJWKS   string
	appleToken  string
	appleRevoke string
	googleJWKS  string
}

var officialEndpoints = endpoints{
	wechatToken: "https://api.weixin.qq.com/sns/oauth2/access_token",
	githubToken: "https://github.com/login/oauth/access_token",
	githubUser:  "https://api.github.com/user",
	githubEmail: "https://api.github.com/user/emails",
	githubAuth:  "https://github.com/login/oauth/authorize",
	appleJWKS:   "https://appleid.apple.com/auth/keys",
	appleToken:  "https://appleid.apple.com/auth/token",
	appleRevoke: "https://appleid.apple.com/auth/revoke",
	googleJWKS:  "https://www.googleapis.com/oauth2/v3/certs",
}

func applePrivateKey(raw string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(raw)))
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, login.ErrCallbackInvalid
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, login.ErrCallbackInvalid
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok || key.Curve.Params().Name != "P-256" {
		return nil, login.ErrCallbackInvalid
	}
	return key, nil
}

func appleClientSecret(request login.VerificationRequest, now time.Time) (string, error) {
	config, err := login.CanonicalPublicConfig(login.ProviderApple, request.PublicConfig)
	if err != nil {
		return "", login.ErrCallbackInvalid
	}
	var public login.ApplePublicConfig
	if json.Unmarshal(config, &public) != nil {
		return "", login.ErrCallbackInvalid
	}
	key, err := applePrivateKey(request.Secrets["private_key_p8"])
	if err != nil {
		return "", err
	}
	claims := jwt.MapClaims{
		"iss": public.TeamID,
		"iat": now.Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
		"aud": "https://appleid.apple.com",
		"sub": request.ExternalClientID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = public.KeyID
	signed, err := token.SignedString(key)
	if err != nil {
		return "", login.ErrCallbackInvalid
	}
	return signed, nil
}

// RevokeApple exchanges the single-use authorization code and immediately
// revokes the returned provider token. Neither value is persisted or logged.
func (a *HTTPAdapter) RevokeApple(ctx context.Context, request login.VerificationRequest) error {
	if request.ProviderCode != login.ProviderApple || request.Mode != "reauth" ||
		strings.TrimSpace(request.AuthorizationCode) == "" || strings.TrimSpace(request.IDToken) == "" ||
		strings.TrimSpace(request.ExpectedSubject) == "" {
		return login.ErrCallbackInvalid
	}
	clientSecret, err := appleClientSecret(request, time.Now().UTC())
	if err != nil {
		return err
	}
	exchangeForm := url.Values{
		"client_id":     {request.ExternalClientID},
		"client_secret": {clientSecret},
		"code":          {strings.TrimSpace(request.AuthorizationCode)},
		"grant_type":    {"authorization_code"},
	}
	exchangeRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoints.appleToken, strings.NewReader(exchangeForm.Encode()))
	if err != nil {
		return login.ErrProviderUnavailable
	}
	exchangeRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	exchangeResponse, err := a.client.Do(exchangeRequest)
	if err != nil {
		return login.ErrProviderUnavailable
	}
	var exchange struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
	}
	if err = decodeLimited(exchangeResponse, &exchange); err != nil {
		return err
	}
	verified, err := a.verifyOIDC(ctx, login.VerificationRequest{
		ProviderCode: login.ProviderApple, ExternalClientID: request.ExternalClientID, IDToken: exchange.IDToken,
	}, "https://appleid.apple.com", a.endpoints.appleJWKS)
	if err != nil || subtle.ConstantTimeCompare([]byte(verified.Subject), []byte(request.ExpectedSubject)) != 1 {
		return login.ErrCallbackInvalid
	}
	tokenValue, hint := strings.TrimSpace(exchange.RefreshToken), "refresh_token"
	if tokenValue == "" {
		tokenValue, hint = strings.TrimSpace(exchange.AccessToken), "access_token"
	}
	if tokenValue == "" {
		return login.ErrCallbackInvalid
	}
	revokeForm := url.Values{
		"client_id":       {request.ExternalClientID},
		"client_secret":   {clientSecret},
		"token":           {tokenValue},
		"token_type_hint": {hint},
	}
	revokeRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoints.appleRevoke, strings.NewReader(revokeForm.Encode()))
	if err != nil {
		return login.ErrProviderUnavailable
	}
	revokeRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	revokeResponse, err := a.client.Do(revokeRequest)
	if err != nil {
		return login.ErrProviderUnavailable
	}
	defer revokeResponse.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(revokeResponse.Body, maxProviderResponse))
	if revokeResponse.StatusCode < 200 || revokeResponse.StatusCode >= 300 {
		return login.ErrAuthorizationDenied
	}
	return nil
}

type cachedKeySet struct {
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
}

// HTTPAdapter uses only fixed, code-reviewed provider endpoints. A private
// endpoint constructor exists for httptest coverage but is not exposed to
// configuration or Admin callers.
type HTTPAdapter struct {
	client      *http.Client
	callbackURI string
	endpoints   endpoints
	mu          sync.Mutex
	keySets     map[string]cachedKeySet
}

func NewHTTPAdapter(client *http.Client, callbackURI string) *HTTPAdapter {
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	return &HTTPAdapter{client: client, callbackURI: callbackURI, endpoints: officialEndpoints, keySets: map[string]cachedKeySet{}}
}

func newHTTPAdapter(client *http.Client, callbackURI string, target endpoints) *HTTPAdapter {
	adapter := NewHTTPAdapter(client, callbackURI)
	adapter.endpoints = target
	return adapter
}

func (a *HTTPAdapter) AuthorizationURL(_ context.Context, descriptor login.ProviderDescriptor, runtime login.RuntimeProvider, state, challenge, _ string) (string, error) {
	if descriptor.ProviderCode != login.ProviderGitHub || strings.TrimSpace(state) == "" || strings.TrimSpace(challenge) == "" || strings.TrimSpace(a.callbackURI) == "" {
		return "", login.ErrInvalid
	}
	values := url.Values{}
	values.Set("client_id", runtime.ExternalClientID)
	values.Set("redirect_uri", a.callbackURI)
	values.Set("scope", "read:user user:email")
	values.Set("state", state)
	values.Set("code_challenge", challenge)
	values.Set("code_challenge_method", "S256")
	return a.endpoints.githubAuth + "?" + values.Encode(), nil
}

func (a *HTTPAdapter) Verify(ctx context.Context, request login.VerificationRequest) (login.VerifiedIdentity, error) {
	switch request.ProviderCode {
	case login.ProviderWechat:
		return a.verifyWechat(ctx, request)
	case login.ProviderGitHub:
		return a.verifyGitHub(ctx, request)
	case login.ProviderApple:
		return a.verifyOIDC(ctx, request, "https://appleid.apple.com", a.endpoints.appleJWKS)
	case login.ProviderGoogle:
		return a.verifyOIDC(ctx, request, "https://accounts.google.com", a.endpoints.googleJWKS)
	default:
		return login.VerifiedIdentity{}, login.ErrProviderUnavailable
	}
}

func decodeLimited(response *http.Response, out any) error {
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxProviderResponse))
		return login.ErrAuthorizationDenied
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxProviderResponse))
	if err := decoder.Decode(out); err != nil {
		return login.ErrCallbackInvalid
	}
	return nil
}

func (a *HTTPAdapter) verifyWechat(ctx context.Context, request login.VerificationRequest) (login.VerifiedIdentity, error) {
	code := strings.TrimSpace(request.AuthorizationCode)
	secret := strings.TrimSpace(request.Secrets["app_secret"])
	if code == "" || secret == "" {
		return login.VerifiedIdentity{}, login.ErrCallbackInvalid
	}
	values := url.Values{}
	values.Set("appid", request.ExternalClientID)
	values.Set("secret", secret)
	values.Set("code", code)
	values.Set("grant_type", "authorization_code")
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, a.endpoints.wechatToken+"?"+values.Encode(), nil)
	if err != nil {
		return login.VerifiedIdentity{}, login.ErrProviderUnavailable
	}
	response, err := a.client.Do(httpRequest)
	if err != nil {
		return login.VerifiedIdentity{}, login.ErrProviderUnavailable
	}
	var body struct {
		OpenID  string `json:"openid"`
		UnionID string `json:"unionid"`
		ErrCode int    `json:"errcode"`
	}
	if err = decodeLimited(response, &body); err != nil {
		return login.VerifiedIdentity{}, err
	}
	if body.ErrCode != 0 || strings.TrimSpace(body.OpenID) == "" {
		return login.VerifiedIdentity{}, login.ErrAuthorizationDenied
	}
	return login.VerifiedIdentity{
		Issuer: "wechat:" + request.ExternalClientID, ExternalClientID: request.ExternalClientID,
		Subject: strings.TrimSpace(body.OpenID), UnionSubject: strings.TrimSpace(body.UnionID), Profile: json.RawMessage(`{}`),
	}, nil
}

func (a *HTTPAdapter) verifyGitHub(ctx context.Context, request login.VerificationRequest) (login.VerifiedIdentity, error) {
	code, secret := strings.TrimSpace(request.AuthorizationCode), strings.TrimSpace(request.Secrets["client_secret"])
	if code == "" || secret == "" || strings.TrimSpace(request.PKCEVerifier) == "" {
		return login.VerifiedIdentity{}, login.ErrCallbackInvalid
	}
	form := url.Values{}
	form.Set("client_id", request.ExternalClientID)
	form.Set("client_secret", secret)
	form.Set("code", code)
	form.Set("redirect_uri", request.RedirectURI)
	form.Set("code_verifier", request.PKCEVerifier)
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoints.githubToken, strings.NewReader(form.Encode()))
	if err != nil {
		return login.VerifiedIdentity{}, login.ErrProviderUnavailable
	}
	httpRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpRequest.Header.Set("Accept", "application/json")
	response, err := a.client.Do(httpRequest)
	if err != nil {
		return login.VerifiedIdentity{}, login.ErrProviderUnavailable
	}
	var exchange struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err = decodeLimited(response, &exchange); err != nil {
		return login.VerifiedIdentity{}, err
	}
	if exchange.Error != "" || strings.TrimSpace(exchange.AccessToken) == "" {
		return login.VerifiedIdentity{}, login.ErrAuthorizationDenied
	}
	var user struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}
	if err = a.githubGET(ctx, a.endpoints.githubUser, exchange.AccessToken, &user); err != nil || user.ID <= 0 {
		return login.VerifiedIdentity{}, login.ErrCallbackInvalid
	}
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	verifiedEmail := ""
	if err = a.githubGET(ctx, a.endpoints.githubEmail, exchange.AccessToken, &emails); err == nil {
		for _, item := range emails {
			if item.Verified && (verifiedEmail == "" || item.Primary) {
				verifiedEmail = strings.ToLower(strings.TrimSpace(item.Email))
				if item.Primary {
					break
				}
			}
		}
	}
	profile, _ := json.Marshal(map[string]string{"avatar_url": user.AvatarURL})
	return login.VerifiedIdentity{
		Issuer: "https://github.com", ExternalClientID: request.ExternalClientID, Subject: strconv.FormatInt(user.ID, 10),
		ProviderUsername: strings.TrimSpace(user.Login), VerifiedEmail: verifiedEmail, DisplayName: strings.TrimSpace(user.Name), Profile: profile,
	}, nil
}

func (a *HTTPAdapter) githubGET(ctx context.Context, target, token string, out any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	response, err := a.client.Do(request)
	if err != nil {
		return login.ErrProviderUnavailable
	}
	return decodeLimited(response, out)
}

type jwkSet struct {
	Keys []struct {
		KID string `json:"kid"`
		KTY string `json:"kty"`
		Use string `json:"use"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

func (a *HTTPAdapter) verifyOIDC(ctx context.Context, request login.VerificationRequest, canonicalIssuer, jwksURI string) (login.VerifiedIdentity, error) {
	rawToken := strings.TrimSpace(request.IDToken)
	if rawToken == "" {
		return login.VerifiedIdentity{}, login.ErrCallbackInvalid
	}
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(rawToken, claims, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, login.ErrCallbackInvalid
		}
		kid, _ := token.Header["kid"].(string)
		if kid == "" {
			return nil, login.ErrCallbackInvalid
		}
		return a.key(ctx, jwksURI, kid)
	}, jwt.WithAudience(request.ExternalClientID), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil || !token.Valid {
		return login.VerifiedIdentity{}, login.ErrCallbackInvalid
	}
	issuer, _ := claims["iss"].(string)
	if request.ProviderCode == login.ProviderGoogle && issuer == "accounts.google.com" {
		issuer = canonicalIssuer
	}
	if issuer != canonicalIssuer {
		return login.VerifiedIdentity{}, login.ErrCallbackInvalid
	}
	subject, _ := claims["sub"].(string)
	nonce, _ := claims["nonce"].(string)
	if strings.TrimSpace(subject) == "" || strings.TrimSpace(nonce) == "" {
		return login.VerifiedIdentity{}, login.ErrCallbackInvalid
	}
	email, _ := claims["email"].(string)
	verified := claims["email_verified"] == true || claims["email_verified"] == "true"
	if !verified {
		email = ""
	}
	name, _ := claims["name"].(string)
	return login.VerifiedIdentity{
		Issuer: canonicalIssuer, ExternalClientID: request.ExternalClientID, Subject: subject,
		VerifiedEmail: strings.ToLower(strings.TrimSpace(email)), DisplayName: strings.TrimSpace(name), Profile: json.RawMessage(`{}`), Nonce: nonce,
	}, nil
}

func (a *HTTPAdapter) key(ctx context.Context, uri, kid string) (*rsa.PublicKey, error) {
	a.mu.Lock()
	cached, exists := a.keySets[uri]
	if exists && time.Now().Before(cached.expiresAt) {
		key := cached.keys[kid]
		a.mu.Unlock()
		if key != nil {
			return key, nil
		}
		// A provider can rotate signing keys before the cache TTL expires.
		// Refresh once for an unknown kid instead of rejecting valid new tokens.
	} else {
		a.mu.Unlock()
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, login.ErrProviderUnavailable
	}
	response, err := a.client.Do(request)
	if err != nil {
		return nil, login.ErrProviderUnavailable
	}
	var set jwkSet
	if err = decodeLimited(response, &set); err != nil {
		return nil, err
	}
	keys := map[string]*rsa.PublicKey{}
	for _, item := range set.Keys {
		if item.KTY != "RSA" || item.KID == "" || item.N == "" || item.E == "" {
			continue
		}
		n, decodeErr := base64.RawURLEncoding.DecodeString(item.N)
		if decodeErr != nil {
			continue
		}
		e, decodeErr := base64.RawURLEncoding.DecodeString(item.E)
		if decodeErr != nil || len(e) == 0 || len(e) > 4 {
			continue
		}
		exponent := 0
		for _, value := range e {
			exponent = exponent<<8 + int(value)
		}
		if exponent < 3 {
			continue
		}
		keys[item.KID] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: exponent}
	}
	if len(keys) == 0 {
		return nil, login.ErrProviderUnavailable
	}
	a.mu.Lock()
	a.keySets[uri] = cachedKeySet{keys: keys, expiresAt: time.Now().Add(time.Hour)}
	a.mu.Unlock()
	key := keys[kid]
	if key == nil {
		return nil, login.ErrCallbackInvalid
	}
	return key, nil
}

func classifyTransport(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return login.ErrProviderUnavailable
	}
	return fmt.Errorf("provider transport: %w", err)
}
