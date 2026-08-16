package ai

import "testing"

func TestSanitizeDropsUncitedAndInventedFigures(t *testing.T) {
	p := Sanitize(Playbook{
		Lookouts: []Lookout{
			{Text: "Connector already cleaned on this VIN", Evidence: []string{"resolution:abc"}},
			{Text: "Typical Toyota issue", Evidence: []string{}},
		},
		LikelyCauses: []Cause{
			{Title: "Guess", Probability: 0.9, Evidence: []string{"my-brain"}},
			{Title: "Valvematic circuit", Probability: 1.4, Evidence: []string{"ledger:resolution:abc"}},
		},
		Steps: []Step{
			{Title: "Read angle", Kind: "test", Figures: []string{"figure:made-up"}},
		},
	}, map[string]struct{}{})
	if len(p.Lookouts) != 1 {
		t.Fatalf("lookouts %+v", p.Lookouts)
	}
	if len(p.LikelyCauses) != 1 || p.LikelyCauses[0].Probability != 1 {
		t.Fatalf("causes %+v", p.LikelyCauses)
	}
	if len(p.Steps[0].Figures) != 0 {
		t.Fatalf("figures should be stripped")
	}
	found := false
	for _, g := range p.Gaps {
		if g == "No access figure on file for this body." {
			found = true
		}
	}
	if !found {
		t.Fatalf("gaps %+v", p.Gaps)
	}
}
