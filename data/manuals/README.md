Drop workshop PDFs here, or one HTML tree next to a sidecar JSON (`html_root` relative to this folder). Pair each PDF with a `.json` sidecar (make, model, years). One language per manual — see `docs/manuals.md`.

After `make ingest`, HTML can be deleted. Keep figure PNGs so the bay can still show diagrams.

The Avensis T27 RM lives in `avensis-zrt27/` (gitignored). Sidecar: `avensis-zrt27.json`.

Then from the repo root:

```bash
make ingest
```
