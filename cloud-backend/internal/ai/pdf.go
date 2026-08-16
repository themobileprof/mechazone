package ai

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func ExtractPDFPages(path string) ([]PageText, error) {
	cmd := exec.Command("pdftotext", "-layout", "-enc", "UTF-8", path, "-")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pdftotext (install poppler-utils): %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	raw := strings.ReplaceAll(stdout.String(), "\r\n", "\n")
	parts := strings.Split(raw, "\f")
	pages := make([]PageText, 0, len(parts))
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		pages = append(pages, PageText{Page: i + 1, Text: part})
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no text extracted from %s", path)
	}
	return pages, nil
}
