package ai

import "testing"

func TestClassifyCode(t *testing.T) {
	u := ClassifyCode("U011B", "Lost communication with valvematic")
	if u.Class != "lost_communication" {
		t.Fatalf("%+v", u)
	}
	bus := ClassifyCode("U0073", "")
	if bus.Class != "bus_off" {
		t.Fatalf("%+v", bus)
	}
	hi := ClassifyCode("P0113", "Intake air temperature circuit high")
	if hi.Class != "short_to_battery" {
		t.Fatalf("%+v", hi)
	}
	open := ClassifyCode("P0122", "Throttle position sensor circuit open")
	if open.Class != "open_circuit" {
		t.Fatalf("%+v", open)
	}
	comp := ClassifyCode("P1047", "")
	if comp.Class != "component" {
		t.Fatalf("%+v", comp)
	}
	if !WiringShaped([]CircuitClass{u, comp}) {
		t.Fatal("U-code should make the job wiring-shaped")
	}
}

func TestInferNetwork(t *testing.T) {
	branch := InferNetwork([]ModuleHit{
		{Name: "ECM", Confirmed: true, Reachable: true},
		{Name: "VALVEMATIC", Confirmed: true, Reachable: false},
		{Name: "ABS", Confirmed: false, Reachable: false},
	})
	if branch.Reading != "branch" {
		t.Fatalf("%+v", branch)
	}
	back := InferNetwork([]ModuleHit{
		{Name: "ECM", Confirmed: true, Reachable: false},
		{Name: "ABS", Confirmed: false, Reachable: false},
	})
	if back.Reading != "backbone" {
		t.Fatalf("%+v", back)
	}
}
