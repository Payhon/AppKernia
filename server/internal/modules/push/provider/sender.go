package provider

import (
	"context"
	"crypto/ecdsa"
	"crypto/md5"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	push "github.com/appkernia/appkernia/server/internal/modules/push/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SecretOpener interface {
	Open([]byte, string) ([]byte, error)
}

type MockSender struct{}

func NewMockSender() *MockSender { return &MockSender{} }

func (*MockSender) Send(_ context.Context, _, _ uuid.UUID, _ string, token string, payload push.SendPayload) push.SendResult {
	switch {
	case strings.HasPrefix(token, "mock-invalid-"):
		return failed("invalid_token", "MOCK.INVALID_TOKEN", "mock provider rejected an invalid token", 0)
	case strings.HasPrefix(token, "mock-throttled-"):
		return failed("throttled", "MOCK.THROTTLED", "mock provider throttled the request", 30*time.Second)
	case strings.HasPrefix(token, "mock-transient-"):
		return failed("transient", "MOCK.TRANSIENT", "mock provider is temporarily unavailable", 15*time.Second)
	case strings.HasPrefix(token, "mock-unknown-"):
		return failed("unknown_after_write", "MOCK.UNKNOWN_AFTER_WRITE", "mock provider outcome is unknown after request write", 0)
	default:
		return push.SendResult{ProviderMessageID: "mock-" + payload.DeliveryID.String(), Class: "accepted"}
	}
}

func (*MockSender) Preflight(context.Context, uuid.UUID, uuid.UUID, string) []string { return nil }

type OfficialSender struct {
	pool        *pgxpool.Pool
	opener      SecretOpener
	environment string
	http        *http.Client
	clock       func() time.Time
	mu          sync.Mutex
	tokens      map[string]cachedToken
	apnsClients map[string]*http.Client
}

type cachedToken struct {
	value     string
	expiresAt time.Time
}

const honorAuthEndpoint = "https://iam.developer.honor.com/auth/token"

var xiaomiAPIHosts = map[string]string{
	"china":     "api.xmpush.xiaomi.com",
	"singapore": "sgp-api.xmpush.global.xiaomi.com",
	"europe":    "fr-api.xmpush.global.xiaomi.com",
	"india":     "idmb-api.xmpush.global.xiaomi.com",
	"russia":    "ru-api.xmpush.global.xiaomi.com",
}

func NewOfficialSender(pool *pgxpool.Pool, opener SecretOpener, environment string) *OfficialSender {
	return &OfficialSender{
		pool: pool, opener: opener, environment: normalizeEnvironment(environment),
		http: &http.Client{Timeout: 30 * time.Second}, clock: time.Now,
		tokens: map[string]cachedToken{}, apnsClients: map[string]*http.Client{},
	}
}

type providerConfig struct {
	public map[string]string
	secret map[string]string
}

func (s *OfficialSender) load(ctx context.Context, tenantID, appID uuid.UUID, provider string, requireActive bool) (providerConfig, error) {
	query := `SELECT public_config,secret_ciphertext FROM notify.push_provider_configs
WHERE tenant_id=$1 AND app_id=$2 AND provider=$3 AND environment=$4`
	if requireActive {
		query += " AND status='active'"
	}
	var publicJSON, ciphertext []byte
	err := s.pool.QueryRow(ctx, query, tenantID, appID, provider, s.environment).Scan(&publicJSON, &ciphertext)
	if errors.Is(err, pgx.ErrNoRows) {
		return providerConfig{}, errors.New("push provider configuration is unavailable")
	}
	if err != nil {
		return providerConfig{}, err
	}
	plaintext, err := s.opener.Open(ciphertext, "push-config:"+appID.String()+":"+provider+":"+s.environment)
	if err != nil {
		return providerConfig{}, errors.New("push provider credential cannot be opened")
	}
	out := providerConfig{public: map[string]string{}, secret: map[string]string{}}
	if json.Unmarshal(publicJSON, &out.public) != nil || json.Unmarshal(plaintext, &out.secret) != nil {
		return providerConfig{}, errors.New("push provider configuration is invalid")
	}
	return out, nil
}

