package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	profile "github.com/appkernia/appkernia/server/internal/modules/mobileprofile/application"
	profiledomain "github.com/appkernia/appkernia/server/internal/modules/mobileprofile/domain"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

func TestBearerHeaderIsStrict(t *testing.T) {
	for _, test := range []struct{ header, want string }{{"Bearer mobile-token", "mobile-token"}, {"bearer mobile-token", "mobile-token"}, {"Bearer", ""}, {"Basic mobile-token", ""}, {"Bearer one two", ""}, {"mobile-token", ""}} {
		if got := bearerHeader(test.header); got != test.want {
			t.Fatalf("bearerHeader(%q)=%q want %q", test.header, got, test.want)
		}
	}
}

func TestDecodeRejectsUnknownFieldsAndMultipleValues(t *testing.T) {
	for _, body := range []string{`{"known":true,"unknown":false}`, `{"known":true} {"known":false}`} {
		req := &ghttp.Request{Request: httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(body))}
		var target struct {
			Known bool `json:"known"`
		}
		if err := decode(req, &target); err == nil {
			t.Fatalf("decode accepted invalid payload %q", body)
		}
	}
}

func TestNotificationPreferencesDTORejectsLocaleAndAppearance(t *testing.T) {
	for _, body := range []string{`{"locale":"en-US","notification_preferences":{"push":true}}`, `{"appearance":"dark","notification_preferences":{"push":true}}`} {
		req := &ghttp.Request{Request: httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(body))}
		var target notificationPreferencesRequest
		if err := decode(req, &target); err == nil {
			t.Fatalf("notification DTO accepted unrelated field: %s", body)
		}
	}
	req := &ghttp.Request{Request: httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(`{"notification_preferences":{"push":true}}`))}
	var target notificationPreferencesRequest
	if err := decode(req, &target); err != nil {
		t.Fatalf("valid notification DTO rejected: %v", err)
	}
}

func TestPublicReleaseResponseIsLocalizedAndDoesNotExposeAdminFields(t *testing.T) {
	storeID := uuid.New()
	value := publicReleaseResponse(profiledomain.Release{Platform: "ios", CurrentVersion: "2.0.0", MinimumVersion: "1.5.0", ReleaseNotes: map[string]string{"zh-CN": "中文说明", "en-US": "English notes"}, StoreList: []profiledomain.StoreListing{{ID: storeID, Name: "App Store", Scheme: "itms-apps://apps.apple.com/app/id1", Priority: 100}}, Active: true, LockVersion: 9}, i18n.LocaleEnUS)
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	encoded := string(raw)
	if !strings.Contains(encoded, "English notes") || !strings.Contains(encoded, "external_link") || !strings.Contains(encoded, storeID.String()) || strings.Contains(encoded, "lock_version") || strings.Contains(encoded, "active") {
		t.Fatalf("unsafe public response: %s", encoded)
	}
}

func TestPublicReleaseErrorMapsValidationAndMissingPolicy(t *testing.T) {
	status, code, _ := publicReleaseError(profile.ErrInvalidRelease)
	if status != http.StatusUnprocessableEntity || code != "VALIDATION.FAILED" {
		t.Fatalf("validation mapping=%d/%s", status, code)
	}
	status, code, _ = publicReleaseError(profiledomain.ErrReleaseNotFound)
	if status != http.StatusServiceUnavailable || code != "APP.RELEASE.UNAVAILABLE" {
		t.Fatalf("missing policy mapping=%d/%s", status, code)
	}
	if status, _, _ := publicReleaseError(errors.New("storage unavailable")); status != http.StatusServiceUnavailable {
		t.Fatalf("storage mapping=%d", status)
	}
	status, code, _ = publicReleaseError(profiledomain.ErrReleasePackageTypeUnsupported)
	if status != http.StatusUnprocessableEntity || code != "SYS.MOBILE_RELEASE.UNSUPPORTED_PACKAGE_TYPE" {
		t.Fatalf("package capability mapping=%d/%s", status, code)
	}
	status, code, _ = publicReleaseError(profiledomain.ErrReleaseDeliveryModeUnsupported)
	if status != http.StatusUnprocessableEntity || code != "SYS.MOBILE_RELEASE.UNSUPPORTED_DELIVERY_MODE" {
		t.Fatalf("delivery capability mapping=%d/%s", status, code)
	}
}
