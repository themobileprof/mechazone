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
}
