package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (s *Store) ListOCRConfigs(ctx context.Context) ([]OcrConfig, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, provider, api_key, enabled, COALESCE(updated_by::text,''), created_at, updated_at
		 FROM ocr_config ORDER BY provider`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []OcrConfig
	for rows.Next() {
		var c OcrConfig
		if err := rows.Scan(&c.ID, &c.Provider, &c.APIKey, &c.Enabled, &c.UpdatedBy, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}

func (s *Store) GetOCRConfig(ctx context.Context, provider string) (OcrConfig, error) {
	var c OcrConfig
	err := s.pool.QueryRow(ctx,
		`SELECT id, provider, api_key, enabled, COALESCE(updated_by::text,''), created_at, updated_at
		 FROM ocr_config WHERE provider = $1`, provider).
		Scan(&c.ID, &c.Provider, &c.APIKey, &c.Enabled, &c.UpdatedBy, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return OcrConfig{}, ErrNotFound
	}
	return c, err
}

func (s *Store) UpsertOCRConfig(ctx context.Context, provider, apiKey string, enabled bool, updatedBy string) (OcrConfig, error) {
	var c OcrConfig
	err := s.pool.QueryRow(ctx,
		`INSERT INTO ocr_config (provider, api_key, enabled, updated_by)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (provider) DO UPDATE SET
		   api_key    = CASE WHEN $2 != '' THEN $2 ELSE ocr_config.api_key END,
		   enabled    = $3,
		   updated_by = $4,
		   updated_at = NOW()
		 RETURNING id, provider, api_key, enabled, COALESCE(updated_by::text,''), created_at, updated_at`,
		provider, apiKey, enabled, nullIfEmpty(updatedBy)).
		Scan(&c.ID, &c.Provider, &c.APIKey, &c.Enabled, &c.UpdatedBy, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}
