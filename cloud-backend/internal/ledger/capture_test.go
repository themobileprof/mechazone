package ledger

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestMergeBusModulesKeepsEverReachable(t *testing.T) {
	prev := []BusModule{
		{Name: "ECM", TxID: "7E0", RxID: "7E8", Reachable: true, EverReachable: true, Confirmed: true, DTCs: []string{"P0420"}},
		{Name: "ABS", TxID: "760", RxID: "768", Reachable: true, EverReachable: true},
	}
	next := []BusModule{
		{Name: "ECM", TxID: "7E0", RxID: "7E8", Reachable: true, DTCs: []string{"P0300"}},
		{Name: "TCU", TxID: "7E1", RxID: "7E9", Reachable: false},
	}
	got := mergeBusModules(prev, next)
	if len(got) != 3 {
		t.Fatalf("len %d", len(got))
	}
	by := map[string]BusModule{}
	for _, m := range got {
		by[m.Name] = m
	}
	if !by["ECM"].EverReachable || !by["ECM"].Reachable || !by["ECM"].Confirmed {
		t.Fatalf("ECM %+v", by["ECM"])
	}
	if got := by["ECM"].DTCs; len(got) != 1 || got[0] != "P0300" {
		t.Fatalf("ECM dtcs %v", got)
	}
	if !by["ABS"].EverReachable {
		t.Fatal("ABS must stay ever_reachable")
	}
	if by["ABS"].Reachable {
		t.Fatal("ABS was dark this scan")
	}
	if len(by["ABS"].DTCs) != 0 {
		t.Fatalf("dark node must not keep last-scan DTCs: %v", by["ABS"].DTCs)
	}
	if by["TCU"].EverReachable {
		t.Fatal("TCU never answered")
	}
}

func TestMergeBusModulesCanonicalTxID(t *testing.T) {
	got := mergeBusModules(
		[]BusModule{{Name: "ECM", TxID: "7E0", RxID: "7E8", Reachable: true}},
		[]BusModule{{Name: "ECM", TxID: "0x7E0", RxID: "0x7E8", Reachable: true}},
	)
	if len(got) != 1 {
		t.Fatalf("len %d %+v", len(got), got)
	}
	if got[0].TxID != "7E0" || got[0].RxID != "7E8" || !got[0].Reachable || !got[0].EverReachable {
		t.Fatalf("%+v", got[0])
	}
}

func TestMergeBusIdentityUnionsByDID(t *testing.T) {
	got := mergeBusIdentity(
		[]BusIdentity{{Name: "VIN", DID: "F190", Text: "old"}},
		[]BusIdentity{{Name: "VIN", DID: "F190", Text: "new"}, {Name: "cal", DID: "F181", Text: "A"}},
	)
	if len(got) != 2 {
		t.Fatalf("len %d", len(got))
	}
	if got[0].Text != "new" || got[1].DID != "F181" {
		t.Fatalf("%+v", got)
	}
}

func TestBusCaptureUpsertMerge(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	code := fmt.Sprintf("ZZZZCAP%010d", time.Now().UnixNano()%1e10)
	if err := s.EnsureVehicle(ctx, code, "Toyota", "Unknown", 2010, "capture"); err != nil {
		t.Fatal(err)
	}
	shop := "00000000-0000-4000-8000-000000000001"
	tech := "00000000-0000-4000-8000-000000000002"
	first, err := s.UpsertBusCapture(ctx, code, shop, tech, BusCaptureIn{
		Profile:     "toyota_common",
		AdapterType: "openport2_rev_e",
		HostOS:      "linux",
		Protocol:    "iso_15765_4",
		MakeHint:    "Toyota",
		YearHint:    2010,
		Modules: []BusModule{
			{Name: "ECM", TxID: "7E0", RxID: "7E8", Reachable: true, Confirmed: true},
		},
		Identity:     []BusIdentity{{Name: "VIN", DID: "F190", Text: ""}},
		Live:         []BusLive{{Name: "rpm", DID: "010C", Unit: "rpm", Value: 800}},
		ActiveCodes:  []string{"P0420"},
		RawHexStream: []string{"7E0 02 10 01"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.ScanCount != 1 || !first.Modules[0].EverReachable {
		t.Fatalf("first %+v", first)
	}
	second, err := s.UpsertBusCapture(ctx, code, shop, tech, BusCaptureIn{
		Profile:     "toyota_common",
		AdapterType: "openport2_rev_e",
		HostOS:      "linux",
		Protocol:    "iso_15765_4",
		Modules: []BusModule{
			{Name: "ECM", TxID: "7E0", RxID: "7E8", Reachable: false},
			{Name: "ABS", TxID: "760", RxID: "768", Reachable: true},
		},
		Identity:    []BusIdentity{{Name: "VIN", DID: "F190", Text: "JTDBT40E501234567"}},
		Live:        []BusLive{{Name: "rpm", DID: "010C", Unit: "rpm", Value: 1200}},
		ActiveCodes: []string{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.ScanCount != 2 {
		t.Fatalf("scan_count %d", second.ScanCount)
	}
	if second.MakeHint != "Toyota" || second.YearHint != 2010 {
		t.Fatalf("hints %+v", second)
	}
	if second.RawHexExcerpt == "" {
		t.Fatal("keep last hex when this scan has none")
	}
	by := map[string]BusModule{}
	for _, m := range second.Modules {
		by[m.Name] = m
	}
	if !by["ECM"].EverReachable || by["ECM"].Reachable {
		t.Fatalf("ECM %+v", by["ECM"])
	}
	if !by["ABS"].EverReachable || !by["ABS"].Reachable {
		t.Fatalf("ABS %+v", by["ABS"])
	}
	if len(second.Identity) != 1 || second.Identity[0].Text != "JTDBT40E501234567" {
		t.Fatalf("identity %+v", second.Identity)
	}
	h, err := s.History(ctx, code, shop, tech)
	if err != nil {
		t.Fatal(err)
	}
	if h.Capture == nil || h.Capture.ScanCount != 2 {
		t.Fatalf("history capture %+v", h.Capture)
	}
	if !h.FirstSeen {
		t.Fatal("capture must not count as a closed job")
	}
	other, err := s.BusCapture(ctx, code, "00000000-0000-4000-8000-000000000099", tech)
	if err != nil {
		t.Fatal(err)
	}
	if other != nil {
		t.Fatal("another shop must not read this bus map")
	}
}
