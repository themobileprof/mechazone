package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"mechazone/cloud-backend/internal/ledger"
)

const embedBatch = 8
const embedMaxRunes = 6000
const bgeSmallMaxRunes = 1000

const bgeQueryPrefix = "Represent this sentence for searching relevant passages: "

// Embedder calls a local or hosted OpenAI-compatible embeddings API. It does not train a model.
type Embedder struct {
	BaseURL    string
	APIKey     string
	Model      string
	Dim        int
	HTTPClient *http.Client
}

func NewEmbedder(baseURL, apiKey string) *Embedder {
	return &Embedder{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		APIKey:     apiKey,
		Model:      ledger.ChunkEmbedModel,
		Dim:        ledger.ChunkEmbedDim,
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
	}
}

func (e *Embedder) Ready() bool {
	return e != nil && e.BaseURL != "" && e.Model != "" && e.Dim > 0
}

// QueryText adds the BGE retrieval instruction for search queries. Chunk text is not prefixed.
func (e *Embedder) QueryText(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return q
	}
	return bgeQueryPrefix + q
}

// Embed returns one vector per input, in the same order.
func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if !e.Ready() {
		return nil, fmt.Errorf("embedding API is not configured")
	}
	if len(texts) == 0 {
		return nil, nil
	}
	out := make([][]float32, 0, len(texts))
	for i := 0; i < len(texts); i += embedBatch {
		end := i + embedBatch
		if end > len(texts) {
			end = len(texts)
		}
		batch, err := e.embedOnce(ctx, texts[i:end])
		if err != nil {
			return nil, err
		}
		out = append(out, batch...)
	}
	return out, nil
}

func (e *Embedder) embedOnce(ctx context.Context, texts []string) ([][]float32, error) {
	clipped := make([]string, len(texts))
	for i, t := range texts {
		clipped[i] = clipEmbedText(t, bgeSmallMaxRunes)
		if clipped[i] == "" {
			clipped[i] = " "
		}
	}
	ollama := strings.Contains(e.BaseURL, ":11434") || strings.Contains(e.BaseURL, "/api/embed")
	body, err := json.Marshal(map[string]any{"model": e.Model, "input": clipped})
	if err != nil {
		return nil, err
	}
	url := e.embedURL(ollama)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if e.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.APIKey)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embed request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embed status %d: %s", resp.StatusCode, clipErr(raw))
	}
	return e.parseEmbedResponse(raw, len(texts))
}

func (e *Embedder) embedURL(ollama bool) string {
	url := e.BaseURL
	if strings.Contains(url, "/embed") {
		return url
	}
	if ollama {
		return strings.TrimRight(url, "/") + "/api/embed"
	}
	return strings.TrimRight(url, "/") + "/v1/embeddings"
}

func (e *Embedder) parseEmbedResponse(raw []byte, n int) ([][]float32, error) {
	var openai struct {
		Data []struct {
			Index     int       `json:"index"`
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &openai); err == nil && len(openai.Data) == n {
		return e.orderVectors(openai.Data, n)
	}
	var ollama struct {
		Embeddings [][]float32 `json:"embeddings"`
		Embedding  []float32   `json:"embedding"`
	}
	if err := json.Unmarshal(raw, &ollama); err != nil {
		return nil, fmt.Errorf("embed decode: %w", err)
	}
	rows := ollama.Embeddings
	if len(rows) == 0 && len(ollama.Embedding) > 0 {
		rows = [][]float32{ollama.Embedding}
	}
	if len(rows) != n {
		return nil, fmt.Errorf("embed returned %d vectors for %d inputs", len(rows), n)
	}
	out := make([][]float32, n)
	for i, v := range rows {
		if len(v) != e.Dim {
			return nil, fmt.Errorf("embed dim %d, expected %d (%s)", len(v), e.Dim, ledger.ChunkEmbedModel)
		}
		out[i] = v
	}
	return out, nil
}

func (e *Embedder) orderVectors(data []struct {
	Index     int       `json:"index"`
	Embedding []float32 `json:"embedding"`
}, n int) ([][]float32, error) {
	out := make([][]float32, n)
	byIndex := false
	for _, row := range data {
		if row.Index != 0 {
			byIndex = true
			break
		}
	}
	for i, row := range data {
		if len(row.Embedding) != e.Dim {
			return nil, fmt.Errorf("embed dim %d, expected %d (%s)", len(row.Embedding), e.Dim, ledger.ChunkEmbedModel)
		}
		pos := i
		if byIndex {
			pos = row.Index
		}
		if pos < 0 || pos >= len(out) {
			return nil, fmt.Errorf("embed index %d out of range", pos)
		}
		out[pos] = row.Embedding
	}
	for i, v := range out {
		if len(v) != e.Dim {
			return nil, fmt.Errorf("embed missing vector at index %d", i)
		}
	}
	return out, nil
}

func clipEmbedText(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if maxRunes <= 0 {
		maxRunes = embedMaxRunes
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	r := []rune(s)
	return string(r[:maxRunes])
}

func embedCorpus(codes []string, body, bodyEN string) string {
	text := strings.TrimSpace(bodyEN)
	if text == "" {
		text = strings.TrimSpace(body)
	}
	if len(codes) == 0 {
		return text
	}
	return strings.TrimSpace(strings.Join(codes, " ") + "\n" + text)
}

// EmbedPending backfills doc_chunks.embedding for rows that still have NULL.
// Does not re-ingest HTML. Customer fields are never embedded.
func (f *Fuser) EmbedPending(ctx context.Context) (int, error) {
	if f == nil || f.Store == nil {
		return 0, fmt.Errorf("ledger store is required")
	}
	if f.Embed == nil || !f.Embed.Ready() {
		return 0, fmt.Errorf("embedding API is not configured")
	}
	if err := f.Store.LockEmbeddingModel(ctx); err != nil {
		return 0, err
	}
	total := 0
	for {
		rows, err := f.Store.ChunksWithoutEmbedding(ctx, embedBatch)
		if err != nil {
			return total, err
		}
		if len(rows) == 0 {
			return total, nil
		}
		ids := make([]string, len(rows))
		texts := make([]string, len(rows))
		for i, r := range rows {
			ids[i] = r.ID
			texts[i] = embedCorpus(r.Codes, r.Body, r.BodyEN)
		}
		vecs, err := f.Embed.Embed(ctx, texts)
		if err != nil {
			return total, err
		}
		if err := f.Store.SetChunkEmbeddings(ctx, ids, vecs); err != nil {
			return total, err
		}
		total += len(ids)
		if f.Log != nil {
			f.Log.Info("embedded chunks", "batch", len(ids), "total", total)
		}
	}
}
