package application

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	iam "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	login "github.com/appkernia/appkernia/server/internal/modules/loginprovider/domain"
	"github.com/google/uuid"
)

type smsCaptchaAuthStub struct {
	verifyErr     error
	verified      int
	scope         iamapp.InteractiveCaptchaScope
	authenticated iam.AuthenticatedContext
	created       int
}

func (stub *smsCaptchaAuthStub) Authenticate(context.Context, string, string) (iam.AuthenticatedContext, error) {
	return stub.authenticated, nil
}
func (stub *smsCaptchaAuthStub) CreateInteractiveCaptcha(_ context.Context, scope iamapp.InteractiveCaptchaScope) (iamapp.LoginCaptcha, error) {
	stub.created++
	stub.scope = scope
	return iamapp.LoginCaptcha{ID: uuid.New(), Token: "opaque"}, nil
}
func (stub *smsCaptchaAuthStub) VerifyInteractiveCaptcha(_ context.Context, _ *iamapp.LoginCaptchaInput, scope iamapp.InteractiveCaptchaScope) error {
	stub.verified++
	stub.scope = scope
	return stub.verifyErr
}

type smsCaptchaRepositoryStub struct {
	login.Repository
	created int
	tenant  uuid.UUID
	target  login.IdentifierTarget
}