func (s *OfficialSender) Send(ctx context.Context, tenantID, appID uuid.UUID, provider, token string, payload push.SendPayload) push.SendResult {
	config, err := s.load(ctx, tenantID, appID, provider, true)
	if err != nil {
		return failed("auth_config_error", "PUSH.CONFIG.UNAVAILABLE", err.Error(), 0)
	}
	switch provider {
	case push.ProviderAPNS:
		return s.sendAPNS(ctx, config, token, payload)
	case push.ProviderFCM:
		return s.sendFCM(ctx, config, token, payload)
	case push.ProviderHuaweiAndroid:
		return s.sendHuawei(ctx, config, token, payload)
	case push.ProviderHonor:
		return s.sendHonor(ctx, config, token, payload)
	case push.ProviderXiaomi:
		return s.sendXiaomi(ctx, config, token, payload)
	case push.ProviderOPPO:
		return s.sendOPPO(ctx, config, token, payload)
	case push.ProviderVivo:
		return s.sendVivo(ctx, config, token, payload)
	case push.ProviderMeizu:
		return s.sendMeizu(ctx, config, token, payload)
	case push.ProviderHarmony:
		return s.sendHarmony(ctx, config, token, payload)
	default:
		return failed("permanent", "PUSH.PROVIDER.UNSUPPORTED", "push provider is not compiled into this server", 0)
	}
}

func (s *OfficialSender) Preflight(ctx context.Context, tenantID, appID uuid.UUID, provider string) []string {
	config, err := s.load(ctx, tenantID, appID, provider, false)
	if err != nil {
		return []string{"push.preflight.credential_unreadable"}
	}
	switch provider {
	case push.ProviderAPNS:
		if _, err = apnsSigningKey(config.secret["private_key_p8"]); err != nil {
			return []string{"push.preflight.private_key_invalid"}
		}
	case push.ProviderFCM:
		if _, err = parseServiceAccount(config.secret["service_account_json"]); err != nil {
			return []string{"push.preflight.service_account_invalid"}
		}
	case push.ProviderHarmony:
		if _, err = parseHuaweiServiceAccount(config.secret["service_account_json"]); err != nil {
			return []string{"push.preflight.service_account_invalid"}
		}
	case push.ProviderHuaweiAndroid, push.ProviderHonor, push.ProviderOPPO:
		if strings.TrimSpace(config.secret[firstSecret(provider)]) == "" {
			return []string{"push.preflight.secret_missing"}
		}
	}
	return nil
}

func firstSecret(provider string) string {
	switch provider {
	case push.ProviderOPPO:
		return "master_secret"
	default:
		return "client_secret"
	}
}

func (s *OfficialSender) sendAPNS(ctx context.Context, config providerConfig, deviceToken string, payload push.SendPayload) push.SendResult {
	authorizationCacheKey := "apns:" + config.public["team_id"] + ":" + config.public["key_id"]
	signed := s.cached(authorizationCacheKey)
	if signed == "" {
		key, err := apnsSigningKey(config.secret["private_key_p8"])
		if err != nil {
			return failed("auth_config_error", "APNS.PRIVATE_KEY_INVALID", "APNs private key is invalid", 0)
		}
		claims := jwt.MapClaims{"iss": config.public["team_id"], "iat": s.clock().Unix()}
		token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
		token.Header["kid"] = config.public["key_id"]
		signed, err = token.SignedString(key)
		if err != nil {
			return failed("auth_config_error", "APNS.TOKEN_SIGN_FAILED", "APNs authorization token cannot be signed", 0)
		}
		s.cache(authorizationCacheKey, signed, 50*time.Minute)
	}
	aps := map[string]any{"alert": map[string]string{"title": payload.Title, "body": payload.Body}, "sound": "default"}
	if payload.CollapseKey != "" {
		aps["thread-id"] = payload.CollapseKey
	}
	body, _ := json.Marshal(map[string]any{"aps": aps, "schema_version": payload.SchemaVersion, "delivery_id": payload.DeliveryID, "message_id": payload.MessageID, "route_key": payload.RouteKey, "route_params": payload.RouteParams})
	host := "https://api.push.apple.com"
	if config.public["apns_environment"] == "sandbox" {
		host = "https://api.sandbox.push.apple.com"
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, host+"/3/device/"+url.PathEscape(deviceToken), strings.NewReader(string(body)))
	req.Header.Set("authorization", "bearer "+signed)
	req.Header.Set("apns-topic", config.public["bundle_id"])
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("apns-priority", "10")
	req.Header.Set("apns-expiration", strconv.FormatInt(s.clock().Add(time.Duration(payload.TTLSeconds)*time.Second).Unix(), 10))
	if payload.CollapseKey != "" {
		req.Header.Set("apns-collapse-id", payload.CollapseKey)
	}
	response := s.executeWith(s.apnsClient(config), req)
	if response.err != nil {
		return transportFailure("APNS", response)
	}
	if response.status == http.StatusOK {
		return push.SendResult{ProviderMessageID: response.header.Get("apns-id"), Class: "accepted"}
	}
	return classifyHTTP("APNS", response.status, response.body, response.header)
}

