package application

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	iam "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	login "github.com/appkernia/appkernia/server/internal/modules/loginprovider/domain"
	"github.com/google/uuid"
)

const (
	flowLifetime   = 10 * time.Minute
	stepUpLifetime = 5 * time.Minute
	otpLifetime    = 10 * time.Minute
	otpRetryAfter  = time.Minute
)

var mobilePattern = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

type mobileSessionIssuer interface {
	IssueMobileSession(context.Context, uuid.UUID, uuid.UUID, string, iamapp.ClientMetadata) (iamapp.SessionTokens, error)
}

type atomicMobileSessionPreparer interface {
	PrepareAtomicMobileSession(uuid.UUID, uuid.UUID, uuid.UUID, string, iamapp.ClientMetadata) (iamapp.PreparedMobileSession, error)
}

type passwordVerifier interface {
	VerifyUserPassword(context.Context, uuid.UUID, string) error
}

type MobileCallbackResult struct {
	Mode            string                `json:"mode"`
	Account         *login.OAuthAccount   `json:"account,omitempty"`
	Session         *iamapp.SessionTokens `json:"session,omitempty"`
	StepUpToken     string                `json:"step_up_token,omitempty"`
	StepUpExpiresAt *time.Time            `json:"step_up_expires_at,omitempty"`
}

type BrowserCallbackResult struct {
	RedirectURI string
}

type CodeChallengeResult struct {
	ChallengeID       uuid.UUID `json:"challenge_id"`
	Accepted          bool      `json:"accepted"`
	RetryAfterSeconds int       `json:"retry_after_seconds"`
	ExpiresAt         time.Time `json:"expires_at"`
}

type OTPLoginInput struct {
	IdentifierType   string    `json:"identifier_type"`
	Identifier       string    `json:"identifier"`
	ChallengeID      uuid.UUID `json:"challenge_id"`
	VerificationCode string    `json:"verification_code"`
}

type IdentifierCodeInput struct {
	Identifier string `json:"identifier"`
}

type IdentifierVerifyInput struct {
	Identifier       string    `json:"identifier"`
	ChallengeID      uuid.UUID `json:"challenge_id"`
	VerificationCode string    `json:"verification_code"`
	StepUpToken      string    `json:"step_up_token"`
}

type StepUpCodeInput struct {
	IdentifierID uuid.UUID `json:"identifier_id"`
	Purpose      string    `json:"purpose"`
	Resource     string    `json:"resource"`
}

type StepUpInput struct {
	Method           string    `json:"method"`
	Purpose          string    `json:"purpose"`
	Resource         string    `json:"resource"`
	Password         string    `json:"password,omitempty"`
	IdentifierID     uuid.UUID `json:"identifier_id,omitempty"`
	ChallengeID      uuid.UUID `json:"challenge_id,omitempty"`
	VerificationCode string    `json:"verification_code,omitempty"`
}

