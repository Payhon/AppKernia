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

func TestValidReleaseEnforcesNativeAndWGTTargetRules(t *testing.T) {
	httpsURL := "https://example.test/download"
	base := profile.Release{PackageType: "native_app", Platforms: []string{"android"}, Version: "2.0.0", ExternalURL: &httpsURL, Active: true, Titles: map[string]string{"zh-CN": "更新", "en-US": "Update"}, Contents: map[string]string{"zh-CN": "说明", "en-US": "Notes"}}
	if err := validRelease(base, false); err != nil {
		t.Fatalf("valid native release rejected: %v", err)
	}
	invalidNative := base
	invalidNative.Platforms = []string{"android", "ios"}
	if !errors.Is(validRelease(invalidNative, false), ErrInvalidRelease) {
		t.Fatal("multi-platform native release accepted")
	}
	wgt := base
	wgt.PackageType = "wgt"
	wgt.Platforms = []string{"android", "ios", "harmony"}
	minimum := "1.0.0"
	wgt.MinimumNativeVersion = &minimum
	if err := validRelease(wgt, false); err != nil {
		t.Fatalf("valid multi-platform WGT release rejected: %v", err)
	}
	wgt.MinimumNativeVersion = nil
	if !errors.Is(validRelease(wgt, false), ErrInvalidRelease) {
		t.Fatal("WGT without minimum native version accepted")
	}
}

func TestValidReleaseCapabilitiesEnforcesAppAndPlatformMatrix(t *testing.T) {
	fileID := uuid.New()
	cases := []struct {
		name    string
		appType string
		release profile.Release
		want    error
	}{
		{name: "uni app x native external", appType: "uni_app_x", release: profile.Release{PackageType: "native_app", Platforms: []string{"ios"}}},
		{name: "uni app x wgt", appType: "uni_app_x", release: profile.Release{PackageType: "wgt", Platforms: []string{"android"}}, want: profile.ErrReleasePackageTypeUnsupported},
		{name: "android internal apk", appType: "uni_app_x", release: profile.Release{PackageType: "native_app", Platforms: []string{"android"}, PackageFileID: &fileID}},
		{name: "ios internal package", appType: "uni_app_x", release: profile.Release{PackageType: "native_app", Platforms: []string{"ios"}, PackageFileID: &fileID}, want: profile.ErrReleaseDeliveryModeUnsupported},
		{name: "harmony internal package", appType: "uni_app", release: profile.Release{PackageType: "native_app", Platforms: []string{"harmony"}, PackageFileID: &fileID}, want: profile.ErrReleaseDeliveryModeUnsupported},
		{name: "classic uni app wgt", appType: "uni_app", release: profile.Release{PackageType: "wgt", Platforms: []string{"android", "ios"}, PackageFileID: &fileID}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			err := validReleaseCapabilities(test.appType, test.release)
			if !errors.Is(err, test.want) || test.want == nil && err != nil {
				t.Fatalf("error=%v want=%v", err, test.want)
			}
		})
	}
}

func TestDraftMayBeIncompleteButImmediatePublishRequiresBilingualContent(t *testing.T) {
	draft := profile.Release{PackageType: "native_app", Version: "1.2.3", Platforms: []string{"android"}}
	if err := validRelease(draft, false); err != nil {
		t.Fatalf("incomplete draft rejected: %v", err)
	}
	draft.Active = true
	if !errors.Is(validRelease(draft, false), ErrInvalidRelease) {
		t.Fatal("immediate publish accepted without bilingual content and source")
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

func TestPackageArchiveHeaderRejectsTypeSpoofing(t *testing.T) {
	for _, header := range [][]byte{{'P', 'K', 3, 4}, {'P', 'K', 5, 6}, {'P', 'K', 7, 8}} {
		if !isZipArchiveHeader(header) {
			t.Fatalf("valid ZIP signature rejected: %v", header)
		}
	}
	for _, header := range [][]byte{nil, {'P', 'K'}, {'N', 'O', 'P', 'E'}, {'P', 'K', 1, 2}} {
		if isZipArchiveHeader(header) {
			t.Fatalf("non-ZIP signature accepted: %v", header)
		}
	}
}

func TestPackageDownloadSignatureBindsIdentifiersAndExpiry(t *testing.T) {
	key := []byte("test-signing-key-with-sufficient-entropy")
	appID, releaseID, fileID := uuid.New(), uuid.New(), uuid.New()
	now := int64(1_800_000_000)
	expires := now + 300
	signature := signPackageDownload(key, appID, releaseID, fileID, expires)
	if !validPackageDownloadSignature(key, appID, releaseID, fileID, expires, now, signature) {
		t.Fatal("valid package signature rejected")
	}
	if validPackageDownloadSignature(key, uuid.New(), releaseID, fileID, expires, now, signature) ||
		validPackageDownloadSignature(key, appID, releaseID, fileID, expires, expires+1, signature) ||
		validPackageDownloadSignature(key, appID, releaseID, fileID, now+601, now, signature) ||
		validPackageDownloadSignature(key, appID, releaseID, fileID, expires, now, signature+"x") {
		t.Fatal("tampered or expired package signature accepted")
	}
}

func TestPublicWebPackageSignatureCannotBypassWebGateThroughMobileRoute(t *testing.T) {
	key := []byte("test-only-package-key")
	app, release, file := uuid.New(), uuid.New(), uuid.New()
	now := int64(1000)
	sig := signPackageDownload(publicWebSigningKey(key), app, release, file, now+300)
	if validPackageDownloadSignature(key, app, release, file, now+300, now, sig) {
		t.Fatal("H5 signature accepted by legacy mobile endpoint")
	}
	if !validPackageDownloadSignature(publicWebSigningKey(key), app, release, file, now+300, now, sig) {
		t.Fatal("H5 signature rejected")
	}
	if validPackageDownloadSignature(publicWebSigningKey(key), app, release, file, now+300, now+301, sig) {
		t.Fatal("expired H5 signature accepted")
	}
}
