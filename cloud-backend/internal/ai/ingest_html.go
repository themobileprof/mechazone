package ai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"mechazone/cloud-backend/internal/ledger"
)

func (f *Fuser) IngestHTMLTree(ctx context.Context, root string, side Sidecar, translate bool) (IngestResult, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return IngestResult{}, err
	}
	if side.Make == "" || side.Model == "" || side.YearFrom == 0 || side.YearTo == 0 {
		return IngestResult{}, fmt.Errorf("HTML tree needs a sidecar with make, model, year_from, year_to")
	}
	if side.Kind == "" {
		side.Kind = "workshop_manual"
	}
	if side.Title == "" {
		side.Title = filepath.Base(root)
	}
	if side.ImageRoot == "" {
		if sib := filepath.Join(filepath.Dir(root), "spanish"); dirExists(sib) {
			side.ImageRoot = sib
		} else if sib := filepath.Join(filepath.Dir(filepath.Dir(root)), "spanish"); dirExists(sib) {
			side.ImageRoot = sib
		}
	}
	if side.ImageRoot != "" {
		side.ImageRoot, _ = filepath.Abs(side.ImageRoot)
	}

	ocr := loadOCRGlossary(root)

	var files []string
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := strings.ToLower(d.Name())
			if name == ".git" || strings.HasPrefix(name, ".venv") || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".html" && ext != ".htm" {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if shouldSkipHTML(rel) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return IngestResult{}, err
	}
	if len(files) == 0 {
		return IngestResult{}, fmt.Errorf("no content HTML under %s", root)
	}

	sample := ""
	chunks := []ledger.DocChunkIn{}
	figs := []ledger.DocFigureIn{}
	seenImg := map[string]struct{}{}
	page := 0
	for _, path := range files {
		parsed, err := ParseHTMLFile(path, root)
		if err != nil || utf8.RuneCountInString(parsed.Text) < 80 {
			continue
		}
		page++
		if sample == "" {
			sample = parsed.Text
		}
		body := parsed.Text
		ocrBits := []string{}
		for _, im := range parsed.Imgs {
			if t := ocr[im.RelKey]; strings.TrimSpace(t) != "" {
				ocrBits = append(ocrBits, t)
			}
		}
		if len(ocrBits) > 0 {
			body += "\n\n[figure text] " + strings.Join(ocrBits, " | ")
		}
		bodyEN := ""
		if translate && side.Language != "" && side.Language != "en" && f.LLM != nil {
			bodyEN, err = f.translateChunk(ctx, body, side.Language)
			if err != nil {
				return IngestResult{}, err
			}
		}
		chunks = append(chunks, ledger.DocChunkIn{
			Page:     page,
			Language: side.Language,
			Body:     body,
			BodyEN:   bodyEN,
			Codes:    ExtractCodes(body),
			RelPath:  parsed.Rel,
		})
		for _, im := range parsed.Imgs {
			if _, ok := seenImg[im.Abs]; ok {
				continue
			}
			seenImg[im.Abs] = struct{}{}
			cap := im.Alt
			if o := strings.TrimSpace(ocr[im.RelKey]); o != "" {
				if cap != "" {
					cap = cap + " — " + clip(o, 160)
				} else {
					cap = clip(o, 160)
				}
			}
			figs = append(figs, ledger.DocFigureIn{
				Page:      page,
				Caption:   cap,
				Language:  side.Language,
				ImagePath: im.Abs,
				OCRText:   strings.TrimSpace(ocr[im.RelKey]),
			})
		}
	}
	if side.Language == "" {
		side.Language = DetectLanguage(sample)
		for i := range chunks {
			if chunks[i].Language == "" {
				chunks[i].Language = side.Language
			}
		}
	}
	if len(chunks) == 0 {
		return IngestResult{}, fmt.Errorf("no usable HTML pages in %s", root)
	}

	sum := hashTreeID(root, side)
	id, err := f.Store.ReplaceDoc(ctx, ledger.DocSource{
		SHA256:    sum,
		Path:      root,
		Title:     side.Title,
		Kind:      side.Kind,
		Make:      side.Make,
		Model:     side.Model,
		YearFrom:  side.YearFrom,
		YearTo:    side.YearTo,
		Engine:    side.Engine,
		Language:  side.Language,
		ImageRoot: side.ImageRoot,
	}, chunks, figs)
	if err != nil {
		return IngestResult{}, err
	}
	return IngestResult{Path: root, SourceID: id, Language: side.Language, Chunks: len(chunks), Figures: len(figs)}, nil
}

func loadOCRGlossary(root string) map[string]string {
	out := map[string]string{}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.EqualFold(d.Name(), "ocr_by_image.json") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var raw map[string]string
		if json.Unmarshal([]byte(DecodeToUTF8(b)), &raw) != nil {
			return nil
		}
		for k, v := range raw {
			k = strings.TrimPrefix(filepath.ToSlash(k), "/")
			out[k] = CleanUTF8(v)
			out[filepath.Base(k)] = CleanUTF8(v)
		}
		return nil
	})
	return out
}

func hashTreeID(root string, side Sidecar) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s\n%s\n%s\n%s\n%d-%d\n", root, side.Title, side.Make, side.Model, side.YearFrom, side.YearTo)
	return hex.EncodeToString(h.Sum(nil))
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
