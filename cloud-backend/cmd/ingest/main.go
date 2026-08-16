package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"mechazone/cloud-backend/internal/ai"
	"mechazone/cloud-backend/internal/config"
	"mechazone/cloud-backend/internal/ledger"
)

func main() {
	dir := flag.String("dir", "data/manuals", "folder of PDFs and HTML-tree sidecar JSON files")
	htmlRoot := flag.String("html", "", "ingest this HTML manual tree (needs a sidecar JSON next to it or make/model flags)")
	translate := flag.Bool("translate", false, "also store an English gloss via the LLM (original language is always kept)")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}
	ctx := context.Background()
	store, err := ledger.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database", "err", err)
		os.Exit(1)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		log.Error("migrate", "err", err)
		os.Exit(1)
	}

	fuser := &ai.Fuser{Store: store, Log: log}
	if *translate && cfg.LLMReady() {
		fuser.LLM = ai.NewClient(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel)
	} else if *translate {
		log.Error("LLM_API_KEY required for -translate")
		os.Exit(1)
	}

	var results []ai.IngestResult
	if strings.TrimSpace(*htmlRoot) != "" {
		root, err := filepath.Abs(*htmlRoot)
		if err != nil {
			log.Error("html", "err", err)
			os.Exit(1)
		}
		side, err := ai.LoadSidecarFile(root + ".json")
		if err != nil || side.Make == "" {
			side, err = ai.LoadSidecarFile(filepath.Join(filepath.Dir(root), "manual.json"))
		}
		if err != nil || side.Make == "" {
			log.Error("html sidecar missing", "hint", "put make/model/years in "+root+".json")
			os.Exit(1)
		}
		res, err := fuser.IngestHTMLTree(ctx, root, side, *translate)
		if err != nil {
			log.Error("ingest", "err", err)
			os.Exit(1)
		}
		results = []ai.IngestResult{res}
	} else {
		abs, err := filepath.Abs(*dir)
		if err != nil {
			log.Error("dir", "err", err)
			os.Exit(1)
		}
		results, err = fuser.IngestDir(ctx, abs, *translate)
		if err != nil {
			log.Error("ingest", "err", err)
			os.Exit(1)
		}
	}
	if len(results) == 0 {
		fmt.Fprintf(os.Stderr, "nothing to ingest — see docs/manuals.md\n")
		os.Exit(1)
	}
	for _, r := range results {
		if r.Skipped != "" {
			fmt.Printf("%s  skipped  %s\n", filepath.Base(r.Path), r.Skipped)
			continue
		}
		fmt.Printf("%s  lang=%s  chunks=%d  figures=%d  id=%s\n", filepath.Base(r.Path), r.Language, r.Chunks, r.Figures, r.SourceID)
	}
}
