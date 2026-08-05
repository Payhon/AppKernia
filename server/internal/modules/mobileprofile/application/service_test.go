package application

import (
	"context"
	"errors"
	"testing"

	profile "github.com/appkernia/appkernia/server/internal/modules/mobileprofile/domain"
	"github.com/google/uuid"
)

func TestUpdatePreferencesRejectsEmptyAndUnknownClientValuesBeforeAuthentication(t *testing.T) {
	service := &Service{}
	if _, err := service.UpdatePreferences(context.Background(), "", "request-1", nil, nil, nil); !errors.Is(err, ErrInvalidPreferences) {
		t.Fatalf("empty update error = %v, want ErrInvalidPreferences", err)
	}
	invalidLocale := "fr-FR"
	if _, err := service.UpdatePreferences(context.Background(), "", "request-1", &invalidLocale, nil, nil); !errors.Is(err, ErrInvalidPreferences) {
		t.Fatalf("locale validation error = %v, want ErrInvalidPreferences", err)
	}
	invalidAppearance := "neon"
	if _, err := service.UpdatePreferences(context.Background(), "", "request-1", nil, &invalidAppearance, nil); !errors.Is(err, ErrInvalidPreferences) {
		t.Fatalf("appearance validation error = %v, want ErrInvalidPreferences", err)
	}
	if _, err := service.UpdatePreferences(context.Background(), "", "request-1", nil, nil, map[string]bool{"sms": true}); !errors.Is(err, ErrInvalidPreferences) {
		t.Fatalf("notification validation error = %v, want ErrInvalidPreferences", err)
	}
}

func TestValidReleaseEnforcesComparableVersionsAndHTTPS(t *testing.T) {
	httpsURL := "https://example.test/download"
	base := profile.Release{Platform: "android", CurrentVersion: "2.1.0", MinimumVersion: "1.9.0", UpgradeURL: &httpsURL, Active: true, ReleaseNotes: map[string]string{"zh-CN": "更新说明", "en-US": "Release notes"}}
	if err := validRelease(base, false); err != nil {
		t.Fatalf("valid release rejected: %v", err)
	}
	invalid := []profile.Release{base, base, base, base}
	invalid[0].CurrentVersion = "2.1"
	invalid[1].MinimumVersion = "3.0.0"
	httpURL := "http://example.test/download"
	invalid[2].UpgradeURL = &httpURL
	invalid[3].ID = uuid.New()
	invalid[3].LockVersion = 0
	for index, release := range invalid {
		update := index == 3
		if err := validRelease(release, update); !errors.Is(err, ErrInvalidRelease) {
			t.Fatalf("case %d: error=%v, want ErrInvalidRelease", index, err)
		}
	}
}

func TestParseSemverComparesNumericSegments(t *testing.T) {
	left, ok := parseSemver("1.10.0")
	if !ok {
		t.Fatal("parse left")
	}
	right, ok := parseSemver("1.9.9")
	if !ok {
		t.Fatal("parse right")
	}
	if compareSemver(left, right) <= 0 {
		t.Fatal("numeric semver comparison is incorrect")
	}
}
