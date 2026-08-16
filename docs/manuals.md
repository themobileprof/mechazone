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

Playbook retrieval: same make + model + year band, then DTC overlap, then full-text (`simple`, language-agnostic). Cited figures are `figure:<id>` (manual title + page). No LLM sketches.

## Copyright

Only ingest manuals you are allowed to keep. Public TSBs and documents you own. Do not dump a pirated AllData library into this folder.
