package ai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"mechazone/cloud-backend/internal/ledger"
)

// Sidecar is JSON next to a PDF or HTML tree. Ingest uses it for make/model/year — not as a generated diagram.
type Sidecar struct {
	Title    string `json:"title"`
	Kind     string `json:"kind"`
	Make     string `json:"make"`
	Model    string `json:"model"`
	YearFrom int    `json:"year_from"`
	YearTo   int    `json:"year_to"`
	Engine   string `json:"engine"`
	Language  string `json:"language"`
	HTMLRoot  string `json:"html_root"`
	ImageRoot string `json:"image_root"`
}

type IngestResult struct {
	Path     string
	SourceID string
	Language string
	Chunks   int
	Figures  int
	Skipped  string
}

func (f *Fuser) IngestDir(ctx context.Context, dir string, translate bool) ([]IngestResult, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := []IngestResult{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name())
		ext := strings.ToLower(filepath.Ext(e.Name()))
		switch {
		case ext == ".pdf":
			res, err := f.IngestPDF(ctx, path, translate)
			if err != nil {
				return out, fmt.Errorf("%s: %w", e.Name(), err)
			}
			out = append(out, res)
		case ext == ".json" && !strings.Contains(e.Name(), ".example."):
			side, err := loadSidecar(path)
			if err != nil {
				return out, err
			}
			if strings.TrimSpace(side.HTMLRoot) == "" {
				continue
			}
			root := side.HTMLRoot
			if !filepath.IsAbs(root) {
				root = filepath.Join(dir, root)
			}
			if side.ImageRoot != "" && !filepath.IsAbs(side.ImageRoot) {
				side.ImageRoot = filepath.Join(dir, side.ImageRoot)
			}
			res, err := f.IngestHTMLTree(ctx, root, side, translate)
			if err != nil {
				return out, fmt.Errorf("%s: %w", e.Name(), err)
			}
			out = append(out, res)
		}
	}
	return out, nil
}

func (f *Fuser) IngestPDF(ctx context.Context, path string, translate bool) (IngestResult, error) {
	sum, err := fileSHA(path)
	if err != nil {
		return IngestResult{}, err
	}
	side, err := loadSidecar(path)
	if err != nil {
		return IngestResult{}, err
	}
	if side.Make == "" || side.Model == "" || side.YearFrom == 0 || side.YearTo == 0 {
		parsed, ok := parseFilename(filepath.Base(path))
		if !ok && (side.Make == "" || side.Model == "") {
			return IngestResult{}, fmt.Errorf("need a sidecar JSON (make, model, year_from, year_to) or a name like toyota_avensis_2009-2012_3zr-fae_de.pdf")
		}
		if side.Make == "" {
			side.Make = parsed.Make
		}
		if side.Model == "" {
			side.Model = parsed.Model
		}
		if side.YearFrom == 0 {
			side.YearFrom = parsed.YearFrom
		}
		if side.YearTo == 0 {
			side.YearTo = parsed.YearTo
		}
		if side.Engine == "" {
			side.Engine = parsed.Engine
		}
		if side.Language == "" {
			side.Language = parsed.Language
		}
	}
	if side.Kind == "" {
		side.Kind = "workshop_manual"
	}
	if side.Title == "" {
		side.Title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	pages, err := ExtractPDFPages(path)
	if err != nil {
		return IngestResult{}, err
	}
	sample := pages[0].Text
	if len(pages) > 1 {
		sample += "\n" + pages[1].Text
	}
	if side.Language == "" {
		side.Language = DetectLanguage(sample)
	}

	lites := ChunkPages(pages, 1400)
	chunks := make([]ledger.DocChunkIn, 0, len(lites))
	figs := []ledger.DocFigureIn{}
	seenFig := map[string]struct{}{}
	for _, lite := range lites {
		bodyEN := ""
		if translate && side.Language != "en" && f.LLM != nil {
			bodyEN, err = f.translateChunk(ctx, lite.Body, side.Language)
			if err != nil {
				return IngestResult{}, err
			}
		}
		chunks = append(chunks, ledger.DocChunkIn{
			Page:     lite.Page,
			Language: side.Language,
			Body:     lite.Body,
			BodyEN:   bodyEN,
			Codes:    lite.Codes,
		})
		for _, cap := range ExtractFigureCaptions(lite.Body) {
			key := fmt.Sprintf("%d:%s", lite.Page, cap)
			if _, ok := seenFig[key]; ok {
				continue
			}
			seenFig[key] = struct{}{}
			figs = append(figs, ledger.DocFigureIn{Page: lite.Page, Caption: cap, Language: side.Language})
		}
	}

	id, err := f.Store.ReplaceDoc(ctx, ledger.DocSource{
		SHA256:   sum,
		Path:     path,
		Title:    side.Title,
		Kind:     side.Kind,
		Make:     side.Make,
		Model:    side.Model,
		YearFrom: side.YearFrom,
		YearTo:   side.YearTo,
		Engine:   side.Engine,
		Language: side.Language,
	}, chunks, figs)
	if err != nil {
		return IngestResult{}, err
	}
	return IngestResult{Path: path, SourceID: id, Language: side.Language, Chunks: len(chunks), Figures: len(figs)}, nil
}

func (f *Fuser) translateChunk(ctx context.Context, body, lang string) (string, error) {
	raw, err := f.LLM.ChatJSON(ctx,
		`Return JSON {"text":"..."} only. Translate the workshop text to English. Keep part numbers, DTCs, voltages, and pin IDs unchanged.`,
		"Source language: "+lang+"\n\n"+body)
	if err != nil {
		return "", err
	}
	var out struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.Text), nil
}

func loadSidecar(pdfPath string) (Sidecar, error) {
	return LoadSidecarFile(strings.TrimSuffix(pdfPath, filepath.Ext(pdfPath)) + ".json")
}

func LoadSidecarFile(sidePath string) (Sidecar, error) {
	b, err := os.ReadFile(sidePath)
	if err != nil {
		if os.IsNotExist(err) {
			return Sidecar{}, nil
		}
		return Sidecar{}, err
	}
	var s Sidecar
	if err := json.Unmarshal(b, &s); err != nil {
		return Sidecar{}, fmt.Errorf("sidecar %s: %w", sidePath, err)
	}
	return s, nil
}

func parseFilename(name string) (Sidecar, bool) {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	parts := strings.Split(base, "_")
	if len(parts) < 3 {
		return Sidecar{}, false
	}
	s := Sidecar{Make: titleish(parts[0]), Model: titleish(parts[1])}
	rest := parts[2:]
	if len(rest) > 0 && strings.Contains(rest[0], "-") {
		a, b, _ := strings.Cut(rest[0], "-")
		s.YearFrom, _ = strconv.Atoi(a)
		s.YearTo, _ = strconv.Atoi(b)
		rest = rest[1:]
	}
	if len(rest) > 0 && len(rest[0]) > 3 {
		s.Engine = strings.ToUpper(strings.ReplaceAll(rest[0], "-", "-"))
		rest = rest[1:]
	}
	if len(rest) > 0 && len(rest[0]) <= 3 {
		s.Language = strings.ToLower(rest[0])
	}
	return s, s.Make != "" && s.Model != "" && s.YearFrom > 0
}

func titleish(s string) string {
	s = strings.ReplaceAll(s, "-", " ")
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

func fileSHA(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
