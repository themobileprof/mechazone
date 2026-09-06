package ledger

import (
	"strings"
	"testing"
)

func TestSanitizeHowToHTMLStripsScript(t *testing.T) {
	got := SanitizeHowToHTML(`<p>Ok</p><script>alert(1)</script><img src="/howto/meter-volts-dial.jpg" alt="dial">`)
	if strings.Contains(got, "script") || strings.Contains(got, "alert") {
		t.Fatal(got)
	}
	if !strings.Contains(got, `/howto/meter-volts-dial.jpg`) {
		t.Fatal(got)
	}
}

func TestSanitizeHowToHTMLDropsRemoteImage(t *testing.T) {
	got := SanitizeHowToHTML(`<img src="https://evil.example/x.png" alt="x"><img src="/howto/ok.jpg" alt="ok">`)
	if strings.Contains(got, "evil") {
		t.Fatal(got)
	}
	if !strings.Contains(got, "/howto/ok.jpg") {
		t.Fatal(got)
	}
}
