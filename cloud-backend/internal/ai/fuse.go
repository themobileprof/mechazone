package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"mechazone/cloud-backend/internal/ledger"
	"mechazone/cloud-backend/internal/vin"
)

type Fuser struct {
	LLM   *Client
	Store *ledger.Store
	Log   *slog.Logger
}

func (f *Fuser) Build(ctx context.Context, req Request) (Playbook, error) {
	norm, err := vin.Normalize(req.VIN)
	if err != nil {
		return Playbook{}, err
	}
	req.VIN = norm
	if len(req.ActiveCodes) == 0 && req.SessionID == "" {
		return Playbook{}, fmt.Errorf("active_codes or session_id is required")
	}

	hist, err := f.Store.History(ctx, req.VIN)
	if err != nil {
		return Playbook{}, fmt.Errorf("ledger: %w", err)
	}
	if hist.Vehicle != nil {
		if req.Make == "" {
			req.Make = hist.Vehicle.Make
		}
		if req.Model == "" {
			req.Model = hist.Vehicle.Model
		}
		if req.Year == 0 {
			req.Year = hist.Vehicle.Year
		}
	}
	if req.SessionID != "" {
		sess, err := f.Store.SessionByID(ctx, req.SessionID)
		if err != nil {
			return Playbook{}, fmt.Errorf("session: %w", err)
		}
		if sess.VIN != req.VIN {
			return Playbook{}, fmt.Errorf("session VIN does not match")
		}
		if len(req.ActiveCodes) == 0 {
			req.ActiveCodes = sess.ActiveCodes
		}
		if len(req.FreezeFrame) == 0 {
			req.FreezeFrame = sess.FreezeFrame
		}
		if req.AdapterType == "" {
			req.AdapterType = sess.AdapterType
		}
		if req.Protocol == "" {
			req.Protocol = sess.Protocol
		}
	}

	matches, err := f.Store.NetworkMatches(ctx, req.VIN, req.Make, req.Model, req.Year, req.ActiveCodes)
	if err != nil {
		return Playbook{}, fmt.Errorf("network: %w", err)
	}
	titles, err := f.Store.DTCTitles(ctx, req.ActiveCodes)
	if err != nil {
		return Playbook{}, fmt.Errorf("dtc: %w", err)
	}

	q := strings.TrimSpace(strings.Join(append(req.ActiveCodes, req.EngineHint, req.Make, req.Model), " "))
	docs, figs, err := f.Store.SearchManuals(ctx, req.Make, req.Model, req.Year, req.ActiveCodes, q)
	if err != nil {
		return Playbook{}, fmt.Errorf("manuals: %w", err)
	}
	allowedFigures := map[string]struct{}{}
	for _, fig := range figs {
		allowedFigures["figure:"+fig.ID] = struct{}{}
	}

	if req.Language == "" {
		req.Language = "en"
	}

	user, err := buildUserPrompt(req, hist, matches, titles, docs, figs)
	if err != nil {
		return Playbook{}, err
	}
	raw, err := f.LLM.ChatJSON(ctx, systemPrompt, user)
	if err != nil {
		return Playbook{}, err
	}
	var book Playbook
	if err := json.Unmarshal(raw, &book); err != nil {
		return Playbook{}, fmt.Errorf("playbook json: %w", err)
	}
	book.VIN = req.VIN
	book.Platform = platformKey(req.Make, req.Model, req.Year, req.EngineHint)
	book.FirstSeen = hist.FirstSeen
	book.Model = f.LLM.Model
	book = Sanitize(book, allowedFigures)
	for _, fig := range figs {
		book.ManualFigures = append(book.ManualFigures, ManualFigure{
			ID: fig.ID, Title: fig.Title, Page: fig.Page, Caption: fig.Caption, Language: fig.Language,
			ImageURL: fig.ImageURL, OCRText: fig.OCRText,
		})
	}
	return book, nil
}

const systemPrompt = `You are a senior diagnostic engineer writing a shop-floor playbook for ONE vehicle.
Return ONLY a JSON object with keys:
lookouts (array of {text, evidence}),
likely_causes (array of {title, probability 0-1, evidence}),
steps (array of {order, kind: test|access|inspect, title, detail, pass, fail, adapter, figures}),
validation (string),
gaps (array of strings).

Rules:
- History on THIS VIN first, then same-platform network matches. Never invent a repair from a code letter.
- Every lookout and cause MUST cite evidence using only these prefixes: ledger:, resolution:<id>, network:<id>, session:<id>, dtc:<code>, live:<name>, vehicle:decode, doc:<id>, figure:<id>.
- Manual chunks may be in another language. Use them. Do not discard a procedure because it is not English.
- Write lookouts, steps, and validation in the requested playbook language.
- Do not invent pin numbers, voltages, or access steps that are not in the provided context. Put missing facts in gaps.
- figures may only list figure:<id> values that appear in retrieved.figures. Never generate a drawing.
- Prefer OpenPort/UDS tests and a shop multimeter. No extra instruments.
- If this VIN already had a fix for the same code, say so in lookouts and do not lead with that same part swap.
- If first_seen is true, say so in gaps and use network + live data only.
- No customer names, phones, or plates.`

func buildUserPrompt(req Request, hist ledger.History, matches []ledger.NetworkMatch, titles map[string]ledger.DTC, docs []ledger.RetrievedChunk, figs []ledger.RetrievedFigure) (string, error) {
	type payload struct {
		PlaybookLanguage string                   `json:"playbook_language"`
		Vehicle          any                      `json:"vehicle"`
		FirstSeen        bool                     `json:"first_seen"`
		LiveScan         Request                  `json:"live_scan"`
		DTCTitles        map[string]ledger.DTC    `json:"dtc_titles"`
		VINLedger        ledger.History           `json:"vin_ledger"`
		Network          []ledger.NetworkMatch    `json:"network_matches"`
		Retrieved        struct {
			Docs    []ledger.RetrievedChunk  `json:"docs"`
			Figures []ledger.RetrievedFigure `json:"figures"`
		} `json:"retrieved"`
	}
	p := payload{
		PlaybookLanguage: req.Language,
		FirstSeen:        hist.FirstSeen,
		LiveScan:         req,
		DTCTitles:        titles,
		VINLedger:        hist,
		Network:          matches,
	}
	if hist.Vehicle != nil {
		p.Vehicle = hist.Vehicle
	}
	p.Retrieved.Docs = docs
	p.Retrieved.Figures = figs
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.WriteString("Build the playbook from this gathered context. Cite retrieved docs as doc:<id> and figures as figure:<id>. If retrieved.docs is empty, do not invent manual text.\n\n")
	sb.Write(b)
	return sb.String(), nil
}
