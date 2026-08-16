package ai

import (
	"testing"
	"unicode/utf8"
)

func TestDecodeToUTF8_loneC3(t *testing.T) {
	// 0xc3 is Ã in Windows-1252; as a lone byte it is invalid UTF-8 (Postgres SQLSTATE 22021).
	got := DecodeToUTF8([]byte("pin 3\xc3"))
	if !utf8.ValidString(got) {
		t.Fatalf("still invalid UTF-8: %q", got)
	}
	if got != "pin 3Ã" {
		t.Fatalf("got %q", got)
	}
}

func TestDecodeToUTF8_keepsValidUTF8(t *testing.T) {
	in := []byte("Valvematic °C — 3ZR-FAE")
	if DecodeToUTF8(in) != string(in) {
		t.Fatalf("rewrote valid UTF-8")
	}
}
