package richtext

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"net/url"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var tokenClass = regexp.MustCompile(`^language-[a-zA-Z0-9_-]+$`)

// RenderBody is the sole conversion to template.HTML. Raw template code is never accepted.
func RenderBody(raw json.RawMessage, format string, appID uuid.UUID) (template.HTML, error) {
	if len(raw) > 1<<20 {
		return "", fmt.Errorf("body too large")
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", err
	}
	var rendered string
	if format == "markdown" {
		md, ok := value.(string)
		if !ok {
			return "", fmt.Errorf("markdown must be a string")
		}
		var buf bytes.Buffer
		if err := goldmark.New(goldmark.WithExtensions(extension.GFM)).Convert([]byte(md), &buf); err != nil {
			return "", err
		}
		rendered = buf.String()
	} else if format == "blocks" {
		if encoded, ok := value.(string); ok {
			if err := json.Unmarshal([]byte(encoded), &value); err != nil {
				return "", err
			}
		}
		rendered = renderBlock(value, 0)
	} else {
		return "", fmt.Errorf("unsupported body format")
	}
	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("class").Matching(tokenClass).OnElements("code")
	safe := policy.Sanitize(rendered)
	nodes, err := xhtml.ParseFragment(strings.NewReader(safe), &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div})
	if err != nil {
		return "", err
	}
	var visit func(*xhtml.Node)
	visit = func(n *xhtml.Node) {
		attrs := n.Attr[:0]
		for _, a := range n.Attr {
			if strings.HasPrefix(a.Key, "hx-") || strings.HasPrefix(a.Key, "data-hx-") || strings.HasPrefix(a.Key, "on") {
				continue
			}
			if n.Data == "img" && a.Key == "src" {
				a.Val = ImageURL(a.Val, appID)
				if a.Val == "" {
					continue
				}
			}
			attrs = append(attrs, a)
		}
		n.Attr = attrs
		if n.Data == "img" {
			n.Attr = append(n.Attr, xhtml.Attribute{Key: "loading", Val: "lazy"}, xhtml.Attribute{Key: "decoding", Val: "async"})
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	var out bytes.Buffer
	for _, n := range nodes {
		visit(n)
		if err = xhtml.Render(&out, n); err != nil {
			return "", err
		}
	}
	return template.HTML(out.String()), nil // #nosec G203 -- sanitized allowlist; no other raw HTML conversion.
}
func ImageURL(raw string, appID uuid.UUID) string {
	u, e := url.Parse(raw)
	if e != nil || u.IsAbs() || u.Host != "" || u.RawQuery != "" {
		return ""
	}
	prefixes := []string{"/api/v1/public/content/assets/", "/s/assets/" + appID.String() + "/", "/h5/apps/" + appID.String() + "/assets/"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(u.Path, prefix) {
			id, e := uuid.Parse(strings.TrimPrefix(u.Path, prefix))
			if e == nil {
				return "/h5/apps/" + appID.String() + "/assets/" + id.String()
			}
		}
	}
	return ""
}
func renderBlock(v any, depth int) string {
	if depth > 20 {
		return ""
	}
	if text, ok := v.(string); ok {
		return html.EscapeString(text)
	}
	if list, ok := v.([]any); ok {
		var out strings.Builder
		for _, x := range list {
			out.WriteString(renderBlock(x, depth+1))
		}
		return out.String()
	}
	node, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	kind, _ := node["type"].(string)
	attrs, _ := node["attrs"].(map[string]any)
	children := renderBlock(node["content"], depth+1)
	switch kind {
	case "doc":
		return children
	case "text":
		text, _ := node["text"].(string)
		out := html.EscapeString(text)
		marks, _ := node["marks"].([]any)
		for _, m := range marks {
			mark, _ := m.(map[string]any)
			switch mark["type"] {
			case "bold":
				out = "<strong>" + out + "</strong>"
			case "italic":
				out = "<em>" + out + "</em>"
			case "code":
				out = "<code>" + out + "</code>"
			case "link":
				a, _ := mark["attrs"].(map[string]any)
				href, _ := a["href"].(string)
				out = "<a href=\"" + html.EscapeString(href) + "\">" + out + "</a>"
			}
		}
		return out
	case "paragraph":
		return "<p>" + children + "</p>"
	case "heading":
		level, _ := attrs["level"].(float64)
		if level < 2 || level > 6 {
			level = 2
		}
		tag := fmt.Sprintf("h%d", int(level))
		return "<" + tag + ">" + children + "</" + tag + ">"
	case "blockquote":
		return "<blockquote>" + children + "</blockquote>"
	case "bulletList":
		return "<ul>" + children + "</ul>"
	case "orderedList":
		return "<ol>" + children + "</ol>"
	case "listItem":
		return "<li>" + children + "</li>"
	case "codeBlock":
		return "<pre><code>" + children + "</code></pre>"
	case "hardBreak":
		return "<br>"
	case "horizontalRule":
		return "<hr>"
	case "image":
		id, _ := attrs["file_id"].(string)
		alt, _ := attrs["alt"].(string)
		if _, e := uuid.Parse(id); e != nil {
			return ""
		}
		return "<img src=\"/api/v1/public/content/assets/" + id + "\" alt=\"" + html.EscapeString(alt) + "\">"
	case "table":
		return "<table>" + children + "</table>"
	case "tableRow":
		return "<tr>" + children + "</tr>"
	case "tableCell":
		return "<td>" + children + "</td>"
	case "tableHeader":
		return "<th>" + children + "</th>"
	default:
		return children
	}
}

// ReferencesImage uses the same sanitizer as HTML rendering; quoted text or raw HTML is not a business image reference.
func ReferencesImage(raw json.RawMessage, format string, appID, fileID uuid.UUID) bool {
	body, err := RenderBody(raw, format, appID)
	if err != nil {
		return false
	}
	nodes, err := xhtml.ParseFragment(strings.NewReader(string(body)), &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div})
	if err != nil {
		return false
	}
	target := "/h5/apps/" + appID.String() + "/assets/" + fileID.String()
	var visit func(*xhtml.Node) bool
	visit = func(n *xhtml.Node) bool {
		if n.Type == xhtml.ElementNode && n.Data == "img" {
			for _, a := range n.Attr {
				if a.Key == "src" && a.Val == target {
					return true
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			if visit(child) {
				return true
			}
		}
		return false
	}
	for _, n := range nodes {
		if visit(n) {
			return true
		}
	}
	return false
}
