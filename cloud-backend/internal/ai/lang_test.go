package ai

import "testing"

func TestDetectLanguage(t *testing.T) {
	if g := DetectLanguage("Der Motor und die Steuerung sind nicht in Ordnung. Eine Prüfung der Ventile."); g != "de" {
		t.Fatalf("de got %s", g)
	}
	if g := DetectLanguage("The engine and the valves from this test with that reading."); g != "en" {
		t.Fatalf("en got %s", g)
	}
}

func TestParseFilename(t *testing.T) {
	s, ok := parseFilename("toyota_avensis_2009-2012_3zr-fae_de.pdf")
	if !ok || s.Make != "Toyota" || s.Model != "Avensis" || s.YearFrom != 2009 || s.YearTo != 2012 || s.Language != "de" {
		t.Fatalf("%+v ok=%v", s, ok)
	}
}

func TestChunkKeepsCodes(t *testing.T) {
	pages := []PageText{{Page: 4, Text: "Valvematic\n\nCode P1047 on ECM.\n\nMore text."}}
	chunks := ChunkPages(pages, 80)
	if len(chunks) == 0 {
		t.Fatal("no chunks")
	}
	found := false
	for _, c := range chunks {
		for _, code := range c.Codes {
			if code == "P1047" {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("codes lost: %+v", chunks)
	}
}
