package application

import (
	"testing"

	"github.com/google/uuid"
)

func TestStartupInputRequiresBothLocalesAndCompleteSlides(t *testing.T) {
	fileA, fileB := uuid.New(), uuid.New()
	valid := StartupInput{
		Translations: map[string]StartupTranslation{
			"zh-CN": {DisplayName: " 应用 ", Subtitle: " 副标题 "},
			"en-US": {DisplayName: " App ", Subtitle: " Subtitle "},
		},
		DraftSlides: []StartupSlide{{Assets: map[string]StartupSlideAsset{
			"zh-CN": {FileID: fileA, AccessibilityLabel: " 中文介绍 "},
			"en-US": {FileID: fileB, AccessibilityLabel: " English intro "},
		}}},
	}
	normalized := normalizeStartupInput(valid)
	if err := validStartupInput(normalized); err != nil {
		t.Fatalf("valid startup rejected: %v", err)
	}
	if normalized.Translations["zh-CN"].DisplayName != "应用" || normalized.DraftSlides[0].Assets["en-US"].AccessibilityLabel != "English intro" {
		t.Fatalf("startup input was not normalized: %#v", normalized)
	}
	missingLocale := normalized
	missingLocale.Translations = map[string]StartupTranslation{"zh-CN": normalized.Translations["zh-CN"]}
	if err := validStartupInput(missingLocale); err == nil {
		t.Fatal("missing English startup translation accepted")
	}
	incompleteSlide := normalized
	incompleteSlide.DraftSlides = []StartupSlide{{Assets: map[string]StartupSlideAsset{"zh-CN": normalized.DraftSlides[0].Assets["zh-CN"]}}}
	if err := validStartupInput(incompleteSlide); err == nil {
		t.Fatal("incomplete localized slide accepted")
	}
}

func TestStartupInputLimitsSlides(t *testing.T) {
	input := StartupInput{Translations: map[string]StartupTranslation{
		"zh-CN": {DisplayName: "应用"}, "en-US": {DisplayName: "App"},
	}}
	for index := 0; index < 11; index++ {
		input.DraftSlides = append(input.DraftSlides, StartupSlide{Assets: map[string]StartupSlideAsset{
			"zh-CN": {FileID: uuid.New(), AccessibilityLabel: "中文"},
			"en-US": {FileID: uuid.New(), AccessibilityLabel: "English"},
		}})
	}
	if err := validStartupInput(normalizeStartupInput(input)); err == nil {
		t.Fatal("more than ten onboarding slides accepted")
	}
}