type serviceAccount struct {
	ProjectID   string `json:"project_id"`
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
	KeyID       string `json:"private_key_id"`
	key         *rsa.PrivateKey
}

type huaweiServiceAccount struct {
	ProjectID  string `json:"project_id"`
	KeyID      string `json:"key_id"`
	SubAccount string `json:"sub_account"`
	PrivateKey string `json:"private_key"`
	TokenURI   string `json:"token_uri"`
	key        *rsa.PrivateKey
}

func parseServiceAccount(raw string) (serviceAccount, error) {
	var account serviceAccount
	if json.Unmarshal([]byte(raw), &account) != nil || account.ClientEmail == "" || account.PrivateKey == "" {
		return account, errors.New("service account JSON is invalid")
	}
	block, _ := pem.Decode([]byte(account.PrivateKey))
	if block == nil {
		return account, errors.New("service account key is invalid")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return account, err
	}
	var ok bool
	account.key, ok = parsed.(*rsa.PrivateKey)
	if !ok {
		return account, errors.New("service account key is not RSA")
	}
	if account.TokenURI == "" {
		account.TokenURI = "https://oauth2.googleapis.com/token"
	}
	return account, nil
}

func parseHuaweiServiceAccount(raw string) (huaweiServiceAccount, error) {
	var account huaweiServiceAccount
	if json.Unmarshal([]byte(raw), &account) != nil || account.KeyID == "" || account.SubAccount == "" || account.PrivateKey == "" {
		return account, errors.New("Huawei service account JSON is invalid")
	}
	keyBytes := []byte(strings.ReplaceAll(account.PrivateKey, `\n`, "\n"))
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		decoded, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(account.PrivateKey), ""))
		if err != nil {
			return account, errors.New("Huawei service account key is invalid")
		}
		block = &pem.Block{Type: "PRIVATE KEY", Bytes: decoded}
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return account, err
	}
	var ok bool
	account.key, ok = parsed.(*rsa.PrivateKey)
	if !ok {
		return account, errors.New("Huawei service account key is not RSA")
	}
	if account.TokenURI == "" {
		account.TokenURI = "https://oauth-login.cloud.huawei.com/oauth2/v3/token"
	}
	return account, nil
}

func (s *OfficialSender) huaweiServiceAccountJWT(raw string) (string, error) {
	account, err := parseHuaweiServiceAccount(raw)
	if err != nil {
		return "", err
	}
	cacheKey := "harmony|" + account.SubAccount + "|" + account.KeyID
	if token := s.cached(cacheKey); token != "" {
		return token, nil
	}
	now := s.clock().UTC()
	claims := jwt.MapClaims{"iss": account.SubAccount, "aud": account.TokenURI, "iat": now.Unix(), "exp": now.Add(time.Hour).Unix()}
	token := jwt.NewWithClaims(jwt.SigningMethodPS256, claims)
	token.Header["kid"] = account.KeyID
	token.Header["typ"] = "JWT"
	signed, err := token.SignedString(account.key)
	if err != nil {
		return "", err
	}
	s.cache(cacheKey, signed, time.Hour)
	return signed, nil
}

