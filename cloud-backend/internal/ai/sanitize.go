package ai

import (
	"fmt"
	"strings"
)

var allowedEvidence = []string{
	"ledger:", "network:", "resolution:", "session:", "dtc:", "live:", "vehicle:", "gap:",
}

func Sanitize(p Playbook, allowedFigures map[string]struct{}) Playbook {
	if p.Lookouts == nil {
		p.Lookouts = []Lookout{}
	}
	if p.LikelyCauses == nil {
		p.LikelyCauses = []Cause{}
	}
	if p.Steps == nil {
		p.Steps = []Step{}
	}
	if p.Gaps == nil {
		p.Gaps = []string{}
	}

	outLooks := make([]Lookout, 0, len(p.Lookouts))
	for _, l := range p.Lookouts {
		l.Text = strings.TrimSpace(l.Text)
		l.Evidence = filterEvidence(l.Evidence)
		if l.Text == "" {
			continue
		}
		if len(l.Evidence) == 0 {
			p.Gaps = append(p.Gaps, "Dropped uncited lookout: "+clip(l.Text, 80))
			continue
		}
		outLooks = append(outLooks, l)
	}
	p.Lookouts = outLooks

	outCauses := make([]Cause, 0, len(p.LikelyCauses))
	for _, c := range p.LikelyCauses {
		c.Title = strings.TrimSpace(c.Title)
		c.Evidence = filterEvidence(c.Evidence)
		if c.Probability < 0 {
			c.Probability = 0
		}
		if c.Probability > 1 {
			c.Probability = 1
		}
		if c.Title == "" || len(c.Evidence) == 0 {
			if c.Title != "" {
				p.Gaps = append(p.Gaps, "Dropped uncited cause: "+c.Title)
			}
			continue
		}
		outCauses = append(outCauses, c)
	}
	p.LikelyCauses = outCauses

	outSteps := make([]Step, 0, len(p.Steps))
	for i, st := range p.Steps {
		st.Title = strings.TrimSpace(st.Title)
		st.Detail = strings.TrimSpace(st.Detail)
		st.Kind = strings.ToLower(strings.TrimSpace(st.Kind))
		if st.Kind == "" {
			st.Kind = "test"
		}
		if st.Order == 0 {
			st.Order = i + 1
		}
		kept := make([]string, 0, len(st.Figures))
		for _, fig := range st.Figures {
			fig = strings.TrimSpace(fig)
			if _, ok := allowedFigures[fig]; ok {
				kept = append(kept, fig)
			} else if fig != "" {
				p.Gaps = append(p.Gaps, "No diagram on file (removed uncited figure).")
			}
		}
		st.Figures = kept
		if st.Title == "" {
			continue
		}
		outSteps = append(outSteps, st)
	}
	p.Steps = outSteps

	if len(allowedFigures) == 0 {
		p.Gaps = appendUnique(p.Gaps, "No access figure on file for this body.")
	}
	p.Validation = strings.TrimSpace(p.Validation)
	return p
}

func filterEvidence(in []string) []string {
	out := make([]string, 0, len(in))
	for _, e := range in {
		e = strings.TrimSpace(e)
		ok := false
		low := strings.ToLower(e)
		for _, p := range allowedEvidence {
			if strings.HasPrefix(low, p) {
				ok = true
				break
			}
		}
		if ok {
			out = append(out, e)
		}
	}
	return out
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func appendUnique(in []string, msg string) []string {
	for _, g := range in {
		if g == msg {
			return in
		}
	}
	return append(in, msg)
}

func platformKey(makeName, model string, year int, engine string) string {
	parts := []string{strings.ToLower(strings.TrimSpace(makeName)), strings.ToLower(strings.TrimSpace(model))}
	if year > 0 {
		parts = append(parts, fmt.Sprintf("%d-%d", year-2, year+2))
	}
	if strings.TrimSpace(engine) != "" {
		parts = append(parts, strings.ToLower(strings.TrimSpace(engine)))
	}
	return strings.Join(parts, " ")
}
