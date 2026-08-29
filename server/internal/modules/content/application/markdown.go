package application

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"

	content "github.com/appkernia/appkernia/server/internal/modules/content/domain"
	"github.com/google/uuid"
)

var (
	markdownHTMLPattern     = regexp.MustCompile(`(?i)<\s*/?\s*[a-z][^>]*>`)
	markdownProtocolPattern = regexp.MustCompile(`(?i)(?:javascript|data|vbscript):`)
	markdownImagePattern    = regexp.MustCompile(`!\[[^\]]*\]\(([^)\s]+)`)
	markdownLinkPattern     = regexp.MustCompile(`\[[^\]]+\]\(([^)\s]+)`)
)

func normalizeArticleBodies(x *content.Article) {
	for locale, translation := range x.Translations {
		if translation.BodyFormat != "blocks" {
			continue
		}
		if markdown, ok := legacyBodyToMarkdown(translation.Body); ok {
			translation.BodyFormat = "markdown"
			translation.Body = json.RawMessage(fmt.Sprintf("%q", markdown))
			x.Translations[locale] = translation
		}
	}
}

func normalizePublicArticleBody(x *content.PublicArticle) {
	if x.BodyFormat != "blocks" {
		return
	}
	if markdown, ok := legacyBodyToMarkdown(x.Body); ok {
		x.BodyFormat = "markdown"
		x.Body = json.RawMessage(fmt.Sprintf("%q", markdown))
	}
}

func legacyBodyToMarkdown(raw json.RawMessage) (string, bool) {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return "", false
	}
	if encoded, ok := value.(string); ok {
		if json.Unmarshal([]byte(encoded), &value) != nil {
			return "", false
		}
	}
	result := renderLegacyNode(value, 0)
	return strings.TrimSpace(result), true
}

func renderLegacyNode(value any, depth int) string {
	if depth > 20 {
		slog.Warn("content legacy markdown conversion depth exceeded")
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	if list, ok := value.([]any); ok {
		parts := make([]string, 0, len(list))
		for _, item := range list {
			parts = append(parts, renderLegacyNode(item, depth+1))
		}
		return strings.Join(parts, "")
	}
	node, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	typeName, _ := node["type"].(string)
	children := renderLegacyNode(node["content"], depth+1)
	attrs, _ := node["attrs"].(map[string]any)
	switch typeName {
	case "doc", "listItem":
		return children
	case "text":
		return renderLegacyText(node)
	case "paragraph":
		return children + "\n\n"
	case "heading":
		level := intValue(attrs["level"], 2)
		if level < 1 || level > 6 {
			level = 2
		}
		return strings.Repeat("#", level) + " " + strings.TrimSpace(children) + "\n\n"
	case "blockquote":
		lines := strings.Split(strings.TrimSpace(children), "\n")
		for i, line := range lines {
			lines[i] = "> " + line
		}
		return strings.Join(lines, "\n") + "\n\n"
	case "bulletList":
		return renderLegacyList(node, "- ", depth) + "\n"
	case "orderedList":
		return renderLegacyList(node, "", depth) + "\n"
	case "codeBlock":
		return "```\n" + strings.TrimSpace(children) + "\n```\n\n"
	case "horizontalRule":
		return "---\n\n"
	case "hardBreak":
		return "\n"
	case "image":
		fileID, _ := attrs["file_id"].(string)
		if _, err := uuid.Parse(fileID); err != nil {
			slog.Warn("content legacy image node discarded", "reason", "missing file reference")
			return ""
		}
		alt, _ := attrs["alt"].(string)
		return fmt.Sprintf("![%s](/api/v1/public/content/assets/%s)", strings.TrimSpace(alt), fileID)
	default:
		if typeName != "" {
			slog.Warn("content legacy node discarded", "node_type", typeName)
		}
		return children
	}
}

func renderLegacyList(node map[string]any, marker string, depth int) string {
	items, _ := node["content"].([]any)
	lines := make([]string, 0, len(items))
	for index, item := range items {
		prefix := marker
		if marker == "" {
			prefix = fmt.Sprintf("%d. ", index+1)
		}
		lines = append(lines, prefix+strings.TrimSpace(renderLegacyNode(item, depth+1)))
	}
	return strings.Join(lines, "\n")
}

func renderLegacyText(node map[string]any) string {
	text, _ := node["text"].(string)
	marks, _ := node["marks"].([]any)
	for _, rawMark := range marks {
		mark, _ := rawMark.(map[string]any)
		kind, _ := mark["type"].(string)
		switch kind {
		case "bold":
			text = "**" + text + "**"
		case "italic":
			text = "*" + text + "*"
		case "code":
			text = "`" + text + "`"
		case "link":
			attrs, _ := mark["attrs"].(map[string]any)
			href, _ := attrs["href"].(string)
			if safeMarkdownLink(href) {
				text = "[" + text + "](" + href + ")"
			}
		}
	}
	return text
}

func validMarkdown(value string) bool {
	if markdownHTMLPattern.MatchString(value) || markdownProtocolPattern.MatchString(value) {
		return false
	}
	for _, match := range markdownImagePattern.FindAllStringSubmatch(value, -1) {
		if !safeMarkdownImage(match[1]) {
			return false
		}
	}
	for _, match := range markdownLinkPattern.FindAllStringSubmatch(value, -1) {
		if !safeMarkdownLink(match[1]) {
			return false
		}
	}
	return true
}

func safeMarkdownImage(value string) bool {
	if strings.HasPrefix(value, "/api/v1/public/content/assets/") {
		return true
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.User == nil
}

func safeMarkdownLink(value string) bool {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, "#") {
		return true
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.User == nil && ((parsed.Scheme == "https" && parsed.Host != "") || (parsed.Scheme == "mailto" && parsed.Opaque != ""))
}

func intValue(value any, fallback int) int {
	if number, ok := value.(float64); ok {
		return int(number)
	}
	return fallback
}
