package ledger

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type DocSource struct {
	ID       string
	SHA256   string
	Path     string
	Title    string
	Kind     string
	Make     string
	Model    string
	YearFrom int
	YearTo   int
	Engine    string
	Language  string
	ImageRoot string
}

type DocChunkIn struct {
	Page     int
	Language string
	Body     string
	BodyEN   string
	Codes    []string
	RelPath  string
}

type DocFigureIn struct {
	Page     int
	Caption  string
	Language  string
	ImagePath string
	OCRText   string
}

type RetrievedChunk struct {
	ID       string `json:"id"`
	SourceID string `json:"source_id"`
	Title    string `json:"title"`
	Page     int    `json:"page"`
	Language string `json:"language"`
	Body     string `json:"body"`
	BodyEN   string `json:"body_en,omitempty"`
	Codes    []string `json:"codes"`
}

type RetrievedFigure struct {
	ID       string `json:"id"`
	SourceID string `json:"source_id"`
	Title    string `json:"title"`
	Page     int    `json:"page"`
	Caption  string `json:"caption"`
	Language  string `json:"language"`
	HasImage  bool   `json:"has_image"`
	ImageURL  string `json:"image_url,omitempty"`
	OCRText   string `json:"ocr_text,omitempty"`
}

func (s *Store) ReplaceDoc(ctx context.Context, src DocSource, chunks []DocChunkIn, figures []DocFigureIn) (string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var id string
	err = tx.QueryRow(ctx, `SELECT id FROM doc_sources WHERE sha256 = $1`, src.SHA256).Scan(&id)
	if err == nil {
		if _, err := tx.Exec(ctx, `DELETE FROM doc_sources WHERE id = $1`, id); err != nil {
			return "", err
		}
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO doc_sources (sha256, path, title, kind, make, model, year_from, year_to, engine, language, image_root)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id
	`, src.SHA256, utf8Safe(src.Path), utf8Safe(src.Title), src.Kind, src.Make, src.Model, src.YearFrom, src.YearTo, src.Engine, src.Language, utf8Safe(src.ImageRoot)).Scan(&id)
	if err != nil {
		return "", err
	}
	for _, c := range chunks {
		if c.Codes == nil {
			c.Codes = []string{}
		}
		for i := range c.Codes {
			c.Codes[i] = utf8Safe(c.Codes[i])
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO doc_chunks (source_id, page, language, body, body_en, codes, rel_path)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
		`, id, c.Page, c.Language, utf8Safe(c.Body), utf8Safe(c.BodyEN), c.Codes, utf8Safe(c.RelPath)); err != nil {
			return "", err
		}
	}
	for _, f := range figures {
		if _, err := tx.Exec(ctx, `
			INSERT INTO doc_figures (source_id, page, caption, language, image_path, ocr_text)
			VALUES ($1,$2,$3,$4,$5,$6)
		`, id, f.Page, utf8Safe(f.Caption), f.Language, utf8Safe(f.ImagePath), utf8Safe(f.OCRText)); err != nil {
			return "", err
		}
	}
	return id, tx.Commit(ctx)
}

func (s *Store) SearchManuals(ctx context.Context, makeName, model string, year int, codes []string, query string) ([]RetrievedChunk, []RetrievedFigure, error) {
	makeName = strings.TrimSpace(makeName)
	model = strings.TrimSpace(model)
	if makeName == "" || model == "" {
		return []RetrievedChunk{}, []RetrievedFigure{}, nil
	}
	if codes == nil {
		codes = []string{}
	}
	query = strings.TrimSpace(query)
	if year <= 0 {
		year = 2000
	}
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.source_id, s.title, c.page, c.language, c.body, c.body_en, c.codes
		FROM doc_chunks c
		JOIN doc_sources s ON s.id = c.source_id
		WHERE lower(s.make) = lower($1)
		  AND lower(s.model) = lower($2)
		  AND ($3 = 2000 OR (s.year_from <= $3 AND s.year_to >= $3))
		  AND (
		        cardinality($4::text[]) = 0
		        OR c.codes && $4
		        OR ($5 <> '' AND c.tsv @@ plainto_tsquery('simple', $5))
		        OR $5 = ''
		      )
		ORDER BY (c.codes && $4) DESC,
		         CASE WHEN $5 <> '' THEN ts_rank(c.tsv, plainto_tsquery('simple', $5)) ELSE 0 END DESC
		LIMIT 8
	`, makeName, model, year, codes, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	chunks := []RetrievedChunk{}
	pages := map[string][]int{}
	for rows.Next() {
		var c RetrievedChunk
		if err := rows.Scan(&c.ID, &c.SourceID, &c.Title, &c.Page, &c.Language, &c.Body, &c.BodyEN, &c.Codes); err != nil {
			return nil, nil, err
		}
		if len(c.Body) > 2500 {
			c.Body = c.Body[:2500]
		}
		if len(c.BodyEN) > 2500 {
			c.BodyEN = c.BodyEN[:2500]
		}
		chunks = append(chunks, c)
		pages[c.SourceID] = append(pages[c.SourceID], c.Page)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	figs := []RetrievedFigure{}
	for sourceID, pg := range pages {
		frows, err := s.pool.Query(ctx, `
			SELECT f.id, f.source_id, s.title, f.page, f.caption, f.language, f.image_path, f.ocr_text
			FROM doc_figures f
			JOIN doc_sources s ON s.id = f.source_id
			WHERE f.source_id = $1 AND f.page = ANY($2)
			ORDER BY f.page
		`, sourceID, pg)
		if err != nil {
			return nil, nil, err
		}
		for frows.Next() {
			var f RetrievedFigure
			var imgPath, ocr string
			if err := frows.Scan(&f.ID, &f.SourceID, &f.Title, &f.Page, &f.Caption, &f.Language, &imgPath, &ocr); err != nil {
				frows.Close()
				return nil, nil, err
			}
			f.OCRText = ocr
			if strings.TrimSpace(imgPath) != "" {
				f.HasImage = true
				f.ImageURL = "/api/v1/manuals/figures/" + f.ID + "/image"
			}
			figs = append(figs, f)
		}
		frows.Close()
		if err := frows.Err(); err != nil {
			return nil, nil, err
		}
	}
	return chunks, figs, nil
}

func (s *Store) FigureImagePath(ctx context.Context, id string) (string, error) {
	var img, root, extra string
	err := s.pool.QueryRow(ctx, `
		SELECT f.image_path, s.path, s.image_root
		FROM doc_figures f
		JOIN doc_sources s ON s.id = f.source_id
		WHERE f.id = $1
	`, id).Scan(&img, &root, &extra)
	if err != nil {
		return "", err
	}
	img = filepath.Clean(img)
	if img == "" || strings.Contains(img, "..") {
		return "", fmt.Errorf("figure has no image")
	}
	ok := under(img, root) || under(img, extra)
	if !ok {
		return "", fmt.Errorf("figure path is outside the manual tree")
	}
	if _, err := os.Stat(img); err != nil {
		return "", fmt.Errorf("figure file missing")
	}
	return img, nil
}

func under(path, root string) bool {
	root = filepath.Clean(root)
	if root == "" || root == "." {
		return false
	}
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// utf8Safe strips NULs and replaces invalid UTF-8 so Postgres will accept the row.
func utf8Safe(s string) string {
	if s == "" {
		return s
	}
	s = strings.ReplaceAll(s, "\x00", "")
	if utf8.ValidString(s) {
		return s
	}
	return string(bytes.ToValidUTF8([]byte(s), []byte("\uFFFD")))
}
