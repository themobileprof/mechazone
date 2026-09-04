// Package ai fuses a live or imported scan with this shop's jobs and retrieved manuals.
// It calls a hosted LLM API. It must not invent pins, voltages, or diagrams.
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
	Embed *Embedder
	Store *ledger.Store
	Log   *slog.Logger
}

// Build loads this shop's VIN history and retrieved chunks, then asks the hosted LLM.
func (f *Fuser) Build(ctx context.Context, req Request) (Playbook, error) {
	norm, err := vin.Normalize(req.VIN)
	if err != nil {
		return Playbook{}, err
	}
	req.VIN = norm
	if len(req.ActiveCodes) == 0 && req.SessionID == "" && len(req.Live) == 0 && len(req.Modules) == 0 {
		return Playbook{}, fmt.Errorf("a live scan is required")
	}

	hist, err := f.Store.History(ctx, req.VIN, req.ShopID, req.TechnicianID)
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
		if !ledger.InShopScope(req.ShopID, sess.ShopID, req.TechnicianID, sess.TechnicianID) {
			return Playbook{}, fmt.Errorf("session is not this shop's job")
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

	matches, err := f.Store.NetworkMatches(ctx, req.VIN, req.ShopID, req.TechnicianID, req.Make, req.Model, req.Year, req.ActiveCodes)
	if err != nil {
		return Playbook{}, fmt.Errorf("network: %w", err)
	}
	titles, err := f.Store.DTCTitles(ctx, req.ActiveCodes)
	if err != nil {
		return Playbook{}, fmt.Errorf("dtc: %w", err)
	}
	titleText := map[string]string{}
	for code, d := range titles {
		titleText[code] = d.Title
	}
	classes := ClassifyCodes(req.ActiveCodes, titleText)
	net := InferNetwork(req.Modules)
	wiring := WiringShaped(classes)

	var pinned *PinnedManual
	if strings.TrimSpace(req.SourceID) != "" {
		src, err := f.Store.GetManual(ctx, req.SourceID)
		if err != nil {
			return Playbook{}, fmt.Errorf("manual: %w", err)
		}
		pinned = &PinnedManual{
			ID: src.ID, Title: src.Title, Make: src.Make, Model: src.Model,
			YearFrom: src.YearFrom, YearTo: src.YearTo, Engine: src.Engine, Language: src.Language,
			Chunks: src.Chunks, Figures: src.Figures,
		}
		if req.Make == "" {
			req.Make = src.Make
		}
		if req.Model == "" {
			req.Model = src.Model
		}
		if req.Year == 0 {
			req.Year = src.YearFrom
		}
		if req.EngineHint == "" {
			req.EngineHint = src.Engine
		}
	}
	q := retrievalQuery(req)
	mq := ledger.ManualQuery{
		Make: req.Make, Model: req.Model, Year: req.Year,
		Codes: req.ActiveCodes, Query: q, Wiring: wiring, SourceID: req.SourceID,
	}
	if f.Embed != nil && f.Embed.Ready() && q != "" && f.Store.HasChunkEmbeddings() {
		meta, err := f.Store.EmbeddingMeta(ctx)
		if err != nil {
			return Playbook{}, fmt.Errorf("embed meta: %w", err)
		}
		if meta.Model == "" {
			if f.Log != nil {
				f.Log.Warn("chunk vectors have no recorded model; FTS only")
			}
		} else if !meta.MatchesIndex() {
			if f.Log != nil {
				f.Log.Error("cosine skipped", "err", ledger.EmbedModelMismatch(meta))
			}
		} else {
			vecs, err := f.Embed.Embed(ctx, []string{f.Embed.QueryText(q)})
			if err != nil {
				if f.Log != nil {
					f.Log.Warn("embed query failed; FTS only", "err", err)
				}
			} else if len(vecs) == 1 {
				mq.Embedding = vecs[0]
			}
		}
	}
	docs, figs, err := f.Store.SearchManuals(ctx, mq)
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

	user, err := buildUserPrompt(req, hist, matches, titles, docs, figs, classes, net, wiring, settledChecks(hist.Checks))
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
	if strings.EqualFold(req.AdapterType, "imported_report") {
		gap := "Codes came from an imported scanner report, not this OpenPort. Confirm with a live scan before treating DIDs or module maps as fact."
		already := false
		for _, g := range book.Gaps {
			if g == gap {
				already = true
				break
			}
		}
		if !already {
			book.Gaps = append(StringList{gap}, book.Gaps...)
		}
	}
	book.CircuitClasses = classes
	book.Network = net
	book.Manual = pinned
	book.RetrievedChunks = len(docs)
	if pinned != nil && len(docs) == 0 {
		book.Gaps = appendUnique(book.Gaps, "This workshop book is on file but no page matched this scan. Adapter tests still apply.")
	}
	for _, fig := range figs {
		book.ManualFigures = append(book.ManualFigures, ManualFigure{
			ID: fig.ID, Title: fig.Title, Page: fig.Page, Caption: fig.Caption, Language: fig.Language,
			ImageURL: fig.ImageURL, OCRText: fig.OCRText, Kind: fig.Kind,
		})
	}
	if err := f.Store.EnsureVehicle(ctx, req.VIN, req.Make, req.Model, req.Year, "playbook"); err != nil {
		return Playbook{}, fmt.Errorf("vehicle: %w", err)
	}
	seeds := make([]ledger.PlaybookStepSeed, 0, len(book.Steps))
	for _, st := range book.Steps {
		seeds = append(seeds, ledger.PlaybookStepSeed{Kind: st.Kind, Title: st.Title, Detail: st.Detail})
	}
	checks, err := f.Store.SyncPlaybookSteps(ctx, req.VIN, req.ShopID, req.TechnicianID, seeds)
	if err != nil {
		return Playbook{}, fmt.Errorf("checks: %w", err)
	}
	book.Checks = checks
	return book, nil
}

func retrievalQuery(req Request) string {
	parts := append([]string{}, req.ActiveCodes...)
	parts = append(parts, req.EngineHint, req.Make, req.Model)
	for _, live := range req.Live {
		if n := strings.TrimSpace(live.Name); n != "" {
			parts = append(parts, n)
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

const systemPrompt = `You are a senior diagnostic engineer writing a shop-floor playbook for ONE vehicle.
Return ONLY a JSON object with keys:
lookouts (array of {text: string, evidence: array of strings}),
likely_causes (array of {title: string, probability: 0-1, evidence: array of strings}),
steps (array of {order: number, kind: test|access|inspect, title: string, detail: string, pass: string, fail: string, adapter: boolean, figures: array of strings}),
validation (string),
gaps (array of strings).

evidence MUST be a JSON array of strings, never a single string. Example: "evidence": ["dtc:P0171", "module:ECM"].

Rules:
- Use this shop's jobs on THIS vehicle first (what was done, parts, closeouts). That record stays in this shop — it is not a public VIN history.
- Then this shop's similar jobs on other vehicles of the same platform. Never treat another shop's work, or another VIN, as this car's history.
- Every lookout and cause MUST cite evidence using only these prefixes: ledger:, resolution:<id>, network:<id>, session:<id>, dtc:<code>, live:<name>, vehicle:decode, doc:<id>, figure:<id>, module:<name>, check:.
- Manual chunks may be in another language. Use them. Do not discard a procedure because it is not English.
- Write lookouts, steps, and validation in the requested playbook language.
- Do not invent pin numbers, voltages, or access steps that are not in the provided context. Put missing facts in gaps.
- figures may only list figure:<id> values that appear in retrieved.figures. Never generate a drawing.
- Prefer OpenPort/UDS tests and a shop multimeter. No extra instruments.
- If circuit_classes mark a code as open/short/lost_communication/bus_off, treat it as wiring/network first, not a failed sensor. Lead with the module map (who is dark), then retrieved EWD/connector figures, then a meter test cited from those chunks, then a DID wiggle on the OpenPort. Do not invent pin numbers.
- If network.reading is backbone, check DLC power/ground/CAN before a single module. If branch, stay on the silent confirmed node.
- There are no captured UDS $2F IO-control IDs on this profile. Do not invent actuator commands. Put that in gaps if an output test would help.
- If this shop already closed the same code on this vehicle, say so in lookouts and do not lead with that same part swap.
- bay_checks are tests this shop already ran or ruled out on this VIN. status done = they performed it (use note as the finding). status ruled_out = they are sure this is not the fault. Do not lead with a ruled_out step unless live data contradicts it. After a done check, pick the next test from the finding — do not repeat the same step.
- If there are no DTCs, still advise from live DIDs, the module map, and this shop's jobs. Lookouts are suspected challenges (repeats, wiring, missing nodes), not a code dump.
- If live_scan.adapter_type is imported_report, the codes were typed from another scanner's file. Do not invent live DIDs, module tx/rx, or freeze-frame. Cite session:<id>. Put the missing live OpenPort scan in gaps.
- No customer names, phones, or plates.`

func settledChecks(in []ledger.PlaybookCheck) []ledger.PlaybookCheck {
	out := make([]ledger.PlaybookCheck, 0, len(in))
	for _, c := range in {
		if c.Status == ledger.CheckDone || c.Status == ledger.CheckRuledOut {
			out = append(out, c)
		}
	}
	return out
}

func buildUserPrompt(req Request, hist ledger.History, matches []ledger.NetworkMatch, titles map[string]ledger.DTC, docs []ledger.RetrievedChunk, figs []ledger.RetrievedFigure, classes []CircuitClass, net NetworkHint, wiring bool, checks []ledger.PlaybookCheck) (string, error) {
	type payload struct {
		PlaybookLanguage string                 `json:"playbook_language"`
		Vehicle          any                    `json:"vehicle"`
		FirstSeen        bool                   `json:"first_seen"`
		WiringShaped     bool                   `json:"wiring_shaped"`
		CircuitClasses   []CircuitClass         `json:"circuit_classes"`
		Network          NetworkHint            `json:"network"`
		LiveScan         Request                `json:"live_scan"`
		DTCTitles        map[string]ledger.DTC  `json:"dtc_titles"`
		ShopWork         ledger.History         `json:"shop_work"`
		BayChecks        []ledger.PlaybookCheck `json:"bay_checks"`
		ShopPlatformJobs []ledger.NetworkMatch  `json:"shop_platform_jobs"`
		Retrieved        struct {
			Docs    []ledger.RetrievedChunk  `json:"docs"`
			Figures []ledger.RetrievedFigure `json:"figures"`
		} `json:"retrieved"`
	}
	p := payload{
		PlaybookLanguage: req.Language,
		FirstSeen:        hist.FirstSeen,
		WiringShaped:     wiring,
		CircuitClasses:   classes,
		Network:          net,
		LiveScan:         req,
		DTCTitles:        titles,
		ShopWork:         hist,
		BayChecks:        checks,
		ShopPlatformJobs: matches,
	}
	p.ShopWork.Customer = nil
	p.ShopWork.Capture = nil
	p.ShopWork.Checks = nil
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
	sb.WriteString("Build the playbook from this gathered context. Always produce lookouts and next tests. shop_work is this shop's jobs on this vehicle only. bay_checks are tests this shop already performed or ruled out on this VIN — iterate from those findings, do not restart the same dead end. shop_platform_jobs are this shop's similar repairs on other cars (no VIN). Cite retrieved docs as doc:<id> and figures as figure:<id>. If retrieved.docs is empty, do not invent manual text. Lookouts are suspected challenges from the live scan (codes, dark modules, odd DIDs) plus this shop's jobs (repeats, parts already replaced) and bay_checks. If live_scan.active_codes is empty, still advise from live DIDs, the module map, and shop_work — do not return an empty playbook. If live_scan.adapter_type is imported_report, do not treat it as an OpenPort capture.\n\n")
	sb.Write(b)
	return sb.String(), nil
}
