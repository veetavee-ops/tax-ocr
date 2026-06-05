# Tax OCR — Developer Makefile
# Usage: make <target>

DB_URL ?= postgres://tax_ocr:tax_ocr_dev@localhost:5433/tax_ocr?sslmode=disable
MIGRATIONS_DIR ?= database/migrations

.PHONY: up down dev-backend dev-admin dev-liff migrate-up migrate-stamp migrate-status migrate-reset psql

# --- Infrastructure ---

up:
	cd infrastructure && docker compose up -d

down:
	cd infrastructure && docker compose down

# --- Migration ---

# Apply all pending migrations (backend also auto-migrates on start)
migrate-up:
	cd backend && MIGRATIONS_DIR=../$(MIGRATIONS_DIR) DATABASE_URL="$(DB_URL)" go run ./cmd/migrate/...

# Stamp all migrations as applied without running SQL (use once on an existing DB)
migrate-stamp:
	cd backend && MIGRATIONS_DIR=../$(MIGRATIONS_DIR) DATABASE_URL="$(DB_URL)" go run ./cmd/migrate/... -stamp

# Show which migrations are applied
migrate-status:
	@docker exec tax-ocr-postgres psql -U tax_ocr -d tax_ocr -c \
	  "SELECT version, applied_at FROM schema_migrations ORDER BY version;"

# Reset DB: drop all tables and re-apply (DEV ONLY — no confirmation prompt)
migrate-reset:
	docker exec tax-ocr-postgres psql -U tax_ocr -d tax_ocr -c \
	  "DROP SCHEMA public CASCADE; CREATE SCHEMA public; GRANT ALL ON SCHEMA public TO tax_ocr;"
	cd backend && MIGRATIONS_DIR=../$(MIGRATIONS_DIR) DATABASE_URL="$(DB_URL)" go run ./cmd/migrate/...

# Open psql shell
psql:
	docker exec -it tax-ocr-postgres psql -U tax_ocr -d tax_ocr

# --- Dev servers ---

dev-backend:
	cd backend && MIGRATIONS_DIR=../database/migrations go run ./cmd/

dev-admin:
	cd frontend/admin && npm run dev

dev-liff:
	cd frontend/liff && npm run dev
