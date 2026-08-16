package ai

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 1x1 PNG so ParseHTMLFile can stat the figure file.
var png1x1 = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
	0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
	0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x05, 0xfe, 0xd4, 0xef, 0x00, 0x00,
	0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}

func TestParseHTMLFileExtractsTextAndImages(t *testing.T) {
	root := t.TempDir()
	imgDir := filepath.Join(root, "repair2", "img")
	pageDir := filepath.Join(root, "repair2", "html", "contents")
	if err := os.MkdirAll(imgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	img := filepath.Join(imgDir, "a202741.png")
	if err := os.WriteFile(img, png1x1, 0o644); err != nil {
		t.Fatal(err)
	}
	page := filepath.Join(pageDir, "rm000000veo00rx.html")
	html := `<html><body><h1>INSTRUMENT PANEL SPEAKER (for 11 speakers) &gt; REMOVAL</h1>` +
		`<p>Follow the same procedure on the right and left sides. The procedure explained below corresponds to the left side.</p>` +
		`<img alt="A202741" src="../../img/a202741.png"/>` +
		`</body></html>`
	if err := os.WriteFile(page, []byte(html), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ParseHTMLFile(page, root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Text, "INSTRUMENT PANEL SPEAKER") {
		t.Fatalf("text: %q", clip(got.Text, 160))
	}
	if len(got.Imgs) != 1 {
		t.Fatalf("images: %+v", got.Imgs)
	}
	if got.Imgs[0].Abs != img {
		t.Fatalf("abs path %s", got.Imgs[0].Abs)
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
