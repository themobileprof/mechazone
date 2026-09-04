package ai

import (
	"strings"
	"testing"

	"mechazone/cloud-backend/internal/ledger"
)

func TestBuildUserPromptStripsCustomer(t *testing.T) {
	hist := ledger.History{
		Customer: &ledger.ShopCustomer{
			DisplayName: "Ada Okonkwo",
			Phone:       "08030000000",
			Plate:       "ABC-123",
		},
		Capture: &ledger.BusCapture{
			Profile: "toyota_common",
			Modules: []ledger.BusModule{{Name: "ECM", TxID: "7E0", EverReachable: true}},
		},
	}
	s, err := buildUserPrompt(Request{VIN: "ZZZZCUSTDEV000001", Language: "en"}, hist, nil, nil, nil, nil, nil, NetworkHint{}, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"Ada Okonkwo", "08030000000", "ABC-123"} {
		if strings.Contains(s, leak) {
			t.Fatalf("playbook prompt must not include %q", leak)
		}
	}
	if strings.Contains(s, `"capture"`) && strings.Contains(s, "toyota_common") {
		t.Fatal("playbook shop_work must not include the bus capture")
	}
}

func TestBuildUserPromptIncludesSettledChecks(t *testing.T) {
	s, err := buildUserPrompt(
		Request{VIN: "ZZZZCUSTDEV000001", Language: "en"},
		ledger.History{},
		nil, nil, nil, nil, nil, NetworkHint{}, false,
		[]ledger.PlaybookCheck{{
			Kind: "test", Title: "Compare target vs actual", Status: ledger.CheckRuledOut, Note: "angle in spec",
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"bay_checks", "Compare target vs actual", "ruled_out", "angle in spec"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q", want)
		}
	}
}
