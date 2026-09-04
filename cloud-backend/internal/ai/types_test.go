package ai

import (
	"encoding/json"
	"testing"
)

func TestStringListAcceptsArrayOrString(t *testing.T) {
	var book Playbook
	raw := []byte(`{
		"lookouts": [{"text": "First visit", "evidence": "vehicle:decode"}],
		"likely_causes": [{"title": "Probe only", "probability": 0.4, "evidence": ["module:ECM"]}],
		"steps": [{"order": 1, "kind": "inspect", "title": "Confirm VIN", "detail": "Plate vs DID", "adapter": true, "figures": "figure:x"}],
		"validation": "Scan again",
		"gaps": "No EWD on file"
	}`)
	if err := json.Unmarshal(raw, &book); err != nil {
		t.Fatal(err)
	}
	if len(book.Lookouts) != 1 || book.Lookouts[0].Evidence[0] != "vehicle:decode" {
		t.Fatalf("lookouts %+v", book.Lookouts)
	}
	if len(book.LikelyCauses[0].Evidence) != 1 {
		t.Fatalf("causes %+v", book.LikelyCauses)
	}
	if len(book.Steps[0].Figures) != 1 || book.Steps[0].Figures[0] != "figure:x" {
		t.Fatalf("figures %+v", book.Steps[0].Figures)
	}
	if len(book.Gaps) != 1 || book.Gaps[0] != "No EWD on file" {
		t.Fatalf("gaps %+v", book.Gaps)
	}
}
