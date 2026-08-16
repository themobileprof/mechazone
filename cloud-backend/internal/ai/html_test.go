package ai

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParseHTMLFileExtractsTextAndSpanishImages(t *testing.T) {
	root := filepath.Clean("../../../../avensis-zrt27/english")
	page := filepath.Join(root, "rm11r2s/MANUAL.HTM/rm11r2s/repair2/html/contents/rm000000veo00rx.html")
	got, err := ParseHTMLFile(page, root)
	if err != nil {
		t.Skip(err)
	}
	if !strings.Contains(got.Text, "INSTRUMENT PANEL SPEAKER") {
		t.Fatalf("text: %q", clip(got.Text, 160))
	}
	if len(got.Imgs) == 0 {
		t.Fatal("expected procedure images")
	}
	for _, im := range got.Imgs {
		if isChromeImage(im.Abs) {
			t.Fatalf("chrome slipped in %s", im.Abs)
		}
		if !strings.Contains(im.Abs, "/spanish/") {
			t.Fatalf("expected spanish image path, got %s", im.Abs)
		}
	}
}

func TestShouldSkipHTML(t *testing.T) {
	if !shouldSkipHTML("rm11r2s/repair2/css/contents.css") {
		t.Fatal("css")
	}
	if shouldSkipHTML("rm11r2s/repair2/html/contents/rm000000veo00rx.html") {
		t.Fatal("contents page should be kept")
	}
}
