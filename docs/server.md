# Ledger server (not the bay laptop)

The shop laptop runs the OpenPort worker. This page is the **Go ledger + Postgres** host (including a small 2GB VPS). Playbook cosine retrieval happens here.

## Do you need Ollama on this server?

**For hybrid (FTS + cosine): yes, but only the tiny embed model.** Stored `doc_chunks.embedding` rows are not enough. Every playbook query is embedded at request time (`POST http://127.0.0.1:11434/api/embed` with hardcoded `bge-small-en-v1.5`), then compared with `<=>` in Postgres.

Measured on this project’s Ollama (model loaded, one embed):

| Process | RSS |
| --- | --- |
| `ollama serve` | ~40 MB |
| `ollama runner` + `bge-small-en-v1.5` Q8 | ~95 MB |
| **Total** | **~135 MB** |

That fits a 2GB VPS next to Postgres + the Go ledger. **Do not** `ollama pull` a chat model (`gemma3:4b` is 3.3 GB). Playbook chat stays `LLM_*` (DeepSeek).

**No Ollama** is fine if FTS-only is acceptable. Keyword / DTC GIN still run. Cosine is skipped when Ollama is down.

You do **not** re-run HTML ingest on the server if chunks and vectors are already in this database. You still need the embed model loaded for *queries* if you want cosine.

---

## Packages and processes

| Piece | Why |
| --- | --- |
| PostgreSQL 18 + `postgresql-18-pgvector` | Ledger + `vector(384)` |
| Extension `vector` in database `mechazone` | `CREATE EXTENSION vector` (superuser once) |
| Ollama listening on `127.0.0.1:11434` | Query-time (and optional backfill) embeddings |
| Ollama model `bge-small-en-v1.5` | Same weights as the laptop; `make embed-model` / `scripts/install-bge-small-embed.sh` |
| Go ledger (`make backend` or `bin/mechazone-server`) | HTTP API, playbooks, migrations |
| Hosted LLM (`LLM_*`) | Playbook JSON — not Ollama |

Not on this host: OpenPort, `J2534_LIB`, Python worker, Node/Vite (use `UI_DIR=client/dist` if you serve the bay UI from here).

---

## `.env` on this host

Copy `.env.example`. Fill only what this process uses.

| Variable | Server |
| --- | --- |
| `DATABASE_URL` | This Postgres (`postgres:///mechazone?sslmode=disable` or host/user/password URL) |
| `HTTP_ADDR` | `:8080` or bind you expose |
| `UI_DIR` | Absolute `client/dist` if this box serves the SPA; else empty |
| `SUPERADMIN_EMAIL` / `SUPERADMIN_PASSWORD` | First admin |
| `VPIC_BASE_URL` | NHTSA (default is fine) |
| `CARAPI_TOKEN` / `CARAPI_SECRET` | Optional VIN fallback |
| `VINCARIO_API_KEY` / `VINCARIO_SECRET_KEY` | Optional VIN fallback |
| `LLM_ENABLED` | `true` when playbooks should call the chat API |
| `LLM_API_KEY` / `LLM_BASE_URL` / `LLM_MODEL` | Hosted playbook model (DeepSeek default) |
| `EMBEDDING_BASE_URL` | `http://127.0.0.1:11434` (Ollama on this machine) |
| `EMBEDDING_API_KEY` | Empty for local Ollama |
| `IMPORT_DIR` | Optional scanner-file drop |

There is **no** `EMBEDDING_MODEL` / `EMBEDDING_DIM`. The binary always uses `bge-small-en-v1.5` / 384.

---

## One-time commands

```bash
sudo apt install postgresql-18-pgvector
sudo -u postgres psql -d mechazone -c 'CREATE EXTENSION IF NOT EXISTS vector;'

# Ollama: https://ollama.com/download — then:
make embed-model
# If this database has NULL embeddings:
make embed
```

Confirm: `ollama list` shows `bge-small-en-v1.5`; `SELECT model, dim FROM doc_embedding_meta;` is `bge-small-en-v1.5` / 384.

Restart the ledger after `.env` or Ollama install. Boot log should show `embed=bge-small-en-v1.5`. If Ollama is down, playbooks still run with FTS only.
