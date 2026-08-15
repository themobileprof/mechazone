.PHONY: up down backend worker client seed deps install start stop

install:
	./install.sh

start:
	./scripts/start-mechazone.sh

stop:
	./scripts/stop-mechazone.sh

deps:
	cd diagnostic-worker && test -d .venv || python3 -m venv .venv
	cd diagnostic-worker && .venv/bin/pip install -r requirements.txt
	test -d third_party/j2534 || git clone --depth 1 https://github.com/NikolaKozina/j2534.git third_party/j2534
	@echo "OpenPort Linux lib: cd third_party/j2534/j2534 && make   (needs libusb-1.0-dev pkg-config)"
	@echo "DTC seed: see docs/integrations.md"
	@echo "VIN keys: CARAPI_* and VINCARIO_* in .env"

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
