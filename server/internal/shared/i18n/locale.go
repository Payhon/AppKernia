package i18n

import (
	"strings"

	"golang.org/x/text/language"
)

type Locale string

const (
	LocaleZhCN Locale = "zh-CN"
	LocaleEnUS Locale = "en-US"
)

var (
	supportedTags = []language.Tag{language.SimplifiedChinese, language.AmericanEnglish}
	matcher       = language.NewMatcher(supportedTags)
)

func Supported() []Locale {
	return []Locale{LocaleZhCN, LocaleEnUS}
}

func Normalize(raw string) Locale {
	switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(raw), "_", "-")) {
	case "zh", "zh-cn", "zh-hans", "zh-sg":
		return LocaleZhCN
	case "en", "en-us":
		return LocaleEnUS
	}
	tag, err := language.Parse(raw)
	if err != nil {
		return LocaleZhCN
	}
	_, index, confidence := matcher.Match(tag)
	if confidence == language.No {
		return LocaleZhCN
	}
	return Supported()[index]
}

func ResolveAcceptLanguage(header string) Locale {
	if strings.TrimSpace(header) == "" {
		return LocaleZhCN
	}
	_, index := language.MatchStrings(matcher, header)
	if index < 0 || index >= len(supportedTags) {
		return LocaleZhCN
	}
	return Supported()[index]
}
