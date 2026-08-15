.PHONY: up down backend worker client seed

up:
	docker compose up -d
	@echo "Postgres on :5432 (pgvector/pg16)"

down:
	docker compose down

backend:
	cd cloud-backend && go run ./cmd/server

worker:
	cd diagnostic-worker && .venv/bin/python -m mechazone_worker

client:
	cd client && npm run dev

seed:
	cd cloud-backend && go run ./cmd/server -seed-only
