package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"tax-ocr/backend/internal/api"
	"tax-ocr/backend/internal/db"
	"tax-ocr/backend/internal/storage"
)

func main() {
	addr := os.Getenv("APP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		connString = "postgres://tax_ocr:tax_ocr_dev@localhost:5433/tax_ocr?sslmode=disable"
	}

	minioEndpoint := os.Getenv("MINIO_ENDPOINT")
	if minioEndpoint == "" {
		minioEndpoint = "localhost:9000"
	}
	minioAccessKey := os.Getenv("MINIO_ROOT_USER")
	if minioAccessKey == "" {
		minioAccessKey = "minioadmin"
	}
	minioSecretKey := os.Getenv("MINIO_ROOT_PASSWORD")
	if minioSecretKey == "" {
		minioSecretKey = "minioadmin"
	}
	minioBucket := os.Getenv("MINIO_BUCKET")
	if minioBucket == "" {
		minioBucket = "tax-ocr-active"
	}

	ctx := context.Background()

	store, err := db.NewStore(ctx, connString)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer store.Close()

	minioClient, err := storage.NewClient(minioEndpoint, minioAccessKey, minioSecretKey, minioBucket)
	if err != nil {
		log.Fatalf("failed to create minio client: %v", err)
	}
	if err := minioClient.EnsureBucket(ctx); err != nil {
		log.Fatalf("failed to ensure minio bucket: %v", err)
	}
	log.Printf("minio bucket %q ready", minioBucket)

	server := &http.Server{
		Addr:    addr,
		Handler: api.NewRouter(store, minioClient),
	}

	log.Printf("tax-ocr backend listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
