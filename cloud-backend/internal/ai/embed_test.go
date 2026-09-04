package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mechazone/cloud-backend/internal/ledger"
)

func TestClipEmbedText(t *testing.T) {
	if got := clipEmbedText("  hello  ", embedMaxRunes); got != "hello" {
		t.Fatalf("got %q", got)
	}
	long := strings.Repeat("ä", embedMaxRunes+10)
	got := clipEmbedText(long, embedMaxRunes)
	if n := len([]rune(got)); n != embedMaxRunes {
		t.Fatalf("clipped to %d runes, want %d", n, embedMaxRunes)
	}
}

func TestEmbedCorpusPrefersEnglish(t *testing.T) {
	got := embedCorpus([]string{"P1047"}, "de text", "Valvematic actuator")
	if !strings.Contains(got, "P1047") || !strings.Contains(got, "Valvematic") {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(got, "de text") {
		t.Fatal("body_en should win over body")
	}
}

func TestEmbedderParsesIndexedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" {
			t.Errorf("path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("auth %q", got)
		}
		raw, _ := io.ReadAll(r.Body)
		var req struct {
			Input []string `json:"input"`
			Model string   `json:"model"`
		}
		if err := json.Unmarshal(raw, &req); err != nil {
			t.Error(err)
		}
		if req.Model != ledger.ChunkEmbedModel || len(req.Input) != 2 {
			t.Errorf("body %+v", req)
		}
		vec0 := make([]float32, ledger.ChunkEmbedDim)
		vec1 := make([]float32, ledger.ChunkEmbedDim)
		vec1[0] = 1
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"index": 1, "embedding": vec1},
				{"index": 0, "embedding": vec0},
			},
		})
	}))
	defer srv.Close()

	e := NewEmbedder(srv.URL, "test-key")
	out, err := e.Embed(context.Background(), []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0][0] != 0 || out[1][0] != 1 {
		t.Fatalf("index order: %#v", out[0][:1])
	}
}

func TestRetrievalQueryUsesDIDNames(t *testing.T) {
	q := retrievalQuery(Request{
		ActiveCodes: []string{"P1047"},
		EngineHint:  "3ZR-FAE",
		Make:        "Toyota",
		Model:       "Avensis",
		Live:        []LiveRow{{Name: "Valvematic duty", Value: 12}},
	})
	for _, want := range []string{"P1047", "3ZR-FAE", "Toyota", "Avensis", "Valvematic duty"} {
		if !strings.Contains(q, want) {
			t.Fatalf("%q missing from %q", want, q)
		}
	}
}

func TestEmbedderParsesOllamaNative(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embed" {
			t.Errorf("path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "" {
			t.Error("local ollama should not send a bearer key")
		}
		vec := make([]float32, ledger.ChunkEmbedDim)
		vec[0] = 0.5
		_ = json.NewEncoder(w).Encode(map[string]any{"embeddings": [][]float32{vec}})
	}))
	defer srv.Close()
	e := NewEmbedder(srv.URL+"/api/embed", "")
	out, err := e.Embed(context.Background(), []string{"P1047"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0][0] != 0.5 || len(out[0]) != ledger.ChunkEmbedDim {
		t.Fatalf("got %#v", out)
	}
}

func TestBGEQueryPrefix(t *testing.T) {
	e := NewEmbedder("http://127.0.0.1:11434", "")
	got := e.QueryText("P1047 Valvematic")
	if !strings.HasPrefix(got, bgeQueryPrefix) {
		t.Fatalf("got %q", got)
	}
}
