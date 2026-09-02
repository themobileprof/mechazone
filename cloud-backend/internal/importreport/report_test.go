package importreport

import "testing"

func TestParseCodes(t *testing.T) {
	got := ParseCodes("pending p0301, P0301 and U0100 plus junk")
	if len(got) != 2 || got[0] != "P0301" || got[1] != "U0100" {
		t.Fatalf("got %#v", got)
	}
	if len(ParseCodes("no codes here")) != 0 {
		t.Fatal("empty expected")
	}
}

func TestNormalizeSource(t *testing.T) {
	s, err := NormalizeSource("X-431")
	if err != nil || s != "x431" {
		t.Fatalf("x431: %q %v", s, err)
	}
	if _, err := NormalizeSource("launch_cloud"); err == nil {
		t.Fatal("unknown source must fail")
	}
	s, err = NormalizeSource("")
	if err != nil || s != "other" {
		t.Fatalf("blank: %q %v", s, err)
	}
}

func TestSanitizeFilename(t *testing.T) {
	if got := SanitizeFilename(`..\..\etc\passwd`); got != "etc_passwd" {
		t.Fatalf("got %q", got)
	}
	if got := SanitizeFilename("/tmp/Avensis report (1).PDF"); got != "Avensis_report__1_.PDF" {
		t.Fatalf("got %q", got)
	}
	if SanitizeFilename("///") != "report" {
		t.Fatalf("empty: %q", SanitizeFilename("///"))
	}
}

func TestSniff(t *testing.T) {
	ct, ext, err := Sniff([]byte("%PDF-1.4\n"), "scan.pdf")
	if err != nil || ct != "application/pdf" || ext != ".pdf" {
		t.Fatalf("pdf: %s %s %v", ct, ext, err)
	}
	jpeg := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00}
	ct, ext, err = Sniff(jpeg, "photo.jpg")
	if err != nil || ct != "image/jpeg" || ext != ".jpg" {
		t.Fatalf("jpeg: %s %s %v", ct, ext, err)
	}
	if _, _, err := Sniff([]byte{0x00, 0x01, 0x02}, "blob.bin"); err == nil {
		t.Fatal("binary must fail")
	}
	ct, ext, err = Sniff([]byte("P0301,P0420\n"), "codes.csv")
	if err != nil || ct != "text/csv" {
		t.Fatalf("csv: %s %s %v", ct, ext, err)
	}
}

func TestResolveStorage(t *testing.T) {
	full, err := ResolveStorage("/var/imports", "shop-a/sess.pdf")
	if err != nil || full != "/var/imports/shop-a/sess.pdf" {
		t.Fatalf("got %q %v", full, err)
	}
	if _, err := ResolveStorage("/var/imports", "../etc/passwd"); err == nil {
		t.Fatal("escape must fail")
	}
	if _, err := ResolveStorage("/var/imports", "/etc/passwd"); err == nil {
		t.Fatal("abs must fail")
	}
}

func TestHostOS(t *testing.T) {
	if HostOS("windows", "") != "windows" {
		t.Fatal("form")
	}
	if HostOS("", "Mozilla/5.0 (X11; Linux x86_64)") != "linux" {
		t.Fatal("ua")
	}
}
