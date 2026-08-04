package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed locales/*.json
var localeFiles embed.FS

type Catalog struct {
	messages map[Locale]map[string]string
}

func LoadCatalog() (*Catalog, error) {
	messages := make(map[Locale]map[string]string, len(Supported()))
	for _, locale := range Supported() {
		raw, err := localeFiles.ReadFile("locales/" + string(locale) + ".json")
		if err != nil {
			return nil, fmt.Errorf("read locale %s: %w", locale, err)
		}
		var values map[string]string
		if err := json.Unmarshal(raw, &values); err != nil {
			return nil, fmt.Errorf("decode locale %s: %w", locale, err)
		}
		messages[locale] = values
	}
	if err := validateParity(messages[LocaleZhCN], messages[LocaleEnUS]); err != nil {
		return nil, err
	}
	return &Catalog{messages: messages}, nil
}

func (c *Catalog) Translate(locale Locale, key string, params map[string]string) string {
	value, ok := c.messages[locale][key]
	if !ok {
		value, ok = c.messages[LocaleZhCN][key]
	}
	if !ok {
		return key
	}
	for name, replacement := range params {
		value = strings.ReplaceAll(value, "{"+name+"}", replacement)
	}
	return value
}

func validateParity(left, right map[string]string) error {
	for key := range left {
		if _, ok := right[key]; !ok {
			return fmt.Errorf("en-US catalog missing key %s", key)
		}
	}
	for key := range right {
		if _, ok := left[key]; !ok {
			return fmt.Errorf("zh-CN catalog missing key %s", key)
		}
	}
	return nil
}
