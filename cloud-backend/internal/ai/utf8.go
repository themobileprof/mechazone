package ai

import (
	"bytes"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// CleanUTF8 makes text safe for PostgreSQL UTF8. OEM HTML and OCR often
// arrive as Windows-1252 (0xc3 is Ã), which Postgres rejects as a truncated UTF-8 sequence.
func CleanUTF8(s string) string {
	return DecodeToUTF8([]byte(s))
}

func DecodeToUTF8(b []byte) string {
	b = bytes.ReplaceAll(b, []byte{0}, nil)
	if utf8.Valid(b) {
		return string(b)
	}
	var out []byte
	for i := 0; i < len(b); {
		r, size := utf8.DecodeRune(b[i:])
		if r == utf8.RuneError && size == 1 {
			mapped, err := charmap.Windows1252.NewDecoder().Bytes([]byte{b[i]})
			if err != nil || len(mapped) == 0 {
				out = utf8.AppendRune(out, '\uFFFD')
			} else {
				out = append(out, mapped...)
			}
			i++
			continue
		}
		out = append(out, b[i:i+size]...)
		i += size
	}
	return string(out)
}
