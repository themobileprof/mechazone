.PHONY: backend worker client seed deps install start stop ingest ingest-translate embed embed-model pack

install:
	./install.sh

start:
	./scripts/start-mechazone.sh

stop:
	./scripts/stop-mechazone.sh

pack:
	./scripts/pack-release.sh linux amd64

deps:
	cd diagnostic-worker && test -d .venv || python3 -m venv .venv
	cd diagnostic-worker && .venv/bin/pip install -r requirements.txt
	test -d third_party/j2534 || git clone --depth 1 https://github.com/NikolaKozina/j2534.git third_party/j2534
	@echo "OpenPort Linux lib: cd third_party/j2534/j2534 && make   (needs libusb-1.0-dev pkg-config)"
	@echo "DTC seed: see docs/integrations.md"
	@echo "VIN keys: CARAPI_* and VINCARIO_* in .env"

backend:
	cd cloud-backend && GOTOOLCHAIN=local go run ./cmd/server

worker:
	cd diagnostic-worker && .venv/bin/python -m mechazone_worker

client:
	cd client && npm run dev

seed:
	cd cloud-backend && GOTOOLCHAIN=local go run ./cmd/server -seed-only

ingest:
	cd cloud-backend && GOTOOLCHAIN=local go run ./cmd/ingest -dir ../data/manuals

ingest-translate:
	cd cloud-backend && GOTOOLCHAIN=local go run ./cmd/ingest -dir ../data/manuals -translate

embed-model:
	./scripts/install-bge-small-embed.sh

embed: embed-model
	cd cloud-backend && GOTOOLCHAIN=local go run ./cmd/ingest -embed-only
