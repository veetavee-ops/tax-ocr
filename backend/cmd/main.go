package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"tax-ocr/backend/internal/api"
	"tax-ocr/backend/internal/archive"
	"tax-ocr/backend/internal/classify"
	"tax-ocr/backend/internal/db"
	"tax-ocr/backend/internal/ocr"
	"tax-ocr/backend/internal/queue"
	rev "tax-ocr/backend/internal/reviewer"
	"tax-ocr/backend/internal/storage"
)

func main() {
	addr := envOr("APP_ADDR", ":8080")
	connString := envOr("DATABASE_URL", "postgres://tax_ocr:tax_ocr_dev@localhost:5433/tax_ocr?sslmode=disable")
	redisAddr := envOr("REDIS_ADDR", "localhost:6380")
	minioEndpoint := envOr("MINIO_ENDPOINT", "localhost:9000")
	minioAccessKey := envOr("MINIO_ROOT_USER", "minioadmin")
	minioSecretKey := envOr("MINIO_ROOT_PASSWORD", "minioadmin")
	minioBucket := envOr("MINIO_BUCKET", "tax-ocr-active")
	openAIKey := os.Getenv("OPENAI_API_KEY")
	gcvKey := os.Getenv("GCV_API_KEY")
	lineToken := os.Getenv("LINE_CHANNEL_TOKEN")
	expireMinutes, _ := strconv.Atoi(envOr("REVIEWER_EXPIRE_MINUTES", "30"))

	ctx := context.Background()

	migrationsDir := envOr("MIGRATIONS_DIR", "../database/migrations")
	if err := db.RunMigrations(ctx, connString, migrationsDir); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	store, err := db.NewStore(ctx, connString)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer store.Close()

	minioClient, err := storage.NewClient(minioEndpoint, minioAccessKey, minioSecretKey, minioBucket)
	if err != nil {
		log.Fatalf("minio: %v", err)
	}
	if err := minioClient.EnsureBucket(ctx); err != nil {
		log.Fatalf("minio bucket: %v", err)
	}
	log.Printf("minio bucket %q ready", minioBucket)

	ocrSvc := ocr.NewServiceWithConfig(ocr.Config{
		OpenAIKey: openAIKey,
		GCVKey:    gcvKey,
	})

	classifySvc := classify.NewServiceWithConfig(store, classify.Config{
		OpenAIKey: openAIKey,
	})

	// Override OCR config from DB (DB takes priority over env vars)
	if dbCfg, err := buildOCRConfigFromDB(ctx, store); err == nil {
		ocrSvc.UpdateConfig(dbCfg)
	}

	queueClient := queue.NewClient(redisAddr)
	defer queueClient.Close()

	lineClient := rev.NewLineClient(lineToken)
	reviewerSvc := rev.NewService(store, lineClient, expireMinutes)
	reviewerSvc.RunExpiryChecker(ctx)
	log.Printf("reviewer expiry checker started (expire=%dm)", expireMinutes)

	archiveScheduler := archive.NewScheduler(store, time.Hour)
	archiveScheduler.Run(ctx)
	log.Printf("archive scheduler started (interval=1h)")

	worker := queue.NewWorker(
		queue.WorkerConfig{RedisAddr: redisAddr, Concurrency: 5},
		ocrSvc, classifySvc, store, minioClient, reviewerSvc, lineClient,
	)
	if err := worker.Start(); err != nil {
		log.Fatalf("worker start: %v", err)
	}
	defer worker.Shutdown()
	log.Printf("asynq worker started (redis=%s)", redisAddr)

	server := &http.Server{
		Addr: addr,
		Handler: api.NewRouter(store, minioClient, queueClient, ocrSvc, api.ServerConfig{
			LineToken:  lineToken,
			LineSecret: os.Getenv("LINE_CHANNEL_SECRET"),
		}),
	}

	log.Printf("tax-ocr backend listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func buildOCRConfigFromDB(ctx context.Context, store *db.Store) (ocr.Config, error) {
	configs, err := store.ListOCRConfigs(ctx)
	if err != nil {
		return ocr.Config{}, err
	}
	cfg := ocr.Config{}
	for _, c := range configs {
		if !c.Enabled || c.APIKey == "" {
			continue
		}
		switch c.Provider {
		case "openai":
			cfg.OpenAIKey = c.APIKey
		case "gcv":
			cfg.GCVKey = c.APIKey
		}
	}
	if cfg.OpenAIKey == "" && cfg.GCVKey == "" {
		return ocr.Config{}, errors.New("no ocr keys in db")
	}
	return cfg, nil
}
