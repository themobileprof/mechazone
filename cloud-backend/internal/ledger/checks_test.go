package ledger

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestCheckFingerprint(t *testing.T) {
	if CheckFingerprint("Test", " Compare target ") != "test|compare target" {
		t.Fatal(CheckFingerprint("Test", " Compare target "))
	}
}

func TestPlaybookCheckUpsertKeepsSettled(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	code := fmt.Sprintf("ZZZZCHK%010d", time.Now().UnixNano()%1e10)
	if err := s.EnsureVehicle(ctx, code, "Toyota", "Avensis", 2010, "check"); err != nil {
		t.Fatal(err)
	}
	shop := "00000000-0000-4000-8000-000000000001"
	tech := "00000000-0000-4000-8000-000000000002"
	if _, err := s.SyncPlaybookSteps(ctx, code, shop, tech, []PlaybookStepSeed{
		{Kind: "test", Title: "Compare target vs actual", Detail: "DID 1A01"},
		{Kind: "inspect", Title: "Connector", Detail: "corrosion"},
	}); err != nil {
		t.Fatal(err)
	}
	saved, err := s.UpsertPlaybookCheck(ctx, code, shop, tech, PlaybookCheckIn{
		Kind: "test", Title: "Compare target vs actual", Status: CheckRuledOut, Note: "angle is in spec",
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.Status != CheckRuledOut || saved.Fingerprint != "test|compare target vs actual" {
		t.Fatalf("%+v", saved)
	}
	rows, err := s.SyncPlaybookSteps(ctx, code, shop, tech, []PlaybookStepSeed{
		{Kind: "test", Title: "Compare target vs actual", Detail: "DID 1A01 again"},
		{Kind: "test", Title: "Wiggle the loom", Detail: "ECM harness"},
	})
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]PlaybookCheck{}
	for _, c := range rows {
		by[c.Fingerprint] = c
	}
	if by["test|compare target vs actual"].Status != CheckRuledOut {
		t.Fatalf("must not reopen a ruled-out check: %+v", by["test|compare target vs actual"])
	}
	if by["test|compare target vs actual"].Note != "angle is in spec" {
		t.Fatalf("note %+v", by["test|compare target vs actual"])
	}
	if by["test|wiggle the loom"].Status != CheckOpen {
		t.Fatalf("new step %+v", by["test|wiggle the loom"])
	}
	h, err := s.History(ctx, code, shop, tech)
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Checks) < 2 {
		t.Fatalf("history checks %d", len(h.Checks))
	}
	other, err := s.PlaybookChecks(ctx, code, "00000000-0000-4000-8000-000000000099", tech)
	if err != nil {
		t.Fatal(err)
	}
	if len(other) != 0 {
		t.Fatal("another shop must not read these checks")
	}
}
