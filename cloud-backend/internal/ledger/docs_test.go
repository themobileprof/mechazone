package ledger

import (
	"context"
	"os"
	"testing"
	"time"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres:///mechazone?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	s, err := Open(ctx, url)
	if err != nil {
		t.Skip(err)
	}
	if err := s.Migrate(ctx); err != nil {
		s.Close()
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s
}

func TestListAndSearchAvensisManual(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	books, err := s.ListManuals(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var avensis ManualCatalog
	for _, b := range books {
		if b.Model == "Avensis" {
			avensis = b
			break
		}
	}
	if avensis.ID == "" {
		t.Skip("Avensis T27 not ingested")
	}
	if avensis.Chunks < 1 {
		t.Fatalf("expected ingested pages, got %+v", avensis)
	}
	chunks, figs, err := s.SearchManuals(ctx, ManualQuery{SourceID: avensis.ID, Query: "Valvematic", Codes: []string{"P1047"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) == 0 {
		t.Fatal("pinned Avensis book returned no chunks for Valvematic/P1047")
	}
	_ = figs
}

func TestVectorLiteral(t *testing.T) {
	if got := VectorLiteral(nil); got != "[]" {
		t.Fatalf("nil: %q", got)
	}
	got := VectorLiteral([]float32{1, -0.5})
	if got != "[1.000000,-0.500000]" {
		t.Fatalf("got %q", got)
	}
}

func TestMergeHybridChunksPrefersCodeAndEWD(t *testing.T) {
	fts := []RetrievedChunk{
		{ID: "a", RelPath: "diag.htm"},
		{ID: "b", RelPath: "foo.htm"},
		{ID: "c", RelPath: "html/ewd/connector.htm", Codes: []string{"P1047"}},
	}
	vec := []RetrievedChunk{
		{ID: "c", RelPath: "html/ewd/connector.htm", Codes: []string{"P1047"}},
		{ID: "d", RelPath: "other.htm"},
		{ID: "a", RelPath: "diag.htm"},
	}
	got := mergeHybridChunks(fts, vec, []string{"P1047"}, true)
	if len(got) == 0 || got[0].ID != "c" {
		t.Fatalf("expected P1047 EWD chunk first, got %+v", got)
	}
}

func TestLockEmbeddingModelIsBGE(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.LockEmbeddingModel(ctx); err != nil {
		t.Fatal(err)
	}
	meta, err := s.EmbeddingMeta(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !meta.MatchesIndex() {
		t.Fatalf("got %+v", meta)
	}
	if err := s.LockEmbeddingModel(ctx); err != nil {
		t.Fatal(err)
	}
}
