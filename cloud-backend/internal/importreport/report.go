package importreport

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const MaxBytes = 8 << 20

var (
	dtcRe   = regexp.MustCompile(`(?i)\b([PCBU][0-9A-F]{4})\b`)
	sources = []string{"x431", "autel", "techstream", "forscan", "golo", "snap_on", "other"}
)

func Sources() []string {
	out := make([]string, len(sources))
	copy(out, sources)
	return out
}

func NormalizeSource(raw string) (string, error) {
	compact := strings.NewReplacer("-", "", "_", "", " ", "").Replace(strings.ToLower(strings.TrimSpace(raw)))
	switch compact {
	case "", "other":
		return "other", nil
	case "x431", "launch", "launchx431":
		return "x431", nil
	case "autel", "maxisys":
		return "autel", nil
	case "techstream", "gts":
		return "techstream", nil
	case "forscan":
		return "forscan", nil
	case "golo", "golo365":
		return "golo", nil
	case "snapon":
		return "snap_on", nil
	default:
		return "", fmt.Errorf("source must be one of: %s", strings.Join(sources, ", "))
	}
}

func ParseCodes(raw string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, m := range dtcRe.FindAllString(raw, 40) {
		code := strings.ToUpper(m)
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	if out == nil {
		return []string{}
	}
	return out
}

func SanitizeFilename(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	base = strings.ReplaceAll(base, "\\", "_")
	var b strings.Builder
	for _, r := range base {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "._")
	if out == "" {
		out = "report"
	}
	if len(out) > 120 {
		out = out[:120]
	}
	return out
}

func Sniff(head []byte, filename string) (contentType, ext string, err error) {
	name := strings.ToLower(filepath.Base(filename))
	switch {
	case bytes.HasPrefix(head, []byte("%PDF")):
		return "application/pdf", ".pdf", nil
	case bytes.HasPrefix(head, []byte("\x89PNG\r\n\x1a\n")):
		return "image/png", ".png", nil
	case len(head) >= 3 && head[0] == 0xff && head[1] == 0xd8 && head[2] == 0xff:
		return "image/jpeg", ".jpg", nil
	case len(head) >= 12 && bytes.Equal(head[:4], []byte("RIFF")) && bytes.Equal(head[8:12], []byte("WEBP")):
		return "image/webp", ".webp", nil
	}

	trimmed := bytes.TrimSpace(head)
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') && utf8.Valid(head) && !bytes.Contains(head, []byte{0}) {
		if strings.HasSuffix(name, ".json") || trimmed[0] == '{' || trimmed[0] == '[' {
			return "application/json", ".json", nil
		}
	}
	if utf8.Valid(head) && !bytes.Contains(head, []byte{0}) {
		switch {
		case strings.HasSuffix(name, ".csv"):
			return "text/csv", ".csv", nil
		case strings.HasSuffix(name, ".txt"), strings.HasSuffix(name, ".log"):
			return "text/plain", ".txt", nil
		case looksLikeCSV(head):
			return "text/csv", ".csv", nil
		case len(bytes.TrimSpace(head)) > 0:
			return "text/plain", ".txt", nil
		}
	}
	return "", "", fmt.Errorf("unsupported file type (PDF, JPEG, PNG, WebP, TXT, CSV, or JSON)")
}

func looksLikeCSV(head []byte) bool {
	line, _, _ := bytes.Cut(head, []byte("\n"))
	return bytes.Count(line, []byte(",")) >= 1
}

func HostOS(formValue, userAgent string) string {
	v := strings.ToLower(strings.TrimSpace(formValue))
	switch {
	case strings.Contains(v, "win"):
		return "windows"
	case strings.Contains(v, "darwin") || strings.Contains(v, "mac"):
		return "darwin"
	case strings.Contains(v, "linux"):
		return "linux"
	}
	ua := strings.ToLower(userAgent)
	switch {
	case strings.Contains(ua, "windows"):
		return "windows"
	case strings.Contains(ua, "mac os") || strings.Contains(ua, "macintosh"):
		return "darwin"
	case strings.Contains(ua, "linux"):
		return "linux"
	}
	return "unknown"
}

func ScopeDir(shopID, technicianID string) string {
	shopID = strings.TrimSpace(shopID)
	if shopID != "" {
		return shopID
	}
	return "tech-" + strings.TrimSpace(technicianID)
}

func ResolveStorage(root, stored string) (string, error) {
	root = filepath.Clean(root)
	if stored == "" {
		return "", fmt.Errorf("empty storage path")
	}
	if filepath.IsAbs(stored) {
		return "", fmt.Errorf("storage path must be relative")
	}
	full := filepath.Clean(filepath.Join(root, stored))
	rel, err := filepath.Rel(root, full)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("storage path escapes import dir")
	}
	return full, nil
}
