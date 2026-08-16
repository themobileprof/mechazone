# Workshop manuals (any language)

Drop PDFs into `data/manuals/`, or point a sidecar at an HTML workshop tree (like `../avensis-zrt27`). Ingest once. The playbook retrieves **original text** — German, French, Spanish, Chinese, or anything else is fine. DeepSeek reads the source language and writes the bay playbook in the language you ask for (default English).

Translated HTML + original-language figures is expected. We keep the pictures as-is (Spanish callouts on a diagram are still the right diagram) and attach any OCR text you already extracted.

We do not require an English manual. We do not machine-translate unless you pass `-translate` (that only adds an English gloss for search; the original stays).

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

A folder of `.htm` / `.html` plus images. Text can be English; images can stay Spanish.

`data/manuals/avensis-zrt27.json` already points at `../avensis-zrt27/english` and `../avensis-zrt27/spanish`.

```bash
make ingest          # picks up that sidecar
# or:
make ingest-html     # english tree + english.json / manual.json
```

We only ingest **content** pages (`…/html/contents/…`), not menus, CSS, or chrome. Figure `src` paths that resolve into the Spanish tree are stored and served as `/api/v1/manuals/figures/{id}/image`. If `_ocr_glossary/ocr_by_image.json` exists, that Spanish-on-image text is stored with the figure and added to search.

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