func (stub *smsCaptchaRepositoryStub) AppLoginSettings(context.Context, uuid.UUID) (login.AppLoginSettings, error) {
	if stub.tenant == uuid.Nil {
		stub.tenant = uuid.New()
	}
	return login.AppLoginSettings{TenantID: stub.tenant, OTPEnabled: true, EmailOTPEnabled: true, MobileOTPEnabled: true, Registration: true}, nil
}
func (stub *smsCaptchaRepositoryStub) ResolveApp(context.Context, uuid.UUID) (uuid.UUID, string, error) {
	return uuid.New(), "zh-CN", nil
}
func (stub *smsCaptchaRepositoryStub) FindOTPLoginUser(context.Context, uuid.UUID, string, string) (uuid.UUID, uuid.UUID, string, error) {
	return uuid.Nil, uuid.Nil, "", login.ErrNotFound
}
func (stub *smsCaptchaRepositoryStub) CreateOTPChallenge(_ context.Context, challenge login.OTPChallenge) (uuid.UUID, error) {
	stub.created++
	return challenge.ID, nil
}
func (stub *smsCaptchaRepositoryStub) IdentifierTarget(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (login.IdentifierTarget, error) {
	return stub.target, nil
}

func captchaTestContext(appID, tenantID, userID, sessionID uuid.UUID) iam.AuthenticatedContext {
	return iam.AuthenticatedContext{AuthContext: iam.AuthContext{User: iam.User{ID: userID}, Tenant: iam.Tenant{ID: tenantID}}, SessionID: sessionID, AppID: &appID}
}

func TestMobileSMSCaptchaIsConsumedBeforeOTPCreation(t *testing.T) {
	ip := netip.MustParseAddr("192.0.2.4")
	client := iamapp.ClientMetadata{IPAddress: &ip, DeviceKey: uuid.NewString()}
	appID := uuid.New()
	repository := &smsCaptchaRepositoryStub{}
	auth := &smsCaptchaAuthStub{verifyErr: iamapp.ErrCaptchaRequired}
	service := NewService(auth, repository, nil, nil, "", nil)
	if _, err := service.SendLoginCode(context.Background(), appID, "mobile", "+15551234567", "zh-CN", client, nil); !errors.Is(err, login.ErrCaptchaRequired) || repository.created != 0 {
		t.Fatalf("missing CAPTCHA: err=%v otp creations=%d", err, repository.created)
	}
	auth.verifyErr = iamapp.ErrCaptchaInvalid
	if _, err := service.SendLoginCode(context.Background(), appID, "mobile", "+15551234567", "zh-CN", client, &iamapp.LoginCaptchaInput{ID: uuid.New(), Token: "tampered"}); !errors.Is(err, login.ErrCaptchaInvalid) || repository.created != 0 {
		t.Fatalf("invalid CAPTCHA: err=%v otp creations=%d", err, repository.created)
	}
	auth.verifyErr = nil
	if _, err := service.SendLoginCode(context.Background(), appID, "mobile", "+15551234567", "zh-CN", client, &iamapp.LoginCaptchaInput{ID: uuid.New(), Token: "opaque"}); err != nil || repository.created != 1 {
		t.Fatalf("valid CAPTCHA must create one OTP: err=%v creations=%d", err, repository.created)
	}
	if auth.verified != 3 || auth.scope.Audience != "ak-mobile" || auth.scope.AppID != appID || auth.scope.Scene != "login" || auth.scope.Target != "+15551234567" || auth.scope.Client.DeviceKey != client.DeviceKey {
		t.Fatalf("unexpected CAPTCHA binding: verified=%d scope=%+v", auth.verified, auth.scope)
	}
	if _, err := service.SendLoginCode(context.Background(), appID, "email", "person@example.test", "zh-CN", client, nil); err != nil || auth.verified != 3 || repository.created != 2 {
		t.Fatalf("email OTP must bypass CAPTCHA: err=%v verified=%d creations=%d", err, auth.verified, repository.created)
	}
}

func TestCreateMobileSMSCaptchaBindsEveryScene(t *testing.T) {
	appID, tenantID, userID, sessionID, identifierID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	repository := &smsCaptchaRepositoryStub{tenant: tenantID, target: login.IdentifierTarget{
		ID: identifierID, AppID: appID, TenantID: tenantID, UserID: userID, IdentifierType: "mobile", NormalizedValue: "+15551234567",
	}}
	auth := &smsCaptchaAuthStub{authenticated: captchaTestContext(appID, tenantID, userID, sessionID)}
	service := NewService(auth, repository, nil, nil, "", nil)
	ip := netip.MustParseAddr("192.0.2.8")
	client := iamapp.ClientMetadata{IPAddress: &ip, DeviceKey: uuid.NewString()}
	tests := []struct {
		name  string
		input SMSCaptchaInput
		auth  bool
	}{
		{name: "login", input: SMSCaptchaInput{Scene: "login", Mobile: "+15551234567"}},
		{name: "registration", input: SMSCaptchaInput{Scene: "registration", Mobile: "+15551234567"}},
		{name: "password reset", input: SMSCaptchaInput{Scene: "password_reset", Mobile: "+15551234567"}},
		{name: "identifier verify", input: SMSCaptchaInput{Scene: "identifier_verify", Mobile: "+15551234567"}, auth: true},
		{name: "step up", input: SMSCaptchaInput{Scene: "step_up", IdentifierID: identifierID, Purpose: "identifier_change", Resource: "mobile"}, auth: true},
	}
	for _, item := range tests {
		t.Run(item.name, func(t *testing.T) {
			token := ""
			if item.auth {
				token = "session"
			}
			if _, err := service.CreateSMSCaptcha(context.Background(), token, appID, item.input, client); err != nil {
				t.Fatalf("create CAPTCHA: %v", err)
			}
			if auth.scope.Scene != item.input.Scene || auth.scope.Target != "+15551234567" || auth.scope.AppID != appID {
				t.Fatalf("unexpected scope: %+v", auth.scope)
			}
			if item.auth && (auth.scope.UserID != userID || auth.scope.SessionID != sessionID) {
				t.Fatalf("authenticated scope missing identity: %+v", auth.scope)
			}
		})
	}
	if auth.created != len(tests) {
		t.Fatalf("created challenges=%d", auth.created)
	}
}

func TestEveryMobileSMSSenderRejectsBeforeOTPWithoutCaptcha(t *testing.T) {
	appID, tenantID, userID, sessionID, identifierID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	repository := &smsCaptchaRepositoryStub{tenant: tenantID, target: login.IdentifierTarget{
		ID: identifierID, AppID: appID, TenantID: tenantID, UserID: userID, IdentifierType: "mobile", NormalizedValue: "+15551234567", Locale: "zh-CN",
	}}
	auth := &smsCaptchaAuthStub{verifyErr: iamapp.ErrCaptchaRequired, authenticated: captchaTestContext(appID, tenantID, userID, sessionID)}
	service := NewService(auth, repository, nil, nil, "", nil)
	ip := netip.MustParseAddr("192.0.2.9")
	client := iamapp.ClientMetadata{IPAddress: &ip, DeviceKey: uuid.NewString()}
	calls := []func() error{
		func() error {
			_, err := service.SendLoginCode(context.Background(), appID, "mobile", "+15551234567", "zh-CN", client, nil)
			return err
		},
		func() error {
			_, err := service.SendRegistrationCode(context.Background(), appID, "mobile", "+15551234567", "zh-CN", client, nil)
			return err
		},
		func() error {
			_, err := service.SendPasswordResetCode(context.Background(), appID, "mobile", "+15551234567", "zh-CN", client, nil)
			return err
		},
		func() error {
			_, err := service.SendIdentifierCode(context.Background(), "session", appID, "mobile", IdentifierCodeInput{Identifier: "+15551234567"}, client)
			return err
		},
		func() error {
			_, err := service.SendStepUpCode(context.Background(), "session", appID, StepUpCodeInput{IdentifierID: identifierID, Purpose: "identifier_change", Resource: "mobile"}, client)
			return err
		},
	}
	for index, call := range calls {
		if err := call(); !errors.Is(err, login.ErrCaptchaRequired) {
			t.Fatalf("sender %d must require CAPTCHA, got %v", index, err)
		}
	}
	if repository.created != 0 || auth.verified != len(calls) {
		t.Fatalf("OTP must not be created: otp=%d captcha checks=%d", repository.created, auth.verified)
	}
}