func (s *OfficialSender) serviceAccountToken(ctx context.Context, raw, scope string) (string, error) {
	account, err := parseServiceAccount(raw)
	if err != nil {
		return "", err
	}
	cacheKey := account.ClientEmail + "|" + scope
	if token := s.cached(cacheKey); token != "" {
		return token, nil
	}
	now := s.clock().UTC()
	claims := jwt.MapClaims{"iss": account.ClientEmail, "scope": scope, "aud": account.TokenURI, "iat": now.Unix(), "exp": now.Add(time.Hour).Unix()}
	assertion := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	if account.KeyID != "" {
		assertion.Header["kid"] = account.KeyID
	}
	signed, err := assertion.SignedString(account.key)
	if err != nil {
		return "", err
	}
	form := url.Values{"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"}, "assertion": {signed}}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, account.TokenURI, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := s.execute(req)
	if response.err != nil || response.status/100 != 2 {
		return "", errors.New("service account authorization failed")
	}
	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if json.Unmarshal(response.body, &result) != nil || result.AccessToken == "" {
		return "", errors.New("service account authorization response is invalid")
	}
	s.cache(cacheKey, result.AccessToken, time.Duration(result.ExpiresIn)*time.Second)
	return result.AccessToken, nil
}

func (s *OfficialSender) sendFCM(ctx context.Context, config providerConfig, deviceToken string, payload push.SendPayload) push.SendResult {
	accessToken, err := s.serviceAccountToken(ctx, config.secret["service_account_json"], "https://www.googleapis.com/auth/firebase.messaging")
	if err != nil {
		return failed("auth_config_error", "FCM.AUTH_FAILED", "FCM service account authorization failed", 0)
	}
	data := payloadData(payload)
	body, _ := json.Marshal(map[string]any{"message": map[string]any{"token": deviceToken, "notification": map[string]string{"title": payload.Title, "body": payload.Body}, "data": data,
		"android": map[string]any{"ttl": fmt.Sprintf("%ds", payload.TTLSeconds), "collapse_key": payload.CollapseKey, "priority": "HIGH"}}})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://fcm.googleapis.com/v1/projects/"+url.PathEscape(config.public["project_id"])+"/messages:send", strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	response := s.execute(req)
	if response.err != nil {
		return transportFailure("FCM", response)
	}
	if response.status/100 == 2 {
		var result struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(response.body, &result)
		return push.SendResult{ProviderMessageID: result.Name, Class: "accepted"}
	}
	return classifyHTTP("FCM", response.status, response.body, response.header)
}

func payloadData(payload push.SendPayload) map[string]string {
	params, _ := json.Marshal(payload.RouteParams)
	return map[string]string{"schema_version": strconv.Itoa(payload.SchemaVersion), "delivery_id": payload.DeliveryID.String(), "message_id": payload.MessageID.String(), "route_key": payload.RouteKey, "route_params": string(params)}
}

