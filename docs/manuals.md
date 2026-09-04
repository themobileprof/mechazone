# Workshop manuals (any language)

Drop PDFs into `data/manuals/`, or point a sidecar at one HTML workshop tree under that folder. Ingest once. Keep a single language per manual. The playbook retrieves that text; DeepSeek writes the bay playbook in the language you ask for (default English).

We do not machine-translate the corpus unless you pass `-translate` (that only adds an English gloss for search; the original stays).

## 1. Put files in the folder

```
data/manuals/
  toyota_avensis_2009-2012_3zr-fae_de.pdf
  toyota_avensis_2009-2012_3zr-fae_de.json
```

Sidecar (same name as the PDF, `.json`):

```json
{
  "title": "Avensis T27 workshop manual",
  "kind": "workshop_manual",
  "make": "Toyota",
  "model": "Avensis",
  "year_from": 2009,
  "year_to": 2012,
  "engine": "3ZR-FAE",
  "language": "de"
}
```

`language` is optional. If you omit it, ingest detects it (de/en/fr/es/… or zh/ja/ru/ar).

If you skip the sidecar, the filename must look like:

`{make}_{model}_{yearFrom}-{yearTo}_{engine}_{lang}.pdf`

### HTML tree (Avensis RM, etc.)

A folder of `.htm` / `.html` plus images, one language.

After ingest, page text lives in Postgres. You can delete the HTML (and CSS/JS/search indexes). **Keep the figure files** — diagrams are served from disk, not stored in the database.

`data/manuals/avensis-zrt27.json` points at `data/manuals/avensis-zrt27/` (figures only after ingest). The tree is gitignored; the sidecar JSON is tracked.

```bash
make ingest
```

We only ingest **content** pages (`…/html/contents/…`), not menus, CSS, or chrome. Figure files next to the HTML are stored and served as `/api/v1/manuals/figures/{id}/image`. If `_ocr_glossary/ocr_by_image.json` exists, that text is stored with the figure and added to search.

## 2. Ingest

Needs `pdftotext` (poppler-utils) — already on this laptop.

```bash
make ingest
# optional English gloss for search, keeps the original:
make ingest-translate
```

Re-running the same PDF replaces its chunks (hash of the file).

## 3. What gets stored

- Original page text, language tag, DTCs found in the page
- Figure captions (Fig. / Abbildung / 図) as page citations — not generated drawings
- Optional `body_en` if you used `-translate`

Playbook retrieval: same make + model + year band (or a pinned `source_id`), then a hybrid of DTC GIN overlap, full-text (`simple`, language-agnostic), and cosine on `doc_chunks.embedding` when the ledger can embed the query (local Ollama). Cited figures are `figure:<id>` (manual title + page). No LLM sketches.

## 4. Embeddings (do not re-ingest)

HTML/PDF ingest does not write vectors. After `010`/`011` add `vector(384)` for **bge-small-en-v1.5**, backfill NULLs locally:

```bash
# Ollama on this machine (no cloud key). ~37MB Q8, 384-dim, fits a 2GB box.
make embed
```

`make embed` installs the GGUF into Ollama if needed (`scripts/install-bge-small-embed.sh`), then fills `doc_chunks.embedding`. The model is always **bge-small-en-v1.5** (384-d) — not an env var — so laptop and server cannot drift. Query-time embed failure falls back to FTS. Customer names/phones/plates are never embedded. Playbook chat (`LLM_*`) stays a separate hosted API.

## Copyright

Only ingest manuals you are allowed to keep. Public TSBs and documents you own. Do not dump a pirated AllData library into this folder.
