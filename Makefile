.DEFAULT_GOAL := help

.PHONY: help setup dev-deps db-setup dev-backend dev-admin build build-backend build-admin test test-backend test-admin check check-blueprints check-server check-admin check-mobile toolchain docker-up docker-down docker-logs docker-bootstrap-admin

help:
	@echo "AppKernia development commands"
	@echo "  make setup             Install frontend dependencies and initialize the database"
	@echo "  make dev-deps          Start the local PostgreSQL dependency"
	@echo "  make db-setup          Apply migrations and idempotent core seeds"
	@echo "  make dev-backend       Run API and worker from source"
	@echo "  make dev-admin         Run the Admin Vite development server"
	@echo "  make build             Build backend binaries and Admin assets"
	@echo "  make test              Run backend and Admin tests"
	@echo "  make check             Run all repository quality gates"
	@echo "  make docker-up         Run the Docker development stack"

setup:
	corepack enable
	$(MAKE) toolchain
	pnpm install --frozen-lockfile
	$(MAKE) dev-deps
	$(MAKE) db-setup

dev-deps:
	docker compose up -d --wait postgres

db-setup:
	$(MAKE) -C server db-setup

dev-backend:
	$(MAKE) -C server dev

dev-admin:
	pnpm dev

build: build-backend build-admin

build-backend:
	$(MAKE) -C server build

build-admin:
	pnpm build

test: test-backend test-admin

test-backend:
	$(MAKE) -C server test

test-admin:
	pnpm test

check: check-blueprints check-server check-admin check-mobile

check-blueprints:
	python3 blueprint/backend/tools/validate_blueprint.py
	python3 blueprint/admin-frontend/scripts/validate_blueprint_specs.py
	python3 blueprint/mobile/scripts/validate_blueprint_specs.py
	python3 blueprint/scripts/validate_i18n_contract.py

check-server:
	$(MAKE) -C server check

check-admin:
	pnpm --filter @appkernia/admin check

check-mobile:
	apps/ak-mobile/scripts/check-project.sh

toolchain:
	./scripts/doctor.sh

docker-up:
	docker compose up --build -d postgres migrate seed api admin

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f api admin

docker-bootstrap-admin:
	docker compose --profile tools run --rm bootstrap-admin