func (s *OfficialSender) clientCredentialsToken(ctx context.Context, cacheKey, endpoint string, values url.Values) (string, error) {
	if token := s.cached(cacheKey); token != "" {
		return token, nil
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := s.execute(req)
	if response.err != nil || response.status/100 != 2 {
		return "", errors.New("provider authorization failed")
	}
	var raw map[string]any
	if json.Unmarshal(response.body, &raw) != nil {
		return "", errors.New("provider authorization response is invalid")
	}
	token := stringValue(raw, "access_token", "accessToken", "auth_token")
	if token == "" {
		if data, ok := raw["data"].(map[string]any); ok {
			token = stringValue(data, "access_token", "accessToken", "auth_token")
		}
	}
	if token == "" {
		return "", errors.New("provider authorization token is missing")
	}
	s.cache(cacheKey, token, 55*time.Minute)
	return token, nil
}

func (s *OfficialSender) sendHuawei(ctx context.Context, config providerConfig, deviceToken string, payload push.SendPayload) push.SendResult {
	form := url.Values{"grant_type": {"client_credentials"}, "client_id": {config.public["client_id"]}, "client_secret": {config.secret["client_secret"]}}
	accessToken, err := s.clientCredentialsToken(ctx, "huawei|"+config.public["client_id"], "https://oauth-login.cloud.huawei.com/oauth2/v3/token", form)
	if err != nil {
		return failed("auth_config_error", "HUAWEI.AUTH_FAILED", "Huawei Push Kit authorization failed", 0)
	}
	body, _ := json.Marshal(map[string]any{"validate_only": false, "message": map[string]any{"token": []string{deviceToken}, "notification": map[string]string{"title": payload.Title, "body": payload.Body}, "data": mustJSONString(payloadData(payload)), "android": map[string]any{"collapse_key": payload.CollapseKey, "ttl": fmt.Sprintf("%ds", payload.TTLSeconds), "notification": map[string]string{"channel_id": config.public["notification_channel_id"]}}}})
	endpoint := "https://push-api.cloud.huawei.com/v1/" + url.PathEscape(config.public["client_id"]) + "/messages:send"
	return s.sendBearerJSON(ctx, "HUAWEI", endpoint, accessToken, body)
}

func (s *OfficialSender) sendHarmony(ctx context.Context, config providerConfig, deviceToken string, payload push.SendPayload) push.SendResult {
	authorization, err := s.huaweiServiceAccountJWT(config.secret["service_account_json"])
	if err != nil {
		return failed("auth_config_error", "HARMONY.AUTH_FAILED", "HarmonyOS Push Kit service account authorization failed", 0)
	}
	extraData := mustJSONString(payloadData(payload))
	category := categoryForPayload(config.public, payload.Category)
	body, _ := json.Marshal(map[string]any{
		"payload": map[string]any{"extraData": extraData, "notification": map[string]any{
			"category": category, "title": payload.Title, "body": payload.Body,
			"clickAction": map[string]int{"actionType": 0}, "foregroundShow": false,
		}},
		"target":      map[string]any{"token": []string{deviceToken}},
		"pushOptions": map[string]any{"ttl": payload.TTLSeconds},
	})
	endpoint := "https://push-api.cloud.huawei.com/v3/" + url.PathEscape(config.public["project_id"]) + "/messages:send"
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+authorization)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("push-type", "0")
	return s.executeVendor(req, "HARMONY")
}

func (s *OfficialSender) sendHonor(ctx context.Context, config providerConfig, deviceToken string, payload push.SendPayload) push.SendResult {
	form := url.Values{"grant_type": {"client_credentials"}, "client_id": {config.public["client_id"]}, "client_secret": {config.secret["client_secret"]}}
	accessToken, err := s.clientCredentialsToken(ctx, "honor|"+config.public["client_id"], honorAuthEndpoint, form)
	if err != nil {
		return failed("auth_config_error", "HONOR.AUTH_FAILED", "Honor Push authorization failed", 0)
	}
	body, _ := json.Marshal(map[string]any{"token": []string{deviceToken}, "notification": map[string]string{"title": payload.Title, "body": payload.Body}, "data": mustJSONString(payloadData(payload)), "android": map[string]any{"ttl": payload.TTLSeconds, "notification": map[string]string{"channel_id": config.public["notification_channel_id"]}}})
	return s.sendBearerJSON(ctx, "HONOR", "https://push-api.cloud.honor.com/api/v1/"+url.PathEscape(config.public["app_id"])+"/sendMessage", accessToken, body)
}

func (s *OfficialSender) sendBearerJSON(ctx context.Context, provider, endpoint, token string, body []byte) push.SendResult {
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	response := s.execute(req)
	if response.err != nil {
		return transportFailure(provider, response)
	}
	if response.status/100 == 2 && providerBodyAccepted(provider, response.body) {
		return push.SendResult{ProviderMessageID: responseMessageID(response.body), Class: "accepted"}
	}
	return classifyHTTP(provider, response.status, response.body, response.header)
}

