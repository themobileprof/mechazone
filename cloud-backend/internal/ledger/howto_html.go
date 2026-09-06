package ledger

import (
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/net/html"
)

func howtoPolicy() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements("p", "br", "div", "span", "h2", "h3", "h4", "ul", "ol", "li", "strong", "b", "em", "i", "u", "blockquote", "figure", "figcaption", "hr", "img", "a")
	p.AllowAttrs("alt").OnElements("img")
	p.AllowAttrs("src").OnElements("img")
	p.AllowAttrs("href").OnElements("a")
	p.AllowRelativeURLs(true)
	p.AllowURLSchemes("https", "http")
	p.RequireNoFollowOnLinks(true)
	return p
}

func allowedHowToSrc(src string) bool {
	u := strings.TrimSpace(src)
	if u == "" || strings.Contains(u, "..") || strings.Contains(u, "://") || strings.HasPrefix(u, "//") {
		return false
	}
	if strings.ContainsAny(u, " \t\n<>\"'") {
		return false
	}
	return strings.HasPrefix(u, "/howto/")
}

func dropDisallowedImages(raw string) string {
	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return raw
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "img" {
			ok := false
			for _, a := range n.Attr {
				if a.Key == "src" && allowedHowToSrc(a.Val) {
					ok = true
					break
				}
			}
			if !ok && n.Parent != nil {
				n.Parent.RemoveChild(n)
				return
			}
		}
		for c := n.FirstChild; c != nil; {
			next := c.NextSibling
			walk(c)
			c = next
		}
	}
	walk(doc)
	var b strings.Builder
	if err := html.Render(&b, doc); err != nil {
		return raw
	}
	out := b.String()
	out = strings.TrimPrefix(out, "<html><head></head><body>")
	out = strings.TrimSuffix(out, "</body></html>")
	return out
}

// SanitizeHowToHTML keeps shop-skill markup. Images must be files under /howto/. No scripts.
func SanitizeHowToHTML(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if len(raw) > 100_000 {
		raw = raw[:100_000]
	}
	clean := howtoPolicy().Sanitize(raw)
	return strings.TrimSpace(dropDisallowedImages(clean))
}

func normalizeMatchWords(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, w := range in {
		w = clip(strings.TrimSpace(strings.ToLower(w)), 40)
		if w == "" || len([]rune(w)) < 2 {
			continue
		}
		if _, ok := actionStop[w]; ok {
			continue
		}
		if _, ok := seen[w]; ok {
			continue
		}
		seen[w] = struct{}{}
		out = append(out, w)
		if len(out) >= 24 {
			break
		}
	}
	return out
}