type StepUpResult struct {
	StepUpToken string    `json:"step_up_token"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type stepUpClaims struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	SessionID uuid.UUID `json:"session_id"`
	AppID     uuid.UUID `json:"app_id"`
	Purpose   string    `json:"purpose"`
	Resource  string    `json:"resource"`
	Method    string    `json:"method"`
	Expires   int64     `json:"expires"`
}

func bearerPrincipal(authenticated iam.AuthenticatedContext, requestID, ipAddress, userAgent string) login.Principal {
	return login.Principal{
		TenantID: authenticated.Tenant.ID, UserID: authenticated.User.ID, SessionID: authenticated.SessionID,
		RequestID: requestID, IPAddress: ipAddress, UserAgent: userAgent,
	}
}

func (service *Service) authenticateMobile(ctx context.Context, token string, appID uuid.UUID) (iam.AuthenticatedContext, error) {
	authenticated, err := service.auth.Authenticate(ctx, token, "ak-mobile")
	if err != nil {
		return iam.AuthenticatedContext{}, err
	}
	if authenticated.AppID == nil || *authenticated.AppID != appID {
		return iam.AuthenticatedContext{}, login.ErrForbidden
	}
	return authenticated, nil
}

func deviceHash(deviceKey string) ([]byte, error) {
	deviceKey = strings.TrimSpace(deviceKey)
	if len(deviceKey) < 16 || len(deviceKey) > 512 {
		return nil, login.ErrInvalid
	}
	digest := sha256.Sum256([]byte(deviceKey))
	return digest[:], nil
}

func opaque(bytes int) (string, error) {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func hashString(value string) []byte {
	digest := sha256.Sum256([]byte(value))
	return digest[:]
}

func supports(descriptor login.ProviderDescriptor, platform, variant string) bool {
	platformOK, variantOK := false, false
	for _, candidate := range descriptor.SupportedPlatforms {
		platformOK = platformOK || candidate == platform
	}
	for _, candidate := range descriptor.BuildVariants {
		variantOK = variantOK || candidate == variant
	}
	return platformOK && variantOK && ((platform == "ios" && variant == "ios") ||
		(platform == "android" && (variant == "android_google" || variant == "android_china")) ||
		(platform == "harmony" && variant == "harmony"))
}

func (service *Service) MobileProviders(ctx context.Context, appID uuid.UUID, platform, variant string) (login.MobileProviderList, error) {
	platform, variant = strings.ToLower(strings.TrimSpace(platform)), strings.ToLower(strings.TrimSpace(variant))
	runtimes, err := service.repository.RuntimeProviders(ctx, appID)
	if err != nil {
		return login.MobileProviderList{}, err
	}
	items := make([]login.MobileProvider, 0, len(runtimes))
	for _, runtime := range runtimes {
		descriptor, ok := login.Descriptor(runtime.ProviderCode)
		if !ok || !supports(descriptor, platform, variant) || !login.ConfigSupportsTarget(runtime.ProviderCode, runtime.PublicConfig, platform, variant) {
			continue
		}
		hash, hashErr := login.BuildConfigHash(runtime.ProviderCode, runtime.ExternalClientID, runtime.PublicConfig)
		build, buildErr := login.BuildConfig(runtime.ProviderCode, runtime.ExternalClientID, runtime.PublicConfig, login.GitHubBrowserCallbackURI(service.callbackURI))
		if hashErr != nil || buildErr != nil {
			return login.MobileProviderList{}, login.ErrProviderUnavailable
		}
		items = append(items, login.MobileProvider{
			ProviderCode: runtime.ProviderCode, DisplayNameKey: descriptor.DisplayNameKey, IconKey: descriptor.IconKey,
			AuthorizationKind: descriptor.AuthorizationKind, SupportedPlatforms: descriptor.SupportedPlatforms,
			BuildVariants: descriptor.BuildVariants, LoginEnabled: runtime.Enabled, BindingEnabled: runtime.Enabled,
			SortOrder: runtime.SortOrder, ConfigSchemaVersion: descriptor.ConfigSchemaVersion,
			BuildConfigHash: hash, BuildConfig: build,
		})
	}
	return login.MobileProviderList{Items: items}, nil
}

func (service *Service) Authorize(ctx context.Context, token string, appID uuid.UUID, providerCode, deviceKey string, input login.AuthorizeInput) (login.AuthorizeResult, error) {
	providerCode = strings.ToLower(strings.TrimSpace(providerCode))
	input.Mode, input.Platform, input.BuildVariant = strings.ToLower(strings.TrimSpace(input.Mode)), strings.ToLower(strings.TrimSpace(input.Platform)), strings.ToLower(strings.TrimSpace(input.BuildVariant))
	descriptor, ok := login.Descriptor(providerCode)
	if !ok || !supports(descriptor, input.Platform, input.BuildVariant) || (input.Mode != "login" && input.Mode != "bind" && input.Mode != "reauth") {
		return login.AuthorizeResult{}, login.ErrInvalid
	}
	deviceKeyHash, err := deviceHash(deviceKey)
	if err != nil {
		return login.AuthorizeResult{}, err
	}
	var authenticated iam.AuthenticatedContext
	var runtime login.RuntimeProvider
	if input.Mode == "login" {
		if input.StepUpToken != "" || input.ReauthPurpose != "" || input.AccountID != nil {
			return login.AuthorizeResult{}, login.ErrInvalid
		}
		runtime, err = service.repository.RuntimeProvider(ctx, appID, providerCode)
	} else {
		authenticated, err = service.authenticateMobile(ctx, token, appID)
		if err != nil {
			return login.AuthorizeResult{}, err
		}
		if input.Mode == "bind" {
			if input.ReauthPurpose != "" || input.AccountID != nil {
				return login.AuthorizeResult{}, login.ErrInvalid
			}
			if err = service.consumeStepUp(ctx, input.StepUpToken, authenticated, appID, "oauth_bind", providerCode, ""); err != nil {
				return login.AuthorizeResult{}, login.ErrStepUpRequired
			}
			runtime, err = service.repository.RuntimeProvider(ctx, appID, providerCode)
		} else {
			if input.AccountID == nil || *input.AccountID == uuid.Nil || (input.ReauthPurpose != "oauth_unbind" && input.ReauthPurpose != "account_delete") {
				return login.AuthorizeResult{}, login.ErrInvalid
			}
			runtime, err = service.repository.RuntimeProviderForReauth(ctx, appID, providerCode, authenticated.User.ID, *input.AccountID)
		}
	}
	if err != nil {
		return login.AuthorizeResult{}, err
	}
	if !login.ConfigSupportsTarget(providerCode, runtime.PublicConfig, input.Platform, input.BuildVariant) {
		return login.AuthorizeResult{}, login.ErrProviderUnavailable
	}
	expectedHash, err := login.BuildConfigHash(providerCode, runtime.ExternalClientID, runtime.PublicConfig)
	if err != nil || subtle.ConstantTimeCompare([]byte(expectedHash), []byte(strings.TrimSpace(input.BuildConfigHash))) != 1 {
		return login.AuthorizeResult{}, login.ErrConfigStale
	}
	state, err := opaque(32)
	if err != nil {
		return login.AuthorizeResult{}, err
	}
	var nonce, verifier, challenge string
	var nonceHash, verifierCiphertext []byte
	var verifierKeyVersion *int32
	if providerCode == login.ProviderApple || providerCode == login.ProviderGoogle {
		nonce, err = opaque(32)
		if err != nil {
			return login.AuthorizeResult{}, err
		}
		nonceHash = hashString(nonce)
	}
	if providerCode == login.ProviderGitHub {
		verifier, err = opaque(48)
		if err != nil {
			return login.AuthorizeResult{}, err
		}
		challenge = base64.RawURLEncoding.EncodeToString(hashString(verifier))
		if service.sealer == nil {
			return login.AuthorizeResult{}, login.ErrProviderUnavailable
		}
		sealed, version, sealErr := service.sealer.Seal([]byte(verifier), flowAAD(appID, providerCode, hashString(state)))
		if sealErr != nil {
			return login.AuthorizeResult{}, sealErr
		}
		verifierCiphertext, verifierKeyVersion = sealed, &version
	}
	now := service.clock().UTC()
	create := login.FlowCreate{
		TenantID: runtime.TenantID, AppID: appID, ConfigID: runtime.ConfigID, ProviderCode: providerCode,
		Mode: input.Mode, Platform: input.Platform, BuildVariant: input.BuildVariant,
		StateHash: hashString(state), NonceHash: nonceHash, PKCECiphertext: verifierCiphertext,
		PKCEKeyVersion: verifierKeyVersion, DeviceKeyHash: deviceKeyHash, ExpiresAt: now.Add(flowLifetime),
		ReauthPurpose: input.ReauthPurpose, TargetOAuthAccountID: input.AccountID,
	}
	if input.Mode != "login" {
		create.UserID, create.SessionID = &authenticated.User.ID, &authenticated.SessionID
	}
	flow, err := service.repository.CreateFlow(ctx, create)
	if err != nil {
		return login.AuthorizeResult{}, err
	}
	result := login.AuthorizeResult{FlowID: flow.ID, ProviderCode: providerCode, Mode: input.Mode,
		AuthorizationKind: descriptor.AuthorizationKind, State: state, Nonce: nonce, ExpiresAt: flow.ExpiresAt}
	if providerCode == login.ProviderGitHub {
		result.AuthorizationURL, err = service.adapter.AuthorizationURL(ctx, descriptor, runtime, state, challenge, nonce)
	}
	return result, err
}

func flowAAD(appID uuid.UUID, providerCode string, stateHash []byte) string {
	return "oauth-flow:" + appID.String() + ":" + providerCode + ":" + hex.EncodeToString(stateHash)
}

func (service *Service) runtimeAndSecrets(ctx context.Context, flow login.Flow) (login.RuntimeProvider, map[string]string, error) {
	var runtime login.RuntimeProvider
	var err error
	if flow.Mode == "reauth" && flow.UserID != nil && flow.TargetOAuthAccountID != nil {
		runtime, err = service.repository.RuntimeProviderForReauth(ctx, flow.AppID, flow.ProviderCode, *flow.UserID, *flow.TargetOAuthAccountID)
	} else {
		runtime, err = service.repository.RuntimeProvider(ctx, flow.AppID, flow.ProviderCode)
	}
	if err != nil || runtime.ConfigID != flow.ConfigID {
		return runtime, nil, login.ErrProviderUnavailable
	}
	secrets := map[string]string{}
	if len(runtime.SecretCiphertext) > 0 {
		if service.sealer == nil {
			return runtime, nil, login.ErrProviderUnavailable
		}
		plaintext, openErr := service.sealer.Open(runtime.SecretCiphertext, configAAD(runtime.TenantID, runtime.ConfigID))
		if openErr != nil || json.Unmarshal(plaintext, &secrets) != nil {
			return runtime, nil, login.ErrProviderUnavailable
		}
	}
	return runtime, secrets, nil
}

func (service *Service) verifyRequest(ctx context.Context, flow login.Flow, runtime login.RuntimeProvider, secrets map[string]string, input login.CallbackInput) (login.VerifiedIdentity, login.VerificationRequest, error) {
	request := login.VerificationRequest{
		ProviderCode: flow.ProviderCode, ExternalClientID: runtime.ExternalClientID, PublicConfig: runtime.PublicConfig,
		Secrets: secrets, AuthorizationCode: strings.TrimSpace(input.AuthorizationCode), IDToken: strings.TrimSpace(input.IDToken),
		Mode: flow.Mode, RedirectURI: login.GitHubBrowserCallbackURI(service.callbackURI),
	}
	if len(flow.PKCECiphertext) > 0 {
		plaintext, err := service.sealer.Open(flow.PKCECiphertext, flowAAD(flow.AppID, flow.ProviderCode, flow.StateHash))
		if err != nil {
			return login.VerifiedIdentity{}, request, login.ErrProviderUnavailable
		}
		request.PKCEVerifier = string(plaintext)
	}
	identity, err := service.adapter.Verify(ctx, request)
	if err != nil {
		return login.VerifiedIdentity{}, request, err
	}
	if (flow.ProviderCode == login.ProviderApple || flow.ProviderCode == login.ProviderGoogle) &&
		(len(flow.NonceHash) != 32 || subtle.ConstantTimeCompare(hashString(identity.Nonce), flow.NonceHash) != 1) {
		return login.VerifiedIdentity{}, request, login.ErrCallbackInvalid
	}
	if flow.ProviderCode == login.ProviderApple && identity.DisplayName == "" {
		identity.DisplayName = strings.Join(strings.Fields(input.DisplayName), " ")
	}
	return identity, request, nil
}

func validAppleDisplayName(providerCode, displayName string) bool {
	if displayName == "" {
		return true
	}
	if providerCode != login.ProviderApple {
		return false
	}
	cleaned := strings.Join(strings.Fields(displayName), " ")
	if cleaned == "" || len([]rune(cleaned)) > 120 {
		return false
	}
	for _, value := range cleaned {
		if value < 0x20 || value == 0x7f {
			return false
		}
	}
	return true
}

func (service *Service) Callback(ctx context.Context, token string, appID uuid.UUID, providerCode, deviceKey string, input login.CallbackInput, client iamapp.ClientMetadata) (MobileCallbackResult, error) {
	providerCode = strings.ToLower(strings.TrimSpace(providerCode))
	deviceKeyHash, err := deviceHash(deviceKey)
	if err != nil || input.FlowID == uuid.Nil || !validAppleDisplayName(providerCode, input.DisplayName) {
		return MobileCallbackResult{}, login.ErrCallbackInvalid
	}
	flow, err := service.repository.GetFlow(ctx, input.FlowID, appID, providerCode, deviceKeyHash)
	if err != nil {
		return MobileCallbackResult{}, err
	}
	if flow.Mode != "login" {
		authenticated, authErr := service.authenticateMobile(ctx, token, appID)
		if authErr != nil || flow.UserID == nil || flow.SessionID == nil || authenticated.User.ID != *flow.UserID || authenticated.SessionID != *flow.SessionID {
			return MobileCallbackResult{}, login.ErrFlowInvalid
		}
	}
	var identity login.VerifiedIdentity
	if providerCode == login.ProviderGitHub {
		if strings.TrimSpace(input.OneTimeTicket) == "" || input.AuthorizationCode != "" || input.IDToken != "" {
			return MobileCallbackResult{}, login.ErrCallbackInvalid
		}
		flow, err = service.repository.ClaimTicketFlow(ctx, flow.ID, appID, providerCode, deviceKeyHash, hashString(input.OneTimeTicket))
		if err != nil {
			return MobileCallbackResult{}, err
		}
		plaintext, openErr := service.sealer.Open(flow.VerifiedIdentityCiphertext, flowAAD(appID, providerCode, flow.StateHash)+":identity")
		if openErr != nil || json.Unmarshal(plaintext, &identity) != nil {
			return MobileCallbackResult{}, login.ErrCallbackInvalid
		}
	} else {
		if strings.TrimSpace(input.State) == "" || subtle.ConstantTimeCompare(hashString(input.State), flow.StateHash) != 1 || input.OneTimeTicket != "" {
			return MobileCallbackResult{}, login.ErrCallbackInvalid
		}
		runtime, secrets, loadErr := service.runtimeAndSecrets(ctx, flow)
		if loadErr != nil {
			return MobileCallbackResult{}, loadErr
		}
		var verifyRequest login.VerificationRequest
		identity, verifyRequest, err = service.verifyRequest(ctx, flow, runtime, secrets, input)
		if err != nil {
			return MobileCallbackResult{}, err
		}
		if flow.ProviderCode == login.ProviderApple && flow.Mode == "reauth" {
			if strings.TrimSpace(input.AuthorizationCode) == "" || strings.TrimSpace(input.IDToken) == "" {
				return MobileCallbackResult{}, login.ErrCallbackInvalid
			}
			revoker, ok := service.adapter.(login.AppleRevoker)
			if !ok {
				return MobileCallbackResult{}, login.ErrProviderUnavailable
			}
			verifyRequest.ExpectedSubject = identity.Subject
			if err = revoker.RevokeApple(ctx, verifyRequest); err != nil {
				return MobileCallbackResult{}, err
			}
		}
		flow, err = service.repository.ClaimNativeFlow(ctx, flow.ID, appID, providerCode, deviceKeyHash, flow.StateHash)
		if err != nil {
			return MobileCallbackResult{}, err
		}
	}
	runtime, _, err := service.runtimeAndSecrets(ctx, flow)
	if err != nil {
		return MobileCallbackResult{}, err
	}
	var preparedSession iamapp.PreparedMobileSession
	var sessionFactory login.LoginSessionFactory
	if flow.Mode == "login" {
		preparer, ok := service.auth.(atomicMobileSessionPreparer)
		if !ok {
			return MobileCallbackResult{}, login.ErrProviderUnavailable
		}
		sessionFactory = func(_ context.Context, userID, tenantID, scopedAppID uuid.UUID) (login.AtomicLoginSession, error) {
			prepared, prepareErr := preparer.PrepareAtomicMobileSession(userID, tenantID, scopedAppID, "oauth", client)
			if prepareErr != nil {
				return login.AtomicLoginSession{}, prepareErr
			}
			preparedSession = prepared
			ipAddress := ""
			if prepared.IPAddress != nil {
				ipAddress = prepared.IPAddress.String()
			}
			return login.AtomicLoginSession{
				ID: prepared.Tokens.SessionID, RefreshTokenHash: prepared.RefreshTokenHash,
				AbsoluteExpiresAt: prepared.AbsoluteExpiresAt, IdleExpiresAt: prepared.IdleExpiresAt,
				RefreshExpiresAt: prepared.RefreshExpiresAt, IPAddress: ipAddress,
				UserAgent: prepared.UserAgent, DeviceKey: prepared.DeviceKey, RequestID: prepared.RequestID,
				AccessTokenVersion: prepared.AccessTokenVersion,
			}, nil
		}
	}
	resolved, err := service.repository.ResolveIdentity(ctx, login.IdentityResolution{
		Runtime: runtime, Identity: identity, Mode: flow.Mode, UserID: flow.UserID, SessionID: flow.SessionID,
		TargetOAuthAccountID: flow.TargetOAuthAccountID,
	}, sessionFactory)
	if err != nil {
		return MobileCallbackResult{}, err
	}
	result := MobileCallbackResult{Mode: flow.Mode, Account: &resolved.Account}
	if flow.Mode == "login" {
		result.Session = &preparedSession.Tokens
	} else if flow.Mode == "reauth" {
		authenticated, authErr := service.authenticateMobile(ctx, token, appID)
		if authErr != nil || flow.TargetOAuthAccountID == nil {
			return MobileCallbackResult{}, login.ErrStepUpInvalid
		}
		resource := flow.TargetOAuthAccountID.String()
		step, issueErr := service.issueStepUp(ctx, authenticated, appID, flow.ReauthPurpose, resource, "oauth:"+flow.ProviderCode, client)
		if issueErr != nil {
			return MobileCallbackResult{}, issueErr
		}
		result.StepUpToken, result.StepUpExpiresAt = step.StepUpToken, &step.ExpiresAt
	}
	return result, nil
}

func (service *Service) GitHubBrowserCallback(ctx context.Context, code, state string) (BrowserCallbackResult, error) {
	code, state = strings.TrimSpace(code), strings.TrimSpace(state)
	if code == "" || state == "" {
		return BrowserCallbackResult{}, login.ErrCallbackInvalid
	}
	flow, err := service.repository.GetBrowserFlow(ctx, login.ProviderGitHub, hashString(state))
	if err != nil {
		return BrowserCallbackResult{}, err
	}
	runtime, secrets, err := service.runtimeAndSecrets(ctx, flow)
	if err != nil {
		return BrowserCallbackResult{}, err
	}
	identity, _, err := service.verifyRequest(ctx, flow, runtime, secrets, login.CallbackInput{AuthorizationCode: code})
	if err != nil {
		return BrowserCallbackResult{}, err
	}
	identityJSON, _ := json.Marshal(identity)
	ciphertext, keyVersion, err := service.sealer.Seal(identityJSON, flowAAD(flow.AppID, flow.ProviderCode, flow.StateHash)+":identity")
	if err != nil {
		return BrowserCallbackResult{}, err
	}
	ticket, err := opaque(32)
	if err != nil {
		return BrowserCallbackResult{}, err
	}
	if err = service.repository.MarkBrowserVerified(ctx, flow.ID, hashString(ticket), ciphertext, keyVersion); err != nil {
		return BrowserCallbackResult{}, err
	}
	var public login.GitHubPublicConfig
	if json.Unmarshal(runtime.PublicConfig, &public) != nil {
		return BrowserCallbackResult{}, login.ErrProviderUnavailable
	}
	returnURI, err := login.CanonicalHTTPSAppLink(public.AppReturnURI, false)
	if err != nil {
		return BrowserCallbackResult{}, login.ErrProviderUnavailable
	}
	parsed, _ := url.Parse(returnURI)
	query := parsed.Query()
	query.Set("flow_id", flow.ID.String())
	query.Set("one_time_ticket", ticket)
	query.Set("provider", login.ProviderGitHub)
	parsed.RawQuery = query.Encode()
	return BrowserCallbackResult{RedirectURI: parsed.String()}, nil
}

func normalizeIdentifier(identifierType, raw string) (string, string, error) {
	identifierType, value := strings.ToLower(strings.TrimSpace(identifierType)), strings.TrimSpace(raw)
	switch identifierType {
	case "email":
		value = strings.ToLower(value)
		parsed, err := mail.ParseAddress(value)
		if err != nil || parsed.Address != value || len(value) > 254 {
			return "", "", login.ErrInvalid
		}
		parts := strings.SplitN(value, "@", 2)
		return value, string([]rune(parts[0])[:1]) + "***@" + parts[1], nil
	case "mobile":
		if !mobilePattern.MatchString(value) {
			return "", "", login.ErrInvalid
		}
		return value, value[:3] + "****" + value[len(value)-4:], nil
	default:
		return "", "", login.ErrInvalid
	}
}

func newVerificationCode() (string, error) {
	var raw [4]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	number := uint32(raw[0])<<24 | uint32(raw[1])<<16 | uint32(raw[2])<<8 | uint32(raw[3])
	return fmt.Sprintf("%06d", number%1000000), nil
}

func validCode(value string) bool {
	if len(value) != 6 {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func (service *Service) sendCode(ctx context.Context, appID uuid.UUID, tenantID uuid.UUID, userID *uuid.UUID, identifierType, raw, purpose, locale string, client iamapp.ClientMetadata) (CodeChallengeResult, error) {
	value, hint, err := normalizeIdentifier(identifierType, raw)
	if err != nil {
		return CodeChallengeResult{}, err
	}
	code, err := newVerificationCode()
	if err != nil {
		return CodeChallengeResult{}, err
	}
	now := service.clock().UTC()
	deviceKeyHash, err := deviceHash(client.DeviceKey)
	if err != nil {
		return CodeChallengeResult{}, err
	}
	challenge := login.OTPChallenge{ID: uuid.New(), TenantID: tenantID, AppID: appID, UserID: userID,
		IdentifierType: identifierType, NormalizedValue: value, DisplayHint: hint, SecretHash: hashString(code),
		Code: code, Purpose: purpose, Locale: locale, ExpiresAt: now.Add(otpLifetime), CreatedIP: ipString(client), DeviceKeyHash: deviceKeyHash}
	storedID, createErr := service.repository.CreateOTPChallenge(ctx, challenge)
	if createErr != nil && !errors.Is(createErr, login.ErrConflict) {
		return CodeChallengeResult{}, createErr
	}
	if storedID != uuid.Nil {
		challenge.ID = storedID
	}
	return CodeChallengeResult{ChallengeID: challenge.ID, Accepted: true, RetryAfterSeconds: int(otpRetryAfter.Seconds()), ExpiresAt: challenge.ExpiresAt}, nil
}

func (service *Service) SendLoginCode(ctx context.Context, appID uuid.UUID, identifierType, identifier, locale string, client iamapp.ClientMetadata) (CodeChallengeResult, error) {
	tenantID, defaultLocale, err := service.repository.ResolveApp(ctx, appID)
	if err != nil {
		return CodeChallengeResult{}, err
	}
	if locale != "en-US" && locale != "zh-CN" {
		locale = defaultLocale
	}
	value, _, normalizeErr := normalizeIdentifier(identifierType, identifier)
	if normalizeErr != nil {
		return CodeChallengeResult{}, normalizeErr
	}
	userID, foundTenant, userLocale, findErr := service.repository.FindOTPLoginUser(ctx, appID, identifierType, value)
	var challengeUserID *uuid.UUID
	if findErr == nil && foundTenant == tenantID && userID != uuid.Nil {
		challengeUserID = &userID
		if userLocale == "zh-CN" || userLocale == "en-US" {
			locale = userLocale
		}
	}
	// Unknown identifiers deliberately use the same persisted cooldown and
	// App/IP/device rate-limit path. The nil user marks a non-delivery dummy;
	// ConsumeOTPChallenge still fails closed because no active identifier joins.
	return service.sendCode(ctx, appID, tenantID, challengeUserID, identifierType, identifier, "login", locale, client)
}

func (service *Service) OTPLogin(ctx context.Context, appID uuid.UUID, input OTPLoginInput, client iamapp.ClientMetadata) (iamapp.SessionTokens, error) {
	value, _, err := normalizeIdentifier(input.IdentifierType, input.Identifier)
	if err != nil || input.ChallengeID == uuid.Nil || !validCode(input.VerificationCode) {
		return iamapp.SessionTokens{}, login.ErrOTPInvalid
	}
	userID, err := service.repository.ConsumeOTPChallenge(ctx, login.OTPConsume{ID: input.ChallengeID, AppID: appID,
		IdentifierType: input.IdentifierType, NormalizedValue: value, TargetHash: hashString(value), SecretHash: hashString(input.VerificationCode), Purpose: "login"})
	if err != nil {
		return iamapp.SessionTokens{}, err
	}
	issuer, ok := service.auth.(mobileSessionIssuer)
	if !ok {
		return iamapp.SessionTokens{}, login.ErrProviderUnavailable
	}
	authMethod := "email_otp"
	if input.IdentifierType == "mobile" {
		authMethod = "sms_otp"
	}
	return issuer.IssueMobileSession(ctx, userID, appID, authMethod, client)
}

func (service *Service) SendIdentifierCode(ctx context.Context, token string, appID uuid.UUID, identifierType string, input IdentifierCodeInput, client iamapp.ClientMetadata) (CodeChallengeResult, error) {
	authenticated, err := service.authenticateMobile(ctx, token, appID)
	if err != nil {
		return CodeChallengeResult{}, err
	}
	return service.sendCode(ctx, appID, authenticated.Tenant.ID, &authenticated.User.ID, identifierType, input.Identifier, "bind", authenticated.User.Locale, client)
}

func (service *Service) VerifyIdentifier(ctx context.Context, token string, appID uuid.UUID, identifierType string, input IdentifierVerifyInput, client iamapp.ClientMetadata) (login.Identifier, error) {
	authenticated, err := service.authenticateMobile(ctx, token, appID)
	if err != nil {
		return login.Identifier{}, err
	}
	value, hint, err := normalizeIdentifier(identifierType, input.Identifier)
	if err != nil || input.ChallengeID == uuid.Nil || !validCode(input.VerificationCode) {
		return login.Identifier{}, login.ErrOTPInvalid
	}
	methods, methodsErr := service.repository.LoginMethods(ctx, appID, authenticated.User.ID)
	if methodsErr != nil {
		return login.Identifier{}, methodsErr
	}
	for _, current := range methods.Identifiers {
		if current.IdentifierType == identifierType && current.Status != "unbound" {
			if err = service.consumeStepUp(ctx, input.StepUpToken, authenticated, appID, "identifier_change", identifierType, ""); err != nil {
				return login.Identifier{}, login.ErrStepUpRequired
			}
			break
		}
	}
	return service.repository.UpsertIdentifier(ctx, login.IdentifierMutation{
		Principal: bearerPrincipal(authenticated, client.RequestID, ipString(client), client.UserAgent), AppID: appID,
		IdentifierType: identifierType, NormalizedValue: value, DisplayHint: hint, ChallengeID: input.ChallengeID,
		TargetHash: hashString(value), SecretHash: hashString(input.VerificationCode),
	})
}

func ipString(client iamapp.ClientMetadata) string {
	if client.IPAddress == nil {
		return ""
	}
	return client.IPAddress.String()
}

func validStepUpScope(purpose, resource string) bool {
	if strings.TrimSpace(resource) == "" {
		return false
	}
	switch purpose {
	case "oauth_bind", "oauth_unbind", "identifier_change", "identifier_unbind", "account_delete":
		return true
	default:
		return false
	}
}

func (service *Service) SendStepUpCode(ctx context.Context, token string, appID uuid.UUID, input StepUpCodeInput, client iamapp.ClientMetadata) (CodeChallengeResult, error) {
	authenticated, err := service.authenticateMobile(ctx, token, appID)
	if err != nil || !validStepUpScope(input.Purpose, input.Resource) {
		return CodeChallengeResult{}, login.ErrInvalid
	}
	target, err := service.repository.IdentifierTarget(ctx, appID, authenticated.User.ID, input.IdentifierID)
	if err != nil || target.TenantID != authenticated.Tenant.ID {
		return CodeChallengeResult{}, login.ErrForbidden
	}
	return service.sendCode(ctx, appID, target.TenantID, &target.UserID, target.IdentifierType, target.NormalizedValue, "step_up", target.Locale, client)
}

func (service *Service) StepUp(ctx context.Context, token string, appID uuid.UUID, input StepUpInput, client iamapp.ClientMetadata) (StepUpResult, error) {
	authenticated, err := service.authenticateMobile(ctx, token, appID)
	if err != nil || !validStepUpScope(input.Purpose, input.Resource) {
		return StepUpResult{}, login.ErrStepUpInvalid
	}
	method := strings.ToLower(strings.TrimSpace(input.Method))
	switch method {
	case "password":
		methods, methodsErr := service.repository.LoginMethods(ctx, appID, authenticated.User.ID)
		if methodsErr != nil || !methods.Password.LoginCapable {
			return StepUpResult{}, login.ErrStepUpInvalid
		}
		verifier, ok := service.auth.(passwordVerifier)
		if !ok || verifier.VerifyUserPassword(ctx, authenticated.User.ID, input.Password) != nil {
			return StepUpResult{}, login.ErrStepUpInvalid
		}
	case "email_otp", "mobile_otp":
		identifierType := strings.TrimSuffix(method, "_otp")
		target, targetErr := service.repository.IdentifierTarget(ctx, appID, authenticated.User.ID, input.IdentifierID)
		if targetErr != nil || target.IdentifierType != identifierType || !validCode(input.VerificationCode) {
			return StepUpResult{}, login.ErrStepUpInvalid
		}
		userID, consumeErr := service.repository.ConsumeOTPChallenge(ctx, login.OTPConsume{
			ID: input.ChallengeID, AppID: appID, IdentifierType: identifierType, NormalizedValue: target.NormalizedValue,
			TargetHash: hashString(target.NormalizedValue), SecretHash: hashString(input.VerificationCode), Purpose: "step_up",
		})
		if consumeErr != nil || userID != authenticated.User.ID {
			return StepUpResult{}, login.ErrStepUpInvalid
		}
	default:
		return StepUpResult{}, login.ErrStepUpInvalid
	}
	return service.issueStepUp(ctx, authenticated, appID, input.Purpose, input.Resource, method, client)
}

func (service *Service) issueStepUp(ctx context.Context, authenticated iam.AuthenticatedContext, appID uuid.UUID, purpose, resource, method string, client iamapp.ClientMetadata) (StepUpResult, error) {
	if len(service.stepUpKey) < 32 || !validStepUpScope(purpose, resource) {
		return StepUpResult{}, login.ErrStepUpInvalid
	}
	expires := service.clock().UTC().Add(stepUpLifetime)
	claims := stepUpClaims{ID: uuid.New(), UserID: authenticated.User.ID, SessionID: authenticated.SessionID, AppID: appID,
		Purpose: purpose, Resource: resource, Method: method, Expires: expires.Unix()}
	payload, _ := json.Marshal(claims)
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, service.stepUpKey)
	_, _ = mac.Write([]byte(encoded))
	token := encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	err := service.repository.CreateStepUpTicket(ctx, login.StepUpTicket{ID: claims.ID,
		Principal: bearerPrincipal(authenticated, client.RequestID, ipString(client), client.UserAgent), AppID: appID,
		Purpose: purpose, Resource: resource, Method: method, TokenHash: hashString(token), ExpiresAt: expires})
	if err != nil {
		return StepUpResult{}, err
	}
	return StepUpResult{StepUpToken: token, ExpiresAt: expires}, nil
}

func (service *Service) consumeStepUp(ctx context.Context, raw string, authenticated iam.AuthenticatedContext, appID uuid.UUID, purpose, resource, requiredMethod string) error {
	parts := strings.Split(raw, ".")
	if len(parts) != 2 || len(service.stepUpKey) < 32 {
		return login.ErrStepUpInvalid
	}
	providedMAC, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return login.ErrStepUpInvalid
	}
	mac := hmac.New(sha256.New, service.stepUpKey)
	_, _ = mac.Write([]byte(parts[0]))
	if !hmac.Equal(providedMAC, mac.Sum(nil)) {
		return login.ErrStepUpInvalid
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	var claims stepUpClaims
	if err != nil || json.Unmarshal(payload, &claims) != nil || claims.UserID != authenticated.User.ID || claims.SessionID != authenticated.SessionID ||
		claims.AppID != appID || claims.Purpose != purpose || claims.Resource != resource || claims.Expires <= service.clock().UTC().Unix() ||
		requiredMethod != "" && claims.Method != requiredMethod {
		return login.ErrStepUpInvalid
	}
	return service.repository.ConsumeStepUpTicket(ctx, login.StepUpConsume{ID: claims.ID, UserID: claims.UserID,
		SessionID: claims.SessionID, AppID: claims.AppID, Purpose: purpose, Resource: resource,
		TokenHash: hashString(raw), RequiredMethod: requiredMethod})
}

func (service *Service) LoginMethods(ctx context.Context, token string, appID uuid.UUID) (login.LoginMethods, error) {
	authenticated, err := service.authenticateMobile(ctx, token, appID)
	if err != nil {
		return login.LoginMethods{}, err
	}
	return service.repository.LoginMethods(ctx, appID, authenticated.User.ID)
}

func (service *Service) OAuthAccounts(ctx context.Context, token string, appID uuid.UUID) ([]login.OAuthAccount, error) {
	authenticated, err := service.authenticateMobile(ctx, token, appID)
	if err != nil {
		return nil, err
	}
	return service.repository.ListOAuthAccounts(ctx, appID, authenticated.User.ID)
}

func (service *Service) DeleteOAuthAccount(ctx context.Context, token string, appID, accountID uuid.UUID, stepUpToken string, client iamapp.ClientMetadata) error {
	authenticated, err := service.authenticateMobile(ctx, token, appID)
	if err != nil {
		return err
	}
	accounts, err := service.repository.ListOAuthAccounts(ctx, appID, authenticated.User.ID)
	if err != nil {
		return err
	}
	var target *login.OAuthAccount
	for index := range accounts {
		if accounts[index].ID == accountID {
			target = &accounts[index]
			break
		}
	}
	if target == nil {
		return login.ErrNotFound
	}
	if !target.CanUnbind {
		return login.ErrLastLoginMethod
	}
	requiredMethod := ""
	if target.ProviderCode == login.ProviderApple {
		requiredMethod = "oauth:apple"
	}
	if err = service.consumeStepUp(ctx, stepUpToken, authenticated, appID, "oauth_unbind", accountID.String(), requiredMethod); err != nil {
		return login.ErrStepUpRequired
	}
	return service.repository.DeleteOAuthAccount(ctx, bearerPrincipal(authenticated, client.RequestID, ipString(client), client.UserAgent), appID, accountID)
}

func (service *Service) DeleteIdentifier(ctx context.Context, token string, appID uuid.UUID, identifierType, stepUpToken string, client iamapp.ClientMetadata) error {
	authenticated, err := service.authenticateMobile(ctx, token, appID)
	if err != nil {
		return err
	}
	if err = service.consumeStepUp(ctx, stepUpToken, authenticated, appID, "identifier_unbind", identifierType, ""); err != nil {
		return login.ErrStepUpRequired
	}
	return service.repository.DeleteIdentifier(ctx, bearerPrincipal(authenticated, client.RequestID, ipString(client), client.UserAgent), appID, identifierType)
}

// ConsumeAccountDeletionStepUp is wired into appmanagement. It is a fail-closed
// no-op only when the App user has no Apple binding; otherwise a fresh Apple
// reauth/revoke ticket bound to this session and account is mandatory.
func (service *Service) ConsumeAccountDeletionStepUp(ctx context.Context, token string, appID uuid.UUID, stepUpToken string) (uuid.UUID, error) {
	authenticated, err := service.authenticateMobile(ctx, token, appID)
	if err != nil {
		return uuid.Nil, err
	}
	accounts, err := service.repository.ListOAuthAccounts(ctx, appID, authenticated.User.ID)
	if err != nil {
		return uuid.Nil, err
	}
	for _, account := range accounts {
		if account.ProviderCode == login.ProviderApple {
			if err = service.consumeStepUp(ctx, stepUpToken, authenticated, appID, "account_delete", account.ID.String(), "oauth:apple"); err != nil {
				return uuid.Nil, login.ErrStepUpRequired
			}
			return account.ID, nil
		}
	}
	return uuid.Nil, nil
}
