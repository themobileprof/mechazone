package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

const maxAskQuestionRunes = 2000
const maxAskTurns = 8

const askSystemPrompt = `You are a senior diagnostic engineer on the shop floor. A technician is stuck on ONE playbook step for ONE vehicle.
Return ONLY a JSON object with keys:
answer (string),
gaps (array of strings),
figures (array of strings).

Rules:
- Answer the technician's question about focus_step. Do not rewrite the whole playbook.
- Use this shop's jobs on THIS vehicle, bay_checks, retrieved docs, and the live scan. Never treat another shop's file as this car's history.
- Do not invent pin numbers, voltages, access steps, or actuator commands. If a fact is not in the provided context, put it in gaps and say so in the answer.
- figures may only list figure:<id> values that appear in retrieved.figures. Never generate a drawing.
- Manual chunks may be in another language. Use them.
- Write the answer in the requested playbook language. Plain sentences. No markdown headings.
- If live_scan.adapter_type is imported_report, do not invent live DIDs or module maps.
- No customer names, phones, or plates.`

type askModelOut struct {
	Answer  string     `json:"answer"`
	Gaps    StringList `json:"gaps"`
	Figures StringList `json:"figures"`
}

// Ask answers a technician question about one playbook step using the same fuse context as Build.
func (f *Fuser) Ask(ctx context.Context, req AskRequest) (AskReply, error) {
	req.Question = strings.TrimSpace(req.Question)
	if req.Question == "" {
		return AskReply{}, fmt.Errorf("a question is required")
	}
	if utf8.RuneCountInString(req.Question) > maxAskQuestionRunes {
		return AskReply{}, fmt.Errorf("question is too long")
	}
	req.Step.Title = strings.TrimSpace(req.Step.Title)
	req.Step.Detail = strings.TrimSpace(req.Step.Detail)
	if req.Step.Title == "" {
		return AskReply{}, fmt.Errorf("a playbook step is required")
	}
	req.Thread = clipAskThread(req.Thread)
	req.Lookouts = clipLookouts(req.Lookouts)

	extra := strings.Join([]string{req.Step.Title, req.Step.Detail, req.Question}, " ")
	g, err := f.gather(ctx, req.Request, extra)
	if err != nil {
		return AskReply{}, err
	}
	req.Request = g.req

	user, err := buildAskPrompt(req, g)
	if err != nil {
		return AskReply{}, err
	}
	raw, err := f.LLM.ChatJSON(ctx, askSystemPrompt, user)
	if err != nil {
		return AskReply{}, err
	}
	var out askModelOut
	if err := json.Unmarshal(raw, &out); err != nil {
		return AskReply{}, fmt.Errorf("ask json: %w", err)
	}
	reply := sanitizeAsk(out, g)
	reply.Model = f.LLM.Model
	reply.RetrievedChunks = len(g.docs)
	return reply, nil
}

func buildAskPrompt(req AskRequest, g gathered) (string, error) {
	p := struct {
		PlaybookLanguage string         `json:"playbook_language"`
		Vehicle          any            `json:"vehicle"`
		FirstSeen        bool           `json:"first_seen"`
		WiringShaped     bool           `json:"wiring_shaped"`
		CircuitClasses   []CircuitClass `json:"circuit_classes"`
		Network          NetworkHint    `json:"network"`
		LiveScan         Request        `json:"live_scan"`
		DTCTitles        any            `json:"dtc_titles"`
		ShopWork         any            `json:"shop_work"`
		BayChecks        any            `json:"bay_checks"`
		ShopPlatformJobs any            `json:"shop_platform_jobs"`
		Retrieved        any            `json:"retrieved"`
		FocusStep        Step           `json:"focus_step"`
		Lookouts         []string       `json:"lookouts"`
		Question         string         `json:"question"`
		PriorTurns       []AskTurn      `json:"prior_turns"`
	}{
		PlaybookLanguage: req.Language,
		FirstSeen:        g.hist.FirstSeen,
		WiringShaped:     g.wiring,
		CircuitClasses:   g.classes,
		Network:          g.net,
		LiveScan:         req.Request,
		DTCTitles:        g.titles,
		BayChecks:        g.checks,
		ShopPlatformJobs: g.matches,
		FocusStep:        req.Step,
		Lookouts:         req.Lookouts,
		Question:         req.Question,
		PriorTurns:       req.Thread,
	}
	work := g.hist
	work.Customer = nil
	work.Capture = nil
	work.Checks = nil
	p.ShopWork = work
	if g.hist.Vehicle != nil {
		p.Vehicle = g.hist.Vehicle
	}
	p.Retrieved = struct {
		Docs    any `json:"docs"`
		Figures any `json:"figures"`
	}{Docs: g.docs, Figures: g.figs}

	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.WriteString("Answer the technician about focus_step only. shop_work is this shop's jobs on this vehicle. bay_checks are tests already performed or ruled out. Cite retrieved docs as doc:<id> and figures as figure:<id>. If retrieved.docs is empty, do not invent manual text. prior_turns are earlier questions on this same step.\n\n")
	sb.Write(b)
	return sb.String(), nil
}

func sanitizeAsk(out askModelOut, g gathered) AskReply {
	reply := AskReply{
		Answer: strings.TrimSpace(out.Answer),
		Gaps:   out.Gaps,
	}
	if reply.Gaps == nil {
		reply.Gaps = StringList{}
	}
	if reply.Answer == "" {
		reply.Gaps = appendUnique(reply.Gaps, "The model returned no answer.")
	}
	keptIDs := map[string]struct{}{}
	for _, fig := range out.Figures {
		fig = strings.TrimSpace(fig)
		if _, ok := g.allowedFigures[fig]; !ok {
			if fig != "" {
				reply.Gaps = appendUnique(reply.Gaps, "No diagram on file (removed uncited figure).")
			}
			continue
		}
		keptIDs[strings.TrimPrefix(fig, "figure:")] = struct{}{}
	}
	for _, fig := range g.figs {
		if _, ok := keptIDs[fig.ID]; ok {
			reply.Figures = append(reply.Figures, ManualFigure{
				ID: fig.ID, Title: fig.Title, Page: fig.Page, Caption: fig.Caption, Language: fig.Language,
				ImageURL: fig.ImageURL, OCRText: fig.OCRText, Kind: fig.Kind,
			})
		}
	}
	return reply
}

func clipAskThread(in []AskTurn) []AskTurn {
	out := make([]AskTurn, 0, maxAskTurns)
	for _, t := range in {
		role := strings.ToLower(strings.TrimSpace(t.Role))
		if role != "user" && role != "assistant" {
			continue
		}
		content := strings.TrimSpace(t.Content)
		if content == "" {
			continue
		}
		if utf8.RuneCountInString(content) > maxAskQuestionRunes {
			content = string([]rune(content)[:maxAskQuestionRunes])
		}
		out = append(out, AskTurn{Role: role, Content: content})
	}
	if len(out) > maxAskTurns {
		out = out[len(out)-maxAskTurns:]
	}
	return out
}

func clipLookouts(in []string) []string {
	out := make([]string, 0, 8)
	for _, t := range in {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if utf8.RuneCountInString(t) > 400 {
			t = string([]rune(t)[:400])
		}
		out = append(out, t)
		if len(out) == 8 {
			break
		}
	}
	return out
}