func (s *OfficialSender) sendXiaomi(ctx context.Context, config providerConfig, deviceToken string, payload push.SendPayload) push.SendResult {
	host := xiaomiAPIHosts[config.public["region"]]
	form := url.Values{"registration_id": {deviceToken}, "restricted_package_name": {config.public["package_name"]}, "title": {payload.Title}, "description": {payload.Body}, "payload": {mustJSONString(payloadData(payload))}, "pass_through": {"0"}, "notify_type": {"-1"}, "time_to_live": {strconv.Itoa(payload.TTLSeconds * 1000)}, "extra.channel_id": {config.public["notification_channel_id"]}}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+host+"/v3/message/regid", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "key="+config.secret["app_secret"])
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return s.executeVendor(req, "XIAOMI")
}

func (s *OfficialSender) sendOPPO(ctx context.Context, config providerConfig, deviceToken string, payload push.SendPayload) push.SendResult {
	timestamp := strconv.FormatInt(s.clock().UnixMilli(), 10)
	signature := sha256.Sum256([]byte(config.public["app_key"] + timestamp + config.secret["master_secret"]))
	auth := url.Values{"app_key": {config.public["app_key"]}, "sign": {hex.EncodeToString(signature[:])}, "timestamp": {timestamp}}
	accessToken, err := s.clientCredentialsToken(ctx, "oppo|"+config.public["app_key"], "https://api.push.oppomobile.com/server/v1/auth", auth)
	if err != nil {
		return failed("auth_config_error", "OPPO.AUTH_FAILED", "OPPO Push authorization failed", 0)
	}
	message, _ := json.Marshal(map[string]any{"title": payload.Title, "content": payload.Body, "click_action_type": 4, "click_action_activity": "", "action_parameters": payloadData(payload), "channel_id": config.public["notification_channel_id"], "off_line_ttl": payload.TTLSeconds})
	form := url.Values{"auth_token": {accessToken}, "registration_id": {deviceToken}, "message": {string(message)}}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.push.oppomobile.com/server/v1/message/notification/unicast", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return s.executeVendor(req, "OPPO")
}

