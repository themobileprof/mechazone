package ai

import (
	"strings"
)

type CircuitClass struct {
	Code   string `json:"code"`
	Class  string `json:"class"`
	Reason string `json:"reason"`
}

type ModuleHit struct {
	Name      string   `json:"name"`
	TxID      string   `json:"tx_id"`
	RxID      string   `json:"rx_id"`
	Family    string   `json:"family,omitempty"`
	Confirmed bool     `json:"confirmed"`
	Reachable bool     `json:"reachable"`
	DTCs      []string `json:"dtcs,omitempty"`
	Error     string   `json:"error,omitempty"`
}

type NetworkHint struct {
	Reading string `json:"reading"`
	Summary string `json:"summary"`
	Live    int    `json:"live"`
	Dark    int    `json:"dark"`
}

func ClassifyCode(code, title string) CircuitClass {
	code = strings.ToUpper(strings.TrimSpace(code))
	title = strings.ToLower(strings.TrimSpace(title))
	out := CircuitClass{Code: code, Class: "component", Reason: "No circuit or bus pattern in the code or title."}
	if code == "" {
		return out
	}
	if code[0] == 'U' {
		if strings.Contains(code, "0073") || strings.Contains(title, "bus off") {
			return CircuitClass{Code: code, Class: "bus_off", Reason: "Network bus-off / U0073-class."}
		}
		return CircuitClass{Code: code, Class: "lost_communication", Reason: "U-code: lost communication with a module."}
	}
	switch {
	case hasAny(title, "short to battery", "short to batt", "short to b+", "circuit high"):
		return CircuitClass{Code: code, Class: "short_to_battery", Reason: "Title is circuit-high / short to battery."}
	case hasAny(title, "short to ground", "short to gnd", "circuit low"):
		return CircuitClass{Code: code, Class: "short_to_ground", Reason: "Title is circuit-low / short to ground."}
	case hasAny(title, "open circuit", "circuit open", "open in"):
		return CircuitClass{Code: code, Class: "open_circuit", Reason: "Title is an open circuit."}
	case strings.Contains(title, "circuit"):
		return CircuitClass{Code: code, Class: "circuit", Reason: "Title names a circuit, not only a component."}
	}
	return out
}

func ClassifyCodes(codes []string, titles map[string]string) []CircuitClass {
	out := make([]CircuitClass, 0, len(codes))
	for _, c := range codes {
		title := ""
		if titles != nil {
			title = titles[strings.ToUpper(strings.TrimSpace(c))]
		}
		out = append(out, ClassifyCode(c, title))
	}
	return out
}

func WiringShaped(classes []CircuitClass) bool {
	for _, c := range classes {
		switch c.Class {
		case "open_circuit", "short_to_battery", "short_to_ground", "circuit", "lost_communication", "bus_off":
			return true
		}
	}
	return false
}

func InferNetwork(mods []ModuleHit) NetworkHint {
	live, dark := 0, 0
	ecmUp := false
	var confirmedDark []string
	for _, m := range mods {
		if m.Reachable {
			live++
			if strings.EqualFold(m.Name, "ECM") {
				ecmUp = true
			}
			continue
		}
		dark++
		if m.Confirmed {
			confirmedDark = append(confirmedDark, m.Name)
		}
	}
	h := NetworkHint{Live: live, Dark: dark}
	if !ecmUp {
		h.Reading = "backbone"
		h.Summary = "ECM did not answer. Check DLC power, ground, and CAN before blaming a single module."
		return h
	}
	if len(confirmedDark) > 0 {
		h.Reading = "branch"
		h.Summary = "ECM is live. Silent confirmed node: " + strings.Join(confirmedDark, ", ") + ". That branch (power / ground / CAN), not the whole bus."
		return h
	}
	if dark >= 3 {
		h.Reading = "probes_silent"
		h.Summary = "ECM is live. Extra Toyota 11-bit probes did not answer — they may be absent on this car. Do not treat as a backbone failure."
		return h
	}
	h.Reading = "ok"
	h.Summary = "Probed modules answered."
	return h
}

func hasAny(s string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}
