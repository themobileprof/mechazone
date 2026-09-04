package ledger

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

type DocSource struct {
	ID        string
	SHA256    string
	Path      string
	Title     string
	Kind      string
	Make      string
	Model     string
	YearFrom  int
	YearTo    int
	Engine    string
	Language  string
	ImageRoot string
}

// Chunk embeddings are always BAAI/bge-small-en-v1.5 (384-d). Not configurable.
const (
	ChunkEmbedModel = "bge-small-en-v1.5"
	ChunkEmbedDim   = 384
)

type DocChunkIn struct {
	Page     int
	Language string
	Body     string
	BodyEN   string
	Codes    []string
	RelPath  string
}

type DocFigureIn struct {
	Page      int
	Caption   string
	Language  string
	ImagePath string
	OCRText   string
}

type RetrievedChunk struct {
	ID       string   `json:"id"`
	SourceID string   `json:"source_id"`
	Title    string   `json:"title"`
	Page     int      `json:"page"`
	Language string   `json:"language"`
	Body     string   `json:"body"`
	BodyEN   string   `json:"body_en,omitempty"`
	Codes    []string `json:"codes"`
	RelPath  string   `json:"-"`
}

type RetrievedFigure struct {
	ID       string `json:"id"`
	SourceID string `json:"source_id"`
	Title    string `json:"title"`
	Page     int    `json:"page"`
	Caption  string `json:"caption"`
	Language string `json:"language"`
	HasImage bool   `json:"has_image"`
	ImageURL string `json:"image_url,omitempty"`
	OCRText  string `json:"ocr_text,omitempty"`
	Kind     string `json:"kind,omitempty"`
}

// ManualCatalog is one ingested workshop book the bay can pin for retrieval.
type ManualCatalog struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Kind     string `json:"kind"`
	Make     string `json:"make"`
	Model    string `json:"model"`
	YearFrom int    `json:"year_from"`
	YearTo   int    `json:"year_to"`
	Engine   string `json:"engine"`
	Language string `json:"language"`
	Chunks   int    `json:"chunks"`
	Figures  int    `json:"figures"`
}

