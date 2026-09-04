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
	s, err := buildUserPrompt(Request{VIN: "ZZZZCUSTDEV000001", Language: "en"}, hist, nil, nil, nil, nil, nil, NetworkHint{}, false)
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
