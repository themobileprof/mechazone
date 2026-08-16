package ai

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var dtcRe = regexp.MustCompile(`\b[PBCU][0-9A-Fa-f]{4}\b`)
var figRe = regexp.MustCompile(`(?i)\b(?:fig(?:ure)?|abb(?:ildung)?|figuur|figura|schéma|図)\.?\s*[\w\-\.]+`)

type PageText struct {
	Page int
	Text string
}

func ExtractCodes(text string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, m := range dtcRe.FindAllString(text, -1) {
		m = strings.ToUpper(m)
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		out = append(out, m)
	}
	return out
}

func ExtractFigureCaptions(text string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, m := range figRe.FindAllString(text, 12) {
		m = strings.TrimSpace(m)
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		out = append(out, m)
	}
	return out
}

func ChunkPages(pages []PageText, maxRunes int) []DocChunkInLite {
	if maxRunes <= 0 {
		maxRunes = 1400
	}
	out := []DocChunkInLite{}
	for _, p := range pages {
		text := strings.TrimSpace(p.Text)
		if text == "" {
			continue
		}
		if utf8.RuneCountInString(text) <= maxRunes {
			out = append(out, DocChunkInLite{Page: p.Page, Body: text, Codes: ExtractCodes(text)})
			continue
		}
		paras := splitKeep(text)
		var buf strings.Builder
		for _, para := range paras {
			if buf.Len() > 0 && utf8.RuneCountInString(buf.String()+"\n\n"+para) > maxRunes {
				body := strings.TrimSpace(buf.String())
				out = append(out, DocChunkInLite{Page: p.Page, Body: body, Codes: ExtractCodes(body)})
				buf.Reset()
			}
			if buf.Len() > 0 {
				buf.WriteString("\n\n")
			}
			buf.WriteString(para)
		}
		if body := strings.TrimSpace(buf.String()); body != "" {
			out = append(out, DocChunkInLite{Page: p.Page, Body: body, Codes: ExtractCodes(body)})
		}
	}
	return out
}

type DocChunkInLite struct {
	Page  int
	Body  string
	Codes []string
}

func splitKeep(text string) []string {
	parts := strings.Split(text, "\n\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{strings.TrimSpace(text)}
	}
	return out
}
