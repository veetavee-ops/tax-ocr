package main

import (
	"context"
	"flag"
	"log"
	"os"

	"tax-ocr/backend/internal/db"
)

func main() {
	stamp := flag.Bool("stamp", false, "mark all migrations as applied without executing SQL (use on existing DB)")
	flag.Parse()

	connString := envOr("DATABASE_URL", "postgres://tax_ocr:tax_ocr_dev@localhost:5433/tax_ocr?sslmode=disable")
	migrationsDir := envOr("MIGRATIONS_DIR", "../../database/migrations")

	ctx := context.Background()
	var err error
	if *stamp {
		err = db.StampMigrations(ctx, connString, migrationsDir)
	} else {
		err = db.RunMigrations(ctx, connString, migrationsDir)
	}
	if err != nil {
		log.Fatalf("migration failed: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
