.PHONY: all build frontend backend test clean

VERSION := $(shell ./scripts/version.sh)

all: build

# Full build: npm build → copy to embed dir → go build.
build: frontend backend

frontend:
	cd web && npm run build
	rm -rf internal/api/frontend
	cp -r web/build internal/api/frontend

backend:
	go build -ldflags "-X github.com/ultimoistante/timbre/internal/version.Version=$(VERSION)" -o bin/timbre-server ./cmd/server

# Run backend only (assumes frontend already built).
run: backend
	TIMBRE_DATA_DIR=./data ./bin/timbre-server

# Dev: run Vite dev server (proxies /api to :8080) alongside Go backend.
dev:
	@echo "Start backend: TIMBRE_DATA_DIR=./data go run ./cmd/server"
	@echo "Start frontend dev: cd web && npm run dev"

test:
	go test ./...

clean:
	rm -rf bin/ internal/api/frontend/ web/build/ web/.svelte-kit/ data/
