//go:build integration

package repository

import (
	"context"
	"crypto/sha256"
	"errors"
	"net/netip"
	"os"
	"sync"
	"testing"
	"time"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	loginapp "github.com/appkernia/appkernia/server/internal/modules/loginprovider/application"
	login "github.com/appkernia/appkernia/server/internal/modules/loginprovider/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type failingOTPDispatcher struct {
	mu          sync.Mutex
	knownTarget []bool
}

func (dispatcher *failingOTPDispatcher) Queue(_ context.Context, _ pgx.Tx, challenge login.OTPChallenge) error {
	dispatcher.mu.Lock()
	defer dispatcher.mu.Unlock()
	dispatcher.knownTarget = append(dispatcher.knownTarget, challenge.UserID != nil)
	return errors.New("injected notification pipeline failure")
}

func TestConcurrentFirstOAuthLoginIsAtomicAndUsesAppLocale(t *testing.T) {
	pool := loginProviderTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	tenantID, appID, runtime := createLoginProviderRuntime(t, pool, "concurrent", "en-US")
	defer cleanupLoginProviderRuntime(pool, tenantID)
	repository := NewPostgres(pool, nil)
	identity := login.VerifiedIdentity{
		Issuer:           "https://github.com",
		ExternalClientID: runtime.ExternalClientID,
		Subject:          "subject-" + uuid.NewString(),
		ProviderUsername: "octocat",
		Profile:          []byte(`{}`),
	}
	start := make(chan struct{})
	results := make(chan login.ResolvedIdentity, 2)
	errorsChannel := make(chan error, 2)
	var workers sync.WaitGroup
	for index := 0; index < 2; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			resolved, err := repository.ResolveIdentity(ctx, login.IdentityResolution{
				Runtime:  runtime,
				Identity: identity,
				Mode:     "login",
			}, atomicSessionFixture)
			if err != nil {
				errorsChannel <- err
				return
			}
			results <- resolved
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	close(errorsChannel)
	for err := range errorsChannel {
		t.Fatalf("concurrent login failed: %v", err)
	}
	resolved := make([]login.ResolvedIdentity, 0, 2)
	for result := range results {
		resolved = append(resolved, result)
	}
	if len(resolved) != 2 || resolved[0].UserID != resolved[1].UserID {
		t.Fatalf("concurrent callbacks resolved %#v, want the same user", resolved)
	}
	assertLoginProviderCount(t, pool, `SELECT count(*) FROM iam.app_oauth_accounts WHERE app_id=$1 AND provider_code='github' AND subject=$2`, 1, appID, identity.Subject)
	assertLoginProviderCount(t, pool, `SELECT count(*) FROM app.user_memberships WHERE app_id=$1 AND source='federated_registration'`, 1, appID)
	assertLoginProviderCount(t, pool, `SELECT count(*) FROM iam.sessions WHERE app_id=$1 AND audience='ak-mobile'`, 2, appID)
	var displayName, locale string
	if err := pool.QueryRow(ctx, `SELECT display_name,locale FROM iam.users WHERE id=$1`, resolved[0].UserID).Scan(&displayName, &locale); err != nil {
		t.Fatal(err)
	}
	if displayName != "octocat" || locale != "en-US" {
		t.Fatalf("federated user display_name=%q locale=%q", displayName, locale)
	}
	var googleConfigID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO sys.login_provider_configs(
tenant_id,name,provider_code,external_client_id,public_config,status,last_preflight_at,last_preflight_status)
VALUES($1,$2,'google',$3,'{}','active',now(),'ready') RETURNING id`, tenantID, "Google "+uuid.NewString(), "google-client-"+uuid.NewString()).Scan(&googleConfigID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO app.application_login_provider_bindings(
tenant_id,app_id,provider_code,login_provider_config_id,enabled)
VALUES($1,$2,'google',$3,true)`, tenantID, appID, googleConfigID); err != nil {
		t.Fatal(err)
	}
	methods, err := repository.LoginMethods(ctx, appID, resolved[0].UserID)
	if err != nil {
		t.Fatal(err)
	}
	google := oauthMethod(methods, login.ProviderGoogle)
	if google.CanBind || google.BlockReason != "step_up_method_required" {
		t.Fatalf("OAuth-only user Google method=%#v, want server-blocked bind", google)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO app.user_login_identifiers(
tenant_id,app_id,user_id,identifier_type,normalized_value,display_hint,verified_at,status)
VALUES($1,$2,$3,'mobile','+15551234567','+15****4567',now(),'active')`, tenantID, appID, resolved[0].UserID); err != nil {
		t.Fatal(err)
	}
	methods, err = repository.LoginMethods(ctx, appID, resolved[0].UserID)
	if err != nil {
		t.Fatal(err)
	}
	google = oauthMethod(methods, login.ProviderGoogle)
	if !google.CanBind || google.BlockReason != "" {
		t.Fatalf("user with mobile step-up Google method=%#v, want bind enabled", google)
	}
}

func TestFirstOAuthLoginRollsBackWhenSessionPreparationFails(t *testing.T) {
	pool := loginProviderTestPool(t)
	ctx := context.Background()
	tenantID, appID, runtime := createLoginProviderRuntime(t, pool, "atomic-failure", "zh-CN")
	defer cleanupLoginProviderRuntime(pool, tenantID)
	repository := NewPostgres(pool, nil)
	subject := "subject-" + uuid.NewString()
	_, err := repository.ResolveIdentity(ctx, login.IdentityResolution{
		Runtime: runtime,
		Identity: login.VerifiedIdentity{
			Issuer:           "https://github.com",
			ExternalClientID: runtime.ExternalClientID,
			Subject:          subject,
		},
		Mode: "login",
	}, func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (login.AtomicLoginSession, error) {
		return login.AtomicLoginSession{}, errors.New("injected session preparation failure")
	})
	if err == nil {
		t.Fatal("expected injected session preparation failure")
	}
	assertLoginProviderCount(t, pool, `SELECT count(*) FROM iam.app_oauth_accounts WHERE app_id=$1 AND subject=$2`, 0, appID, subject)
	assertLoginProviderCount(t, pool, `SELECT count(*) FROM app.user_memberships WHERE app_id=$1 AND source='federated_registration'`, 0, appID)
	assertLoginProviderCount(t, pool, `SELECT count(*) FROM iam.sessions WHERE app_id=$1 AND audience='ak-mobile'`, 0, appID)
}

func TestOTPDeliveryFailureDoesNotEnumerateKnownIdentifiers(t *testing.T) {
	pool := loginProviderTestPool(t)
	ctx := context.Background()
	tenantID, appID, _ := createLoginProviderRuntime(t, pool, "otp-enumeration", "zh-CN")
	defer cleanupLoginProviderRuntime(pool, tenantID)
	knownMobile := "+15551234567"
	var userID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO iam.users(display_name,locale,status)
VALUES('OTP User','zh-CN','active') RETURNING id`).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO iam.tenant_members(tenant_id,user_id,status) VALUES($1,$2,'active')`, tenantID, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO app.user_memberships(app_id,tenant_id,user_id,source,status,verified_at)
VALUES($1,$2,$3,'self_registration','active',now())`, appID, tenantID, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO app.user_login_identifiers(
tenant_id,app_id,user_id,identifier_type,normalized_value,display_hint,verified_at,status)
VALUES($1,$2,$3,'mobile',$4,'+15****4567',now(),'active')`, tenantID, appID, userID, knownMobile); err != nil {
		t.Fatal(err)
	}
	dispatcher := &failingOTPDispatcher{}
	service := loginapp.NewService(nil, NewPostgres(pool, dispatcher), nil, nil, "", nil)
	ipAddress := netip.MustParseAddr("192.0.2.42")
	client := iamapp.ClientMetadata{IPAddress: &ipAddress, DeviceKey: uuid.NewString(), RequestID: "otp-enumeration"}
	type targetResult struct {
		first  loginapp.CodeChallengeResult
		second loginapp.CodeChallengeResult
	}
	results := make([]targetResult, 0, 2)
	for _, mobile := range []string{knownMobile, "+15557654321"} {
		first, err := service.SendLoginCode(ctx, appID, "mobile", mobile, "zh-CN", client)
		if err != nil || !first.Accepted {
			t.Fatalf("first accepted response for %s: %#v err=%v", mobile, first, err)
		}
		second, err := service.SendLoginCode(ctx, appID, "mobile", mobile, "zh-CN", client)
		if err != nil || !second.Accepted || second.ChallengeID != first.ChallengeID || second.RetryAfterSeconds != first.RetryAfterSeconds {
			t.Fatalf("cooldown response for %s: first=%#v second=%#v err=%v", mobile, first, second, err)
		}
		results = append(results, targetResult{first: first, second: second})
	}
	if results[0].first.ChallengeID == uuid.Nil || results[1].first.ChallengeID == uuid.Nil {
		t.Fatal("known and unknown responses must both contain persisted challenge IDs")
	}
	dispatcher.mu.Lock()
	knownTargets := append([]bool(nil), dispatcher.knownTarget...)
	dispatcher.mu.Unlock()
	if len(knownTargets) != 2 || !knownTargets[0] || knownTargets[1] {
		t.Fatalf("delivery readiness path calls=%v, want one known and one dummy call", knownTargets)
	}
	assertLoginProviderCount(t, pool, `SELECT count(*) FROM iam.verification_challenges