func (s *OfficialSender) sendVivo(ctx context.Context, config providerConfig, deviceToken string, payload push.SendPayload) push.SendResult {
	timestamp := strconv.FormatInt(s.clock().UnixMilli(), 10)
	digest := md5.Sum([]byte(config.public["app_id"] + config.public["app_key"] + timestamp + config.secret["app_secret"]))
	authBody, _ := json.Marshal(map[string]string{"appId": config.public["app_id"], "appKey": config.public["app_key"], "timestamp": timestamp, "sign": hex.EncodeToString(digest[:])})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api-push.vivo.com.cn/message/auth", strings.NewReader(string(authBody)))
	req.Header.Set("Content-Type", "application/json")
	response := s.execute(req)
	if response.err != nil || response.status/100 != 2 {
		return failed("auth_config_error", "VIVO.AUTH_FAILED", "vivo Push authorization failed", 0)
	}
	var auth map[string]any
	_ = json.Unmarshal(response.body, &auth)
	authToken := stringValue(auth, "authToken")
	if authToken == "" {
		return failed("auth_config_error", "VIVO.AUTH_FAILED", "vivo Push authorization response is invalid", 0)
	}
	classification := 1
	if payload.Category == push.CategoryNewsOperations {
		classification = 0
	}
	body, _ := json.Marshal(map[string]any{"regId": deviceToken, "notifyType": 4, "title": payload.Title, "content": payload.Body, "timeToLive": payload.TTLSeconds, "skipType": 1, "networkType": -1, "classification": classification, "category": categoryForPayload(config.public, payload.Category), "extra": payloadData(payload)})
	req, _ = http.NewRequestWithContext(ctx, http.MethodPost, "https://api-push.vivo.com.cn/message/send", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("authToken", authToken)
	return s.executeVendor(req, "VIVO")
}

func categoryForPayload(config map[string]string, category string) string {
	if category == push.CategoryNewsOperations {
		return config["operations_category"]
	}
	return config["service_category"]
}

func (s *OfficialSender) sendMeizu(ctx context.Context, config providerConfig, deviceToken string, payload push.SendPayload) push.SendResult {
	jsonPayload, _ := json.Marshal(map[string]any{"noticeBarInfo": map[string]any{"title": payload.Title, "content": payload.Body}, "noticeExpandInfo": map[string]any{"noticeExpandType": 0}, "clickTypeInfo": map[string]any{"clickType": 3, "parameters": payloadData(payload)}, "timeValid": payload.TTLSeconds / 3600})
	form := url.Values{"appId": {config.public["app_id"]}, "pushIds": {deviceToken}, "messageJson": {string(jsonPayload)}}
	keys := make([]string, 0, len(form))
	for key := range form {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var signature strings.Builder
	for _, key := range keys {
		signature.WriteString(key)
		signature.WriteString("=")
		signature.WriteString(form.Get(key))
	}
	signature.WriteString(config.secret["app_secret"])
	digest := md5.Sum([]byte(signature.String()))
	form.Set("sign", hex.EncodeToString(digest[:]))
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://server-api-push.meizu.com/garcia/api/server/push/varnished/pushByPushId", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return s.executeVendor(req, "MEIZU")
}

func (s *OfficialSender) executeVendor(req *http.Request, provider string) push.SendResult {
	response := s.execute(req)
	if response.err != nil {
		return transportFailure(provider, response)
	}
	if response.status/100 == 2 && providerBodyAccepted(provider, response.body) {
		return push.SendResult{ProviderMessageID: responseMessageID(response.body), Class: "accepted"}
	}
	return classifyHTTP(provider, response.status, response.body, response.header)
}

type httpResult struct {
	status int
	body   []byte
	header http.Header
	wrote  bool
	err    error
}

func (s *OfficialSender) execute(req *http.Request) httpResult {
	return s.executeWith(s.http, req)
}

func (s *OfficialSender) executeWith(client *http.Client, req *http.Request) httpResult {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	wrote := false
	trace := &httptrace.ClientTrace{WroteRequest: func(httptrace.WroteRequestInfo) { wrote = true }}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	response, err := client.Do(req)
	if err != nil {
		return httpResult{wrote: wrote, err: err}
	}
	defer func() { _ = response.Body.Close() }()
	body, readErr := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	return httpResult{status: response.StatusCode, body: body, header: response.Header.Clone(), wrote: wrote, err: readErr}
}

func (s *OfficialSender) apnsClient(config providerConfig) *http.Client {
	connectionKey := strings.Join([]string{
		config.public["team_id"], config.public["key_id"],
		config.public["bundle_id"], config.public["apns_environment"],
	}, "\x00")
	s.mu.Lock()
	defer s.mu.Unlock()
	if client := s.apnsClients[connectionKey]; client != nil {
		return client
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ForceAttemptHTTP2 = true
	client := &http.Client{Timeout: 30 * time.Second, Transport: transport}
	if s.apnsClients == nil {
		s.apnsClients = map[string]*http.Client{}
	}
	s.apnsClients[connectionKey] = client
	return client
}

func transportFailure(provider string, response httpResult) push.SendResult {
	if response.wrote {
		return failed("unknown_after_write", provider+".UNKNOWN_AFTER_WRITE", "provider outcome is unknown after request write", 0)
	}
	return failed("transient", provider+".CONNECTION_FAILED", "provider connection failed before request write", 15*time.Second)
}

func classifyHTTP(provider string, status int, body []byte, header http.Header) push.SendResult {
	upper := strings.ToUpper(string(body))
	code := provider + ".HTTP_" + strconv.Itoa(status)
	if status == http.StatusTooManyRequests {
		return failed("throttled", code, "provider throttled the request", retryAfter(header))
	}
	if status >= 500 {
		return failed("transient", code, "provider is temporarily unavailable", retryAfter(header))
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return failed("auth_config_error", code, "provider rejected the configured credentials", 0)
	}
	if invalidTokenResponse(provider, status, upper) {
		return failed("invalid_token", code, "provider rejected the device registration token", 0)
	}
	return failed("permanent", code, "provider permanently rejected the notification", 0)
}

func invalidTokenResponse(provider string, status int, upperBody string) bool {
	if provider == "APNS" && (status == http.StatusGone || strings.Contains(upperBody, "BADDEVICETOKEN") || strings.Contains(upperBody, "DEVICETOKENNOTFORTOPIC") || strings.Contains(upperBody, "UNREGISTERED")) {
		return true
	}
	if provider == "FCM" && (strings.Contains(upperBody, "UNREGISTERED") || strings.Contains(upperBody, "SENDER_ID_MISMATCH") || strings.Contains(upperBody, "REGISTRATION TOKEN IS NOT A VALID FCM REGISTRATION TOKEN")) {
		return true
	}
	for _, marker := range []string{"INVALID REGID", "INVALID_TOKEN", "INVALID TOKEN", "TOKEN_INVALID", "TOKEN INVALID", "DEVICE_NOT_REGISTERED", "NOTREGISTERED"} {
		if strings.Contains(upperBody, marker) {
			return true
		}
	}
	return false
}

func providerBodyAccepted(provider string, body []byte) bool {
	if len(body) == 0 {
		return true
	}
	var raw map[string]any
	if json.Unmarshal(body, &raw) != nil {
		return false
	}
	code := strings.ToLower(stringValue(raw, "code", "result", "ret"))
	if code == "" {
		return false
	}
	accepted := map[string][]string{"HUAWEI": {"80000000", "0", "success"}, "HARMONY": {"80000000", "0", "success"}, "HONOR": {"12200000", "0", "success"}, "XIAOMI": {"0", "success"}, "OPPO": {"0", "success"}, "VIVO": {"0", "success"}, "MEIZU": {"200", "0", "success"}}
	for _, candidate := range accepted[provider] {
		if code == candidate {
			return true
		}
	}
	return false
}

func responseMessageID(body []byte) string {
	var raw map[string]any
	if json.Unmarshal(body, &raw) != nil {
		return ""
	}
	return stringValue(raw, "requestId", "request_id", "messageId", "message_id", "msgId", "id")
}

func stringValue(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := raw[key]; ok {
			switch typed := value.(type) {
			case string:
				return typed
			case float64:
				return strconv.FormatInt(int64(typed), 10)
			}
		}
	}
	return ""
}

func retryAfter(header http.Header) time.Duration {
	value := strings.TrimSpace(header.Get("Retry-After"))
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if at, err := http.ParseTime(value); err == nil {
		if delay := time.Until(at); delay > 0 {
			return delay
		}
	}
	return 30 * time.Second
}

func failed(class, code, summary string, retry time.Duration) push.SendResult {
	return push.SendResult{Class: class, ErrorCode: code, SafeSummary: summary, RetryAfter: retry}
}

func apnsSigningKey(raw string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, errors.New("APNs key is not PEM")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("APNs key is not EC")
	}
	return key, nil
}

func mustJSONString(value any) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

func normalizeEnvironment(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "development", "test", "staging", "production":
		return value
	default:
		return "development"
	}
}

func (s *OfficialSender) cached(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.tokens[key]
	if item.value != "" && s.clock().Before(item.expiresAt.Add(-time.Minute)) {
		return item.value
	}
	delete(s.tokens, key)
	return ""
}

func (s *OfficialSender) cache(key, value string, ttl time.Duration) {
	if ttl <= 2*time.Minute {
		ttl = 5 * time.Minute
	}
	s.mu.Lock()
	if s.tokens == nil {
		s.tokens = map[string]cachedToken{}
	}
	s.tokens[key] = cachedToken{value: value, expiresAt: s.clock().Add(ttl)}
	s.mu.Unlock()
}