// ManualQuery is playbook retrieval. SourceID pins one book when decode did not name the body.
type ManualQuery struct {
	Make      string
	Model     string
	Year      int
	Codes     []string
	Query     string
	Wiring    bool
	SourceID  string
	Embedding []float32
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

func (s *Store) ListManuals(ctx context.Context) ([]ManualCatalog, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT s.id::text, s.title, s.kind, s.make, s.model, s.year_from, s.year_to, s.engine, s.language,
		       (SELECT count(*) FROM doc_chunks c WHERE c.source_id = s.id),
		       (SELECT count(*) FROM doc_figures f WHERE f.source_id = s.id)
		FROM doc_sources s
		ORDER BY s.make, s.model, s.year_from, s.title
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ManualCatalog{}
	for rows.Next() {
		var m ManualCatalog
		if err := rows.Scan(&m.ID, &m.Title, &m.Kind, &m.Make, &m.Model, &m.YearFrom, &m.YearTo, &m.Engine, &m.Language, &m.Chunks, &m.Figures); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) GetManual(ctx context.Context, id string) (ManualCatalog, error) {
	var m ManualCatalog
	err := s.pool.QueryRow(ctx, `
		SELECT s.id::text, s.title, s.kind, s.make, s.model, s.year_from, s.year_to, s.engine, s.language,
		       (SELECT count(*) FROM doc_chunks c WHERE c.source_id = s.id),
		       (SELECT count(*) FROM doc_figures f WHERE f.source_id = s.id)
		FROM doc_sources s
		WHERE s.id = $1::uuid
	`, strings.TrimSpace(id)).Scan(&m.ID, &m.Title, &m.Kind, &m.Make, &m.Model, &m.YearFrom, &m.YearTo, &m.Engine, &m.Language, &m.Chunks, &m.Figures)
	if errors.Is(err, pgx.ErrNoRows) {
		return ManualCatalog{}, fmt.Errorf("workshop book not on file")
	}
	return m, err
}

func (s *Store) SearchManuals(ctx context.Context, q ManualQuery) ([]RetrievedChunk, []RetrievedFigure, error) {
	q.Make = strings.TrimSpace(q.Make)
	q.Model = strings.TrimSpace(q.Model)
	q.SourceID = strings.TrimSpace(q.SourceID)
	if q.SourceID == "" && (q.Make == "" || q.Model == "") {
		return []RetrievedChunk{}, []RetrievedFigure{}, nil
	}
	if q.Codes == nil {
		q.Codes = []string{}
	}
	query := strings.TrimSpace(q.Query)
	year := q.Year
	if year <= 0 || q.SourceID != "" {
		year = 2000
	}
	if q.Wiring {
		if query == "" {
			query = "wiring connector EWD circuit"
		} else {
			query = query + " wiring connector EWD open circuit"
		}
	}
	ewdBoost := 0
	if q.Wiring {
		ewdBoost = 1
	}
	var source any
	if q.SourceID != "" {
		source = q.SourceID
	}
	makeName := q.Make
	model := q.Model
	if q.SourceID != "" {
		makeName = ""
		model = ""
	}
	if !s.hasChunkEmbeddings {
		q.Embedding = nil
	} else if len(q.Embedding) > 0 {
		meta, err := s.EmbeddingMeta(ctx)
		if err != nil {
			return nil, nil, err
		}
		if meta.Model == "" || !meta.MatchesIndex() || len(q.Embedding) != ChunkEmbedDim {
			q.Embedding = nil
		}
	}
	fts, err := s.searchManualsRanked(ctx, source, makeName, model, year, q.Codes, query, ewdBoost, "", 30)
	if err != nil {
		return nil, nil, err
	}
	chunks := fts
	if len(q.Embedding) > 0 {
		vec, err := s.searchManualsRanked(ctx, source, makeName, model, year, q.Codes, query, ewdBoost, VectorLiteral(q.Embedding), 30)
		if err != nil {
			return nil, nil, err
		}
		chunks = mergeHybridChunks(fts, vec, q.Codes, q.Wiring)
	} else if len(chunks) > 10 {
		chunks = chunks[:10]
	}
	return s.figuresForChunks(ctx, chunks, q, source, makeName, model, year)
}

func (s *Store) searchManualsRanked(ctx context.Context, source any, makeName, model string, year int, codes []string, query string, ewdBoost int, qvec string, limit int) ([]RetrievedChunk, error) {
	filter := `
		SELECT c.id, c.source_id, s.title, c.page, c.language, c.body, c.body_en, c.codes, c.rel_path
		FROM doc_chunks c
		JOIN doc_sources s ON s.id = c.source_id
		WHERE ($1::uuid IS NULL OR s.id = $1::uuid)
		  AND ($2 = '' OR lower(s.make) = lower($2))
		  AND ($3 = '' OR lower(s.model) = lower($3))
		  AND ($4 = 2000 OR (s.year_from <= $4 AND s.year_to >= $4))
		  AND (
		        cardinality($5::text[]) = 0
		        OR c.codes && $5
		        OR ($6 <> '' AND c.tsv @@ plainto_tsquery('simple', $6))
		        OR $6 = ''
		      )`
	var rows pgx.Rows
	var err error
	if qvec == "" {
		rows, err = s.pool.Query(ctx, filter+`
			ORDER BY ($7::int * CASE WHEN c.rel_path ILIKE '%ewd%' OR c.rel_path ILIKE '%connector%' THEN 1 ELSE 0 END) DESC,
			         (c.codes && $5) DESC,
			         CASE WHEN $6 <> '' THEN ts_rank(c.tsv, plainto_tsquery('simple', $6)) ELSE 0 END DESC
			LIMIT $8
		`, source, makeName, model, year, codes, query, ewdBoost, limit)
	} else {
		rows, err = s.pool.Query(ctx, `
		SELECT c.id, c.source_id, s.title, c.page, c.language, c.body, c.body_en, c.codes, c.rel_path
		FROM doc_chunks c
		JOIN doc_sources s ON s.id = c.source_id
		WHERE ($1::uuid IS NULL OR s.id = $1::uuid)
		  AND ($2 = '' OR lower(s.make) = lower($2))
		  AND ($3 = '' OR lower(s.model) = lower($3))
		  AND ($4 = 2000 OR (s.year_from <= $4 AND s.year_to >= $4))
		  AND c.embedding IS NOT NULL
		ORDER BY ($5::int * CASE WHEN c.rel_path ILIKE '%ewd%' OR c.rel_path ILIKE '%connector%' THEN 1 ELSE 0 END) DESC,
		         (c.codes && $6) DESC,
		         c.embedding <=> $7::vector
		LIMIT $8
		`, source, makeName, model, year, ewdBoost, codes, qvec, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	chunks := []RetrievedChunk{}
	for rows.Next() {
		var c RetrievedChunk
		if err := rows.Scan(&c.ID, &c.SourceID, &c.Title, &c.Page, &c.Language, &c.Body, &c.BodyEN, &c.Codes, &c.RelPath); err != nil {
			return nil, err
		}
		if len(c.Body) > 2500 {
			c.Body = c.Body[:2500]
		}
		if len(c.BodyEN) > 2500 {
			c.BodyEN = c.BodyEN[:2500]
		}
		chunks = append(chunks, c)
	}
	return chunks, rows.Err()
}

func (s *Store) figuresForChunks(ctx context.Context, chunks []RetrievedChunk, q ManualQuery, source any, makeName, model string, year int) ([]RetrievedChunk, []RetrievedFigure, error) {
	pages := map[string][]int{}
	for _, c := range chunks {
		pages[c.SourceID] = append(pages[c.SourceID], c.Page)
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
			f.Kind = figureKind(imgPath)
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
	if q.Wiring {
		extra, err := s.searchWiringFigures(ctx, source, makeName, model, year)
		if err != nil {
			return nil, nil, err
		}
		figs = mergeFigures(figs, extra)
	}
	return chunks, figs, nil
}

type ChunkEmbedRow struct {
	ID     string
	Body   string
	BodyEN string
	Codes  []string
}

func (s *Store) ChunksWithoutEmbedding(ctx context.Context, limit int) ([]ChunkEmbedRow, error) {
	if !s.hasChunkEmbeddings {
		return nil, fmt.Errorf("doc_chunks.embedding is missing; install postgresql-18-pgvector, then: sudo -u postgres psql -d mechazone -c 'CREATE EXTENSION vector'")
	}
	if limit <= 0 {
		limit = 32
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, body, body_en, codes
		FROM doc_chunks
		WHERE embedding IS NULL
		ORDER BY id
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ChunkEmbedRow{}
	for rows.Next() {
		var r ChunkEmbedRow
		if err := rows.Scan(&r.ID, &r.Body, &r.BodyEN, &r.Codes); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) SetChunkEmbeddings(ctx context.Context, ids []string, vecs [][]float32) error {
	if len(ids) != len(vecs) {
		return fmt.Errorf("embed batch size mismatch")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for i, id := range ids {
		if _, err := tx.Exec(ctx, `UPDATE doc_chunks SET embedding = $2::vector WHERE id = $1::uuid`, id, VectorLiteral(vecs[i])); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// EmbeddingMeta is the model that wrote doc_chunks.embedding. Empty Model means none yet.
type EmbeddingMeta struct {
	Model string
	Dim   int
}

func CanonicalEmbedModel(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.TrimSuffix(s, ":latest")
}

func (m EmbeddingMeta) MatchesIndex() bool {
	return CanonicalEmbedModel(m.Model) == ChunkEmbedModel && m.Dim == ChunkEmbedDim
}

func (s *Store) EmbeddingMeta(ctx context.Context) (EmbeddingMeta, error) {
	var m EmbeddingMeta
	err := s.pool.QueryRow(ctx, `SELECT model, dim FROM doc_embedding_meta WHERE singleton`).Scan(&m.Model, &m.Dim)
	if errors.Is(err, pgx.ErrNoRows) {
		return EmbeddingMeta{}, nil
	}
	return m, err
}

func EmbedModelMismatch(index EmbeddingMeta) error {
	return fmt.Errorf("embedding index is %s (%d-dim); Mechazone only uses %s (%d-dim)", CanonicalEmbedModel(index.Model), index.Dim, ChunkEmbedModel, ChunkEmbedDim)
}

// LockEmbeddingModel records bge-small-en-v1.5 as the index model. Refuses any other row.
func (s *Store) LockEmbeddingModel(ctx context.Context) error {
	meta, err := s.EmbeddingMeta(ctx)
	if err != nil {
		return err
	}
	if meta.Model == "" {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO doc_embedding_meta (singleton, model, dim)
			VALUES (TRUE, $1, $2)
			ON CONFLICT (singleton) DO NOTHING
		`, ChunkEmbedModel, ChunkEmbedDim)
		if err != nil {
			return err
		}
		meta, err = s.EmbeddingMeta(ctx)
		if err != nil {
			return err
		}
	}
	if !meta.MatchesIndex() {
		return EmbedModelMismatch(meta)
	}
	return nil
}

func VectorLiteral(v []float32) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(x), 'f', 6, 32))
	}
	b.WriteByte(']')
	return b.String()
}

func mergeHybridChunks(fts, vec []RetrievedChunk, codes []string, wiring bool) []RetrievedChunk {
	const k = 60
	type scored struct {
		chunk RetrievedChunk
		score float64
	}
	byID := map[string]*scored{}
	add := func(list []RetrievedChunk, weight float64) {
		for i, c := range list {
			if c.ID == "" {
				continue
			}
			row, ok := byID[c.ID]
			if !ok {
				cp := c
				row = &scored{chunk: cp}
				byID[c.ID] = row
			}
			row.score += weight / float64(k+i+1)
		}
	}
	add(fts, 1)
	add(vec, 1)
	want := map[string]struct{}{}
	for _, c := range codes {
		c = strings.ToUpper(strings.TrimSpace(c))
		if c != "" {
			want[c] = struct{}{}
		}
	}
	out := make([]scored, 0, len(byID))
	for _, row := range byID {
		for _, code := range row.chunk.Codes {
			if _, ok := want[strings.ToUpper(strings.TrimSpace(code))]; ok {
				row.score += 1
				break
			}
		}
		if wiring {
			p := strings.ToLower(row.chunk.RelPath)
			if strings.Contains(p, "ewd") || strings.Contains(p, "connector") {
				row.score += 0.5
			}
		}
		out = append(out, *row)
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].score > out[i].score {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	n := 10
	if n > len(out) {
		n = len(out)
	}
	chunks := make([]RetrievedChunk, 0, n)
	for i := 0; i < n; i++ {
		chunks = append(chunks, out[i].chunk)
	}
	return chunks
}

func (s *Store) searchWiringFigures(ctx context.Context, source any, makeName, model string, year int) ([]RetrievedFigure, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT f.id, f.source_id, s.title, f.page, f.caption, f.language, f.image_path, f.ocr_text
		FROM doc_figures f
		JOIN doc_sources s ON s.id = f.source_id
		WHERE ($1::uuid IS NULL OR s.id = $1::uuid)
		  AND ($2 = '' OR lower(s.make) = lower($2))
		  AND ($3 = '' OR lower(s.model) = lower($3))
		  AND ($4 = 2000 OR (s.year_from <= $4 AND s.year_to >= $4))
		  AND (f.image_path ILIKE '%/ewd/%' OR f.image_path ILIKE '%connector%' OR f.caption ILIKE '%connector%')
		ORDER BY (f.image_path ILIKE '%/ewd/%') DESC, f.page
		LIMIT 8
	`, source, makeName, model, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RetrievedFigure{}
	for rows.Next() {
		var f RetrievedFigure
		var imgPath, ocr string
		if err := rows.Scan(&f.ID, &f.SourceID, &f.Title, &f.Page, &f.Caption, &f.Language, &imgPath, &ocr); err != nil {
			return nil, err
		}
		f.OCRText = ocr
		f.Kind = figureKind(imgPath)
		if strings.TrimSpace(imgPath) != "" {
			f.HasImage = true
			f.ImageURL = "/api/v1/manuals/figures/" + f.ID + "/image"
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func figureKind(path string) string {
	p := strings.ToLower(path)
	if strings.Contains(p, "/ewd/") {
		return "ewd"
	}
	if strings.Contains(p, "connector") {
		return "connector"
	}
	return "procedure"
}

func mergeFigures(base, extra []RetrievedFigure) []RetrievedFigure {
	seen := map[string]struct{}{}
	out := make([]RetrievedFigure, 0, len(base)+len(extra))
	for _, f := range base {
		if f.ID == "" {
			continue
		}
		if _, ok := seen[f.ID]; ok {
			continue
		}
		seen[f.ID] = struct{}{}
		out = append(out, f)
	}
	for _, f := range extra {
		if f.ID == "" {
			continue
		}
		if _, ok := seen[f.ID]; ok {
			continue
		}
		seen[f.ID] = struct{}{}
		out = append(out, f)
	}
	return out
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