WHERE metadata->>'app_id'=$1 AND metadata->>'delivery_status'='failed'`, 2, appID.String())
}

func TestUsableLoginMethodCountIncludesAppIdentifiers(t *testing.T) {
	pool := loginProviderTestPool(t)
	ctx := context.Background()
	tenantID, appID, runtime := createLoginProviderRuntime(t, pool, "login-method-count", "zh-CN")
	defer cleanupLoginProviderRuntime(pool, tenantID)
	repository := NewPostgres(pool, nil)

	createUser := func(label string) (uuid.UUID, uuid.UUID) {
		t.Helper()
		var userID uuid.UUID
		if err := pool.QueryRow(ctx, `INSERT INTO iam.users(display_name,locale,status)
VALUES($1,'zh-CN','active') RETURNING id`, label).Scan(&userID); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO iam.tenant_members(tenant_id,user_id,status) VALUES($1,$2,'active')`, tenantID, userID); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO app.user_memberships(app_id,tenant_id,user_id,source,status,verified_at)
VALUES($1,$2,$3,'self_registration','active',now())`, appID, tenantID, userID); err != nil {
			t.Fatal(err)
		}
		var sessionID uuid.UUID
		if err := pool.QueryRow(ctx, `INSERT INTO iam.sessions(
user_id,tenant_id,app_id,audience,status,access_token_version,absolute_expires_at,idle_expires_at)
VALUES($1,$2,$3,'ak-mobile','active',1,now()+interval '1 hour',now()+interval '30 minutes')
RETURNING id`, userID, tenantID, appID).Scan(&sessionID); err != nil {
			t.Fatal(err)
		}
		return userID, sessionID
	}
	bindIdentifier := func(userID uuid.UUID, identifierType, value, hint string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `INSERT INTO app.user_login_identifiers(
tenant_id,app_id,user_id,identifier_type,normalized_value,display_hint,verified_at,status)
VALUES($1,$2,$3,$4,$5,$6,now(),'active')`, tenantID, appID, userID, identifierType, value, hint); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("two OTP identifiers allow exactly one unbind", func(t *testing.T) {
		userID, sessionID := createUser("Two OTP methods")
		bindIdentifier(userID, "email", "two-otp@example.test", "t***@example.test")
		bindIdentifier(userID, "mobile", "+15551234567", "+15****4567")

		methods, err := repository.LoginMethods(ctx, appID, userID)
		if err != nil {
			t.Fatal(err)
		}
		if methods.RemainingLoginMethods != 2 || !identifierMethod(methods, "email").CanUnbind || !identifierMethod(methods, "mobile").CanUnbind {
			t.Fatalf("two OTP methods=%#v, want both identifiers counted and removable", methods)
		}
		principal := login.Principal{TenantID: tenantID, UserID: userID, SessionID: sessionID}
		if err = repository.DeleteIdentifier(ctx, principal, appID, "email"); err != nil {
			t.Fatalf("delete email while mobile remains: %v", err)
		}
		methods, err = repository.LoginMethods(ctx, appID, userID)
		if err != nil {
			t.Fatal(err)
		}
		if methods.RemainingLoginMethods != 1 || identifierMethod(methods, "mobile").CanUnbind {
			t.Fatalf("single remaining mobile method=%#v, want last-method protection", methods)
		}
		if err = repository.DeleteIdentifier(ctx, principal, appID, "mobile"); !errors.Is(err, login.ErrLastLoginMethod) {
			t.Fatalf("delete final mobile error=%v, want ErrLastLoginMethod", err)
		}
	})

	t.Run("OAuth can be removed while mobile OTP remains", func(t *testing.T) {
		userID, sessionID := createUser("OAuth and mobile OTP")
		bindIdentifier(userID, "mobile", "+15557654321", "+15****4321")
		var accountID uuid.UUID
		if err := pool.QueryRow(ctx, `INSERT INTO iam.app_oauth_accounts(
tenant_id,app_id,user_id,provider_code,issuer,external_client_id,subject,status)
VALUES($1,$2,$3,'github','https://github.com',$4,$5,'active') RETURNING id`,
			tenantID, appID, userID, runtime.ExternalClientID, "subject-"+uuid.NewString()).Scan(&accountID); err != nil {
			t.Fatal(err)
		}

		methods, err := repository.LoginMethods(ctx, appID, userID)
		if err != nil {
			t.Fatal(err)
		}
		if methods.RemainingLoginMethods != 2 || !oauthMethod(methods, login.ProviderGitHub).CanUnbind {
			t.Fatalf("OAuth plus mobile methods=%#v, want OAuth removable", methods)
		}
		principal := login.Principal{TenantID: tenantID, UserID: userID, SessionID: sessionID}
		if err = repository.DeleteOAuthAccount(ctx, principal, appID, accountID); err != nil {
			t.Fatalf("delete OAuth while mobile remains: %v", err)
		}
		if err = repository.DeleteIdentifier(ctx, principal, appID, "mobile"); !errors.Is(err, login.ErrLastLoginMethod) {
			t.Fatalf("delete final mobile error=%v, want ErrLastLoginMethod", err)
		}
	})
}

func atomicSessionFixture(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (login.AtomicLoginSession, error) {
	id := uuid.New()
	hash := sha256.Sum256([]byte(id.String()))
	now := time.Now().UTC()
	return login.AtomicLoginSession{
		ID:                 id,
		RefreshTokenHash:   hash[:],
		AbsoluteExpiresAt:  now.Add(24 * time.Hour),
		IdleExpiresAt:      now.Add(time.Hour),
		RefreshExpiresAt:   now.Add(24 * time.Hour),
		AccessTokenVersion: 1,
		RequestID:          "login-provider-integration-" + id.String(),
	}, nil
}

func loginProviderTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func createLoginProviderRuntime(t *testing.T, pool *pgxpool.Pool, prefix, locale string) (uuid.UUID, uuid.UUID, login.RuntimeProvider) {
	t.Helper()
	ctx := context.Background()
	suffix := uuid.NewString()
	var tenantID, appID, configID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO iam.tenants(code,name) VALUES($1,$2) RETURNING id`, prefix+"-"+suffix, prefix+"-"+suffix).Scan(&tenantID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `UPDATE app.applications SET default_locale=$2,registration_enabled=true
WHERE tenant_id=$1 AND is_default RETURNING id`, tenantID, locale).Scan(&appID); err != nil {
		t.Fatal(err)
	}
	clientID := "github-client-" + suffix
	if err := pool.QueryRow(ctx, `INSERT INTO sys.login_provider_configs(
tenant_id,name,provider_code,external_client_id,public_config,status,last_preflight_at,last_preflight_status)
VALUES($1,$2,'github',$3,'{"app_return_uri":"https://app.example.test/oauth/"}','active',now(),'ready') RETURNING id`,
		tenantID, "GitHub "+suffix, clientID).Scan(&configID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO app.application_login_provider_bindings(
tenant_id,app_id,provider_code,login_provider_config_id,enabled)
VALUES($1,$2,'github',$3,true)`, tenantID, appID, configID); err != nil {
		t.Fatal(err)
	}
	repository := NewPostgres(pool, nil)
	runtime, err := repository.RuntimeProvider(ctx, appID, login.ProviderGitHub)
	if err != nil {
		t.Fatal(err)
	}
	return tenantID, appID, runtime
}

func cleanupLoginProviderRuntime(pool *pgxpool.Pool, tenantID uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, _ = pool.Exec(ctx, `DELETE FROM iam.tenants WHERE id=$1`, tenantID)
}

func assertLoginProviderCount(t *testing.T, pool *pgxpool.Pool, query string, expected int, arguments ...any) {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), query, arguments...).Scan(&count); err != nil || count != expected {
		t.Fatalf("count=%d want=%d err=%v query=%s", count, expected, err, query)
	}
}

func oauthMethod(methods login.LoginMethods, providerCode string) login.OAuthAccount {
	for _, method := range methods.OAuthAccounts {
		if method.ProviderCode == providerCode {
			return method
		}
	}
	return login.OAuthAccount{}
}

func identifierMethod(methods login.LoginMethods, identifierType string) login.Identifier {
	for _, method := range methods.Identifiers {
		if method.IdentifierType == identifierType {
			return method
		}
	}
	return login.Identifier{}
}
