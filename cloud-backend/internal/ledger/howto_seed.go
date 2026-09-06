package ledger

import (
	"context"
	"html"
	"strconv"
	"strings"
)

type seedPlate struct {
	Title, Body, File, Alt string
}

type seedGuide struct {
	Slug, Title, Blurb, Warning string
	Words                       []string
	Plates                      []seedPlate
}

func seedGuideHTML(plates []seedPlate) string {
	var b strings.Builder
	for i, p := range plates {
		n := strconv.Itoa(i + 1)
		if len(n) == 1 {
			n = "0" + n
		}
		b.WriteString("<h3>")
		b.WriteString(html.EscapeString(n + " · " + p.Title))
		b.WriteString("</h3><p>")
		b.WriteString(html.EscapeString(p.Body))
		b.WriteString(`</p><figure><img src="/howto/`)
		b.WriteString(html.EscapeString(p.File))
		b.WriteString(`" alt="`)
		b.WriteString(html.EscapeString(p.Alt))
		b.WriteString(`"></figure>`)
	}
	return b.String()
}

func defaultHowToSeeds() []seedGuide {
	return []seedGuide{
		{
			Slug: "dlc-power-ground", Title: "Power and ground at the 16-pin DLC",
			Blurb:   "The under-dash OBD port is SAE J1962. Pin 16 is battery. Pins 4 and 5 are grounds. That numbering is the port, not a random ECU plug.",
			Warning: "Use this card only when the playbook means the diagnostic link connector under the dash. If it named a module connector, ignore DLC pin numbers and use the cited figure.",
			Words:   []string{"dlc", "obd", "16-pin", "diagnostic link", "pin 16", "j1962"},
			Plates: []seedPlate{
				{"This is the OBD2 port", "The socket in the car is a female trapezoid: two rows of eight holes, plastic housing, often dusty. It is not a round cigarette-lighter hole and not a small OEM plug beside it. If it does not look like this, you are on the wrong connector.", "obd2-port-face.jpg", "Close-up of a 16-pin female OBD-II socket in a vehicle"},
				{"Find it under the dash", "Sit in the driver seat. The 16-pin DLC is usually under the dash, left of the steering column, within arm's reach. Do not confuse it with a smaller OEM plug next to it.", "dlc-power-ground-location.jpg", "16-pin diagnostic port under a dashboard"},
				{"Know which holes are which", "Look into the car socket with the wide side up. Top row is pins 1–8 left to right. Bottom row is 9–16. Pin 16 (bottom right) is battery. Pin 4 is chassis ground. Pin 5 is signal ground. If the playbook said pin 4 or 5 on the DLC, those are the two grounds. Do not use these numbers on a module connector.", "dlc-power-ground-pinout.svg", "SAE J1962 vehicle socket pin map with 4, 5, and 16 marked"},
				{"Prove battery on pin 16", "Key as the playbook says (usually ignition on). Meter on DC volts. Black probe on battery negative or a known-good chassis ground. Red probe into pin 16. Expect about battery voltage (often 12–14 V). Zero or OL means the port is not powered — stop and fix that before any module test.", "dlc-power-ground-pin16.jpg", "Red probe in DLC pin 16, meter showing about 12 volts"},
			},
		},
		{
			Slug: "backprobe", Title: "Land on a pin without wrecking the connector",
			Blurb:   "Most live tests want the connector still plugged in. You reach the metal from the wire side, not by stabbing the insulation.",
			Warning: "The playbook (or a cited figure) names the cavity. This card does not. Do not pierce the wire if you have a backprobe pin or a meter lead that fits the rear of the housing.",
			Words:   []string{"backprobe", "back-probe", "rear of the connector", "cavity", "still plugged"},
			Plates: []seedPlate{
				{"Leave it connected", "Push the connector fully home so the lock clicks. You are measuring the circuit as the car sees it. Unplugging is for ohms on a component, not for a live voltage at a running module.", "backprobe-connected.jpg", "A locked automotive connector still mated, wires exiting the back"},
				{"Slide in from the wire side", "From the back of the housing, slip a backprobe pin or a thin meter probe alongside the wire until it touches the terminal. You should feel a light stop. Do not force it past the seal any harder than a paperclip into a tight hole.", "backprobe-rear.jpg", "A thin probe entering the rear of a connector beside a wire"},
				{"Do not pierce the loom", "A vampire / piercing probe through the insulation lets water in and can cut a strand. Use it only if you have no rear access and you will seal the nick. Prefer the rear cavity.", "backprobe-no-pierce.jpg", "A piercing probe on a wire marked as the wrong method"},
			},
		},
		{
			Slug: "meter-continuity", Title: "Continuity (beep) — is there a path?",
			Blurb:   "Power off. You are asking: do these two points connect through a wire or a closed switch? A beep is a path. Silence plus OL is open.",
			Warning: "Never continuity-test a live circuit. Key out, battery still in the car is fine, but no voltage on the two points. If the playbook named two pins, those are the two points — not any pin 4.",
			Words:   []string{"continuity", "beep test", "diode mode", "open wire", "open circuit"},
			Plates: []seedPlate{
				{"Leads and dial", "Black lead in COM. Red lead in the jack labelled VΩmA or VΩ (not the 10 A / 20 A jack). Turn the dial to the speaker / diode symbol — not V, not A, not a high ohm range if your meter has a dedicated beep.", "meter-continuity-dial.jpg", "Multimeter dial on the continuity or diode symbol"},
				{"Prove the meter first", "Touch the two probe tips together. You should hear a beep and see a number near 0 Ω (maybe 0.2–0.5 because of the leads). If nothing happens, the leads are in the wrong holes or the dial is wrong.", "meter-continuity-beep.jpg", "Probe tips touching, meter beeping near zero ohms"},
				{"Then the two points from the playbook", "One probe on each point the playbook named. Beep + low ohms = a path. No beep and OL or 1. on the left of the display = open. A few ohms on a long earth strap can still be good; a few thousand on a supposed short jumper is not.", "meter-continuity-ol.jpg", "Meter display showing OL with probes on an open circuit"},
			},
		},
		{
			Slug: "meter-ohms", Title: "Measure resistance (ohms)",
			Blurb:   "Power off. Red and black go in specific holes. The dial must be on Ω. The number is how hard it is for current to get through. OL means the path is open.",
			Warning: "Do not measure ohms on a live pin. Key out. Unplug the component if the playbook says to. The pin numbers come from the playbook or a cited figure — this card never invents pin 4 or pin 5 on a mystery connector.",
			Words:   []string{"ohm", "resist", "Ω", "kΩ", "kohm", "megohm"},
			Plates: []seedPlate{
				{"Plug the leads in", "Black banana plug → COM (common). Red banana plug → the jack labelled VΩmA, VΩ, or a star of V / Ω / mA. Do not put red in the separate 10 A jack — that hole is only for high current, and ohms will not work there.", "meter-ohms-jacks.jpg", "Black lead in COM and red lead in the V ohms milliamps jack"},
				{"Set the dial to Ω", "Turn to the Greek omega (Ω). If the meter is manual-range, start on a middle range (200 Ω or 2 kΩ) and go up if you only see OL. Auto-range meters pick the range for you. Not V (volts). Not A (amps). Not the speaker unless the playbook asked for continuity.", "meter-ohms-dial.jpg", "Multimeter rotary switch pointing at the ohms symbol"},
				{"Zero the leads", "Touch red and black tips together. You should see a small number, often 0.2–0.8 Ω. That is the leads, not the car. If you see OL, the dial or the jacks are still wrong.", "meter-ohms-zero.jpg", "Probes shorted together, display a fraction of an ohm"},
				{"Read the two points the playbook named", "One probe on each pin or terminal in the playbook (for example the two ends of a sensor, or a pin to ground if it said so). Hold still. OL or 1. on the left = open (no path). A number with k means thousands of ohms (2.20 kΩ is 2200 Ω). M means millions. Compare to the pass/fail line on that playbook step — not to a number from this card.", "meter-ohms-on-pins.jpg", "Probes on two terminals of an unplugged generic connector"},
			},
		},
		{
			Slug: "meter-volts", Title: "Measure DC voltage",
			Blurb:   "This is how much electrical pressure is on a pin relative to ground. Key position follows the playbook (often ignition on, engine off).",
			Warning: "Use DC volts (solid line), not AC (wavy line). Black goes on a ground the playbook named (battery negative or chassis). Red goes on the test pin. This card does not pick the pin.",
			Words:   []string{"volt", "voltage", "vdc", "dc volt"},
			Plates: []seedPlate{
				{"Leads: same holes as ohms", "Black in COM. Red in VΩmA. The 10 A jack stays empty. Voltage on the 10 A jack can blow the meter’s fuse or read nonsense.", "meter-volts-jacks.jpg", "Black in COM, red in the voltage jack"},
				{"Dial to V with a solid line", "That is DC. The wavy line is AC mains — wrong for a 12 V car. Auto-range is fine. Manual-range: 20 V DC covers battery and 5 V sensor supplies.", "meter-volts-dial.jpg", "Dial on DC volts, not AC"},
				{"Black on ground, then red on the pin", "Clip or hold black on battery negative or clean chassis (paint is an insulator). Red on the pin or backprobe the playbook named. Hold still. Typical pictures: about 12–14 V is battery, about 5 V is often a sensor supply, near 0 V is ground or a dead feed. OL means the probe is not on metal. Write the number you see in the finding — do not invent a spec.", "meter-volts-display.jpg", "Meter showing about 12 volts with probes on a battery"},
			},
		},
	}
}

// SeedHowTos inserts the five shop-skill cards if those slugs are missing. Admin edits are kept.
func (s *Store) SeedHowTos(ctx context.Context) error {
	for _, g := range defaultHowToSeeds() {
		body := SanitizeHowToHTML(seedGuideHTML(g.Plates))
		if _, err := s.pool.Exec(ctx, `
			INSERT INTO howto_guides (slug, title, blurb, warning, body_html, match_words, published)
			VALUES ($1, $2, $3, $4, $5, $6, TRUE)
			ON CONFLICT (slug) DO NOTHING
		`, g.Slug, g.Title, g.Blurb, g.Warning, body, normalizeMatchWords(g.Words)); err != nil {
			return err
		}
	}
	return nil
}
