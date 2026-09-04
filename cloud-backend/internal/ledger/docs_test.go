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
