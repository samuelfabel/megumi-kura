.PHONY: dev test build db-migrate db-seed web-build web-install clean

ROOT := $(abspath .)
API_DIR := $(ROOT)/api
WEB_DIR := $(ROOT)/web
BIN_DIR := $(ROOT)/bin
BINARY := $(BIN_DIR)/megumi-kura

export MK_DATABASE_URL ?= postgres://megumi:megumi@localhost:5432/megumi_kura?sslmode=disable
export MK_HTTP_ADDR ?= :8080
export MK_STATIC_DIR ?= $(WEB_DIR)/dist
export MK_SESSION_SECRET ?= change-me-dev-only-secret

dev: web-build
	cd $(API_DIR) && go run ./cmd/megumi-kura

test:
	cd $(API_DIR) && go test ./...

build: web-build
	mkdir -p $(BIN_DIR)
	cd $(API_DIR) && go build -o $(BINARY) ./cmd/megumi-kura

db-migrate:
	cd $(API_DIR) && go run ./cmd/megumi-kura -migrate-only

db-seed:
	psql "$(MK_DATABASE_URL)" -v ON_ERROR_STOP=1 -f $(ROOT)/scripts/seed.sql

web-install:
	cd $(WEB_DIR) && npm install

web-build: web-install
	cd $(WEB_DIR) && npm run build

clean:
	rm -rf $(BIN_DIR) $(WEB_DIR)/dist $(WEB_DIR)/node_modules
