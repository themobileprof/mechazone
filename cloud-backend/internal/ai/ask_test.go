package ai

import (
	"strings"
	"testing"

	"mechazone/cloud-backend/internal/ledger"
)

func TestBuildAskPromptStripsCustomer(t *testing.T) {
	g := gathered{
		hist: ledger.History{
			Customer: &ledger.ShopCustomer{
				DisplayName: "Ada Okonkwo",
				Phone:       "08030000000",
				Plate:       "ABC-123",
			},
			Capture: &ledger.BusCapture{Profile: "toyota_common"},
		},
		allowedFigures: map[string]struct{}{},
	}
	s, err := buildAskPrompt(AskRequest{
		Request:  Request{VIN: "ZZZZCUSTDEV000001", Language: "en"},
		Step:     Step{Order: 1, Kind: "test", Title: "Prove battery on pin 16", Detail: "Red in DLC pin 16."},
		Question: "What voltage should I see?",
	}, g)
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"Ada Okonkwo", "08030000000", "ABC-123"} {
		if strings.Contains(s, leak) {
			t.Fatalf("ask prompt must not include %q", leak)
		}
	}
	if strings.Contains(s, `"capture"`) && strings.Contains(s, "toyota_common") {
		t.Fatal("ask shop_work must not include the bus capture")
	}
	for _, want := range []string{"focus_step", "Prove battery on pin 16", "What voltage should I see?", "question"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q", want)
		}
	}
}

func TestSanitizeAskDropsInventedFigures(t *testing.T) {
	g := gathered{
		allowedFigures: map[string]struct{}{"figure:real": {}},
		figs: []ledger.RetrievedFigure{{
			ID: "real", Title: "DLC", Page: 12, Caption: "pin 16", ImageURL: "/api/v1/manuals/figures/real/image",
		}},
	}
	got := sanitizeAsk(askModelOut{
		Answer:  "Use the cited figure.",
		Figures: StringList{"figure:invented", "figure:real"},
	}, g)
	if got.Answer != "Use the cited figure." {
		t.Fatalf("answer: %q", got.Answer)
	}
	if len(got.Figures) != 1 || got.Figures[0].ID != "real" {
		t.Fatalf("figures: %+v", got.Figures)
	}
	found := false
	for _, gap := range got.Gaps {
		if strings.Contains(gap, "uncited figure") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected uncited figure gap, got %v", got.Gaps)
	}
}

func TestClipAskThread(t *testing.T) {
	in := []AskTurn{
		{Role: "system", Content: "ignore"},
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "ok"},
		{Role: "user", Content: "   "},
	}
	got := clipAskThread(in)
	if len(got) != 2 || got[0].Role != "user" || got[1].Role != "assistant" {
		t.Fatalf("%+v", got)
	}
}
