GO      ?= go
BIN      = server
DB       = data/archivist.db
EXCEL   := archivist.xlsx

.PHONY: help build run run-release dev build-server build-importer import admin tidy vet fmt clean db-reset test

help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  build         Build server + importer binaries"
	@echo "  build-server  Build server binary only"
	@echo "  build-importer Build importer binary only"
	@echo "  run           Start dev server (GIN_MODE=debug)"
	@echo "  run-release   Start server in release mode"
	@echo "  dev           Build then run"
	@echo "  import        Import Excel data into DB"
	@echo "  admin         Create admin user (prompts for password)"
	@echo "  tidy          go mod tidy"
	@echo "  vet           go vet ./..."
	@echo "  fmt           go fmt ./..."
	@echo "  clean         Remove binaries and DB"
	@echo "  db-reset      Delete SQLite database only"
	@echo "  test          go test ./..."
	@echo "  help          Show this message"

build: build-server build-importer

build-server:
	$(GO) build -o $(BIN) ./cmd/server/

build-importer:
	$(GO) build -o importer ./cmd/importer/

run:
	GIN_MODE=debug $(GO) run ./cmd/server/main.go

run-release:
	$(GO) run ./cmd/server/main.go

dev: build run

import:
	$(GO) run ./cmd/importer/main.go ./$(EXCEL)

admin:
	$(GO) run ./cmd/server/main.go --create-admin

tidy:
	$(GO) mod tidy

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

clean:
	rm -f $(BIN) importer $(DB)

db-reset:
	rm -f $(DB)

test:
	$(GO) test ./...
