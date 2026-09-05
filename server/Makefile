SHELL := /bin/bash
.DEFAULT_GOAL := help

GO := GOTOOLCHAIN=go1.26.5 go
BIN_DIR := bin
APP_VERSION ?= dev
GIT_COMMIT ?= unknown
BUILD_TIME ?= unknown
BUILD_LDFLAGS := -X github.com/appkernia/appkernia/server/internal/platform/buildinfo.Version=$(APP_VERSION) -X github.com/appkernia/appkernia/server/internal/platform/buildinfo.Commit=$(GIT_COMMIT) -X github.com/appkernia/appkernia/server/internal/platform/buildinfo.BuildTime=$(BUILD_TIME)
ENV_FILE ?= ../.env
LOAD_ENV = set -a; if [ -f "$(ENV_FILE)" ]; then . "$(ENV_FILE)"; fi; set +a;

.PHONY: help check fmt fmt-fix vet test test-race test-integration build build-api build-worker build-cli clean dev dev-api dev-worker run run-worker doctor db-setup sqlc-generate migration-up migration-down seed-core bootstrap-admin

help:
	@echo "AppKernia backend commands"
	@echo "  make dev               Run ak-api and ak-worker from source"
	@echo "  make dev-api           Run only ak-api with hot restart by hand"
	@echo "  make dev-worker        Run only ak-worker"
	@echo "  make build             Build API, worker, and CLI into server/bin"
	@echo "  make test              Run backend unit and contract tests"
	@echo "  make test-race         Run the Go race detector"
	@echo "  make test-integration  Run tagged tests using AK_TEST_DATABASE_URL"
	@echo "  make db-setup          Apply migrations and idempotent core seeds"
	@echo "  make bootstrap-admin   Interactively create the first administrator"
	@echo "  make check             Run formatting, vet, and tests"

check: fmt vet test

fmt:
	@test -z "$(shell GOTOOLCHAIN=go1.26.5 gofmt -l .)" || (GOTOOLCHAIN=go1.26.5 gofmt -d .; exit 1)

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

test-integration:
	@test -n "$(AK_TEST_DATABASE_URL)" || (echo "AK_TEST_DATABASE_URL is required"; exit 2)
	$(GO) test -tags=integration ./internal/seed
	$(GO) test -tags=integration ./internal/modules/opsadmin/repository ./internal/modules/iam/application ./internal/modules/iam/repository ./internal/modules/jobadmin/repository ./internal/modules/systemsettings/repository ./internal/modules/notificationadmin/repository ./internal/modules/storage/repository ./internal/modules/content/repository ./internal/modules/appmanagement/application ./internal/modules/appmanagement/worker ./internal/modules/feedback/repository ./internal/modules/feedback/worker ./internal/modules/loginprovider/repository ./internal/modules/apiclientadmin/repository

fmt-fix:
	GOTOOLCHAIN=go1.26.5 gofmt -w .

build: build-api build-worker build-cli

build-api:
	@mkdir -p "$(BIN_DIR)"
	$(GO) build -trimpath -ldflags "$(BUILD_LDFLAGS)" -o "$(BIN_DIR)/ak-api" ./cmd/ak-api

build-worker:
	@mkdir -p "$(BIN_DIR)"
	$(GO) build -trimpath -ldflags "$(BUILD_LDFLAGS)" -o "$(BIN_DIR)/ak-worker" ./cmd/ak-worker

build-cli:
	@mkdir -p "$(BIN_DIR)"
	$(GO) build -trimpath -ldflags "$(BUILD_LDFLAGS)" -o "$(BIN_DIR)/ak-cli" ./cmd/ak-cli

clean:
	rm -rf "$(BIN_DIR)" coverage.out

dev:
	@$(LOAD_ENV) \
	trap 'kill "$$api_pid" "$$worker_pid" 2>/dev/null || true' EXIT INT TERM; \
	$(MAKE) --no-print-directory dev-api & api_pid=$$!; \
	$(MAKE) --no-print-directory dev-worker & worker_pid=$$!; \
	wait "$$api_pid"; status=$$?; \
	exit "$$status"

dev-api run:
	@$(LOAD_ENV) $(GO) run ./cmd/ak-api

dev-worker run-worker:
	@$(LOAD_ENV) $(GO) run ./cmd/ak-worker

doctor:
	@$(LOAD_ENV) $(GO) run ./cmd/ak-cli doctor

db-setup: migration-up seed-core

sqlc-generate:
	$(GO) run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate

migration-up:
	@$(LOAD_ENV) $(GO) run ./cmd/ak-cli migrate up

migration-down:
	@$(LOAD_ENV) $(GO) run ./cmd/ak-cli migrate down 1

seed-core:
	@$(LOAD_ENV) $(GO) run ./cmd/ak-cli seed core

bootstrap-admin:
	@$(LOAD_ENV) $(GO) run ./cmd/ak-cli bootstrap-admin \
		--email "$${AK_BOOTSTRAP_EMAIL:-admin@example.test}" \
		--tenant-code "$${AK_BOOTSTRAP_TENANT_CODE:-local}" \
		--tenant-name "$${AK_BOOTSTRAP_TENANT_NAME:-Local Workspace}" \
		--display-name "$${AK_BOOTSTRAP_DISPLAY_NAME:-Local Admin}" \
		--locale "$${AK_BOOTSTRAP_LOCALE:-zh-CN}"
