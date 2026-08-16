package ai

import (
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	tagRe     = regexp.MustCompile(`(?s)<script[^>]*>.*?</script>|<style[^>]*>.*?</style>|<[^>]+>`)
	imgTagRe  = regexp.MustCompile(`(?i)<img\b[^>]*>`)
	imgSrcRe  = regexp.MustCompile(`(?i)\bsrc\s*=\s*["']([^"']+)["']`)
	imgAltRe  = regexp.MustCompile(`(?i)\balt\s*=\s*["']([^"']+)["']`)
	spaceRe   = regexp.MustCompile(`[ \t\r\f]+`)
	skipParts = []string{
		"/css/", "/js/", "/scripts/", "/styles/", "/menu/", "/howto/",
		"/_ocr", "/_img_captions", "/.git/", "/.venv", "/node_modules/",
		"/title/",
	}
	chromeImgs = []string{"print.png", "logo.png", "guardrail.png", "next_right.png", "next_down.png", "fin_right.png", "fin_down.png", "gas.png", "repair.png"}
)

type HTMLPage struct {
	Rel   string
	Title string
	Text  string
	Imgs  []HTMLImage
}

type HTMLImage struct {
	Abs     string
	RelKey  string
	Alt     string
	Caption string
}

func shouldSkipHTML(rel string) bool {
	low := strings.ToLower(strings.ReplaceAll(rel, "\\", "/"))
	for _, p := range skipParts {
		if strings.Contains(low, p) {
			return true
		}
	}
	base := filepath.Base(low)
	if strings.HasPrefix(base, "index") {
		return true
	}
	if !strings.Contains(low, "/contents/") && !strings.Contains(low, "/html/contents/") {
		return true
	}
	return false
}

func isChromeImage(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	for _, c := range chromeImgs {
		if base == c {
			return true
		}
	}
	return false
}

func ParseHTMLFile(abs, root string) (HTMLPage, error) {
	b, err := os.ReadFile(abs)
	if err != nil {
		return HTMLPage{}, err
	}
	raw := DecodeToUTF8(b)
	rel, _ := filepath.Rel(root, abs)
	page := HTMLPage{Rel: filepath.ToSlash(rel)}

	for _, tag := range imgTagRe.FindAllString(raw, -1) {
		src := ""
		if m := imgSrcRe.FindStringSubmatch(tag); len(m) == 2 {
			src = html.UnescapeString(m[1])
		}
		if src == "" || strings.HasPrefix(strings.ToLower(src), "data:") {
			continue
		}
		resolved := filepath.Clean(filepath.Join(filepath.Dir(abs), src))
		if isChromeImage(resolved) {
			continue
		}
		if st, err := os.Stat(resolved); err != nil || st.IsDir() {
			continue
		}
		alt := ""
		if m := imgAltRe.FindStringSubmatch(tag); len(m) == 2 {
			alt = strings.TrimSpace(html.UnescapeString(m[1]))
		}
		key := figureKey(resolved)
		page.Imgs = append(page.Imgs, HTMLImage{Abs: resolved, RelKey: key, Alt: alt})
	}

	text := tagRe.ReplaceAllString(raw, " ")
	text = html.UnescapeString(text)
	text = spaceRe.ReplaceAllString(text, " ")
	text = strings.ReplaceAll(text, "\n ", "\n")
	text = strings.TrimSpace(text)
	if i := strings.Index(text, "GSIC - "); i >= 0 && i < 40 {
		text = strings.TrimSpace(text[i+len("GSIC - Global Service Information Center"):])
	}
	page.Text = text
	if utf8.RuneCountInString(text) > 40 {
		runes := []rune(text)
		end := 80
		if len(runes) < end {
			end = len(runes)
		}
		page.Title = strings.TrimSpace(string(runes[:end]))
	}
	return page, nil
}

func figureKey(abs string) string {
	abs = filepath.ToSlash(abs)
	for _, needle := range []string{"/repair2/img/", "/ncf/repair2/img/", "/brm/", "/ewd/"} {
		if i := strings.Index(strings.ToLower(abs), needle); i >= 0 {
			return strings.TrimPrefix(abs[i+1:], "/")
		}
	}
	return filepath.Base(abs)
}
