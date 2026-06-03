package db

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Store) ListRules(ctx context.Context, tenantID string) ([]ClassificationRule, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, tenant_id, keyword, asset_type, source, confidence, created_at, updated_at
		 FROM classification_rules WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ClassificationRule
	for rows.Next() {
		var r ClassificationRule
		if err := rows.Scan(&r.ID, &r.TenantID, &r.Keyword, &r.AssetType, &r.Source, &r.Confidence, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	return items, nil
}

func (s *Store) GetRule(ctx context.Context, id string) (ClassificationRule, error) {
	var r ClassificationRule
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, keyword, asset_type, source, confidence, created_at, updated_at
		 FROM classification_rules WHERE id = $1`, id).
		Scan(&r.ID, &r.TenantID, &r.Keyword, &r.AssetType, &r.Source, &r.Confidence, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ClassificationRule{}, ErrNotFound
	}
	return r, err
}

func (s *Store) CreateRule(ctx context.Context, input ClassificationRule) (ClassificationRule, error) {
	if input.TenantID == "" || input.Keyword == "" || input.AssetType == "" {
		return ClassificationRule{}, ErrInvalidInput
	}
	var r ClassificationRule
	err := s.pool.QueryRow(ctx,
		`INSERT INTO classification_rules (tenant_id, keyword, asset_type, source, confidence)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, tenant_id, keyword, asset_type, source, confidence, created_at, updated_at`,
		input.TenantID, input.Keyword, input.AssetType, input.Source, input.Confidence).
		Scan(&r.ID, &r.TenantID, &r.Keyword, &r.AssetType, &r.Source, &r.Confidence, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ClassificationRule{}, ErrDuplicateKeyword
		}
		return ClassificationRule{}, err
	}
	return r, nil
}

func (s *Store) UpdateRule(ctx context.Context, id string, input ClassificationRule) (ClassificationRule, error) {
	var r ClassificationRule
	err := s.pool.QueryRow(ctx,
		`UPDATE classification_rules SET
			keyword    = COALESCE(NULLIF($2,''), keyword),
			asset_type = COALESCE(NULLIF($3,''), asset_type),
			source     = COALESCE(NULLIF($4,''), source),
			confidence = CASE WHEN $5 > 0 THEN $5 ELSE confidence END,
			updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, tenant_id, keyword, asset_type, source, confidence, created_at, updated_at`,
		id, input.Keyword, input.AssetType, input.Source, input.Confidence).
		Scan(&r.ID, &r.TenantID, &r.Keyword, &r.AssetType, &r.Source, &r.Confidence, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ClassificationRule{}, ErrNotFound
	}
	return r, err
}

func (s *Store) DeleteRule(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM classification_rules WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) TestRule(ctx context.Context, tenantID, keyword string) (*ClassificationRule, error) {
	// exact match first
	var r ClassificationRule
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, keyword, asset_type, source, confidence, created_at, updated_at
		 FROM classification_rules WHERE tenant_id = $1 AND LOWER(keyword) = LOWER($2) LIMIT 1`,
		tenantID, keyword).
		Scan(&r.ID, &r.TenantID, &r.Keyword, &r.AssetType, &r.Source, &r.Confidence, &r.CreatedAt, &r.UpdatedAt)
	if err == nil {
		return &r, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// partial match
	rows, err := s.pool.Query(ctx,
		`SELECT id, tenant_id, keyword, asset_type, source, confidence, created_at, updated_at
		 FROM classification_rules WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	kw := strings.ToLower(keyword)
	for rows.Next() {
		var cr ClassificationRule
		if err := rows.Scan(&cr.ID, &cr.TenantID, &cr.Keyword, &cr.AssetType, &cr.Source, &cr.Confidence, &cr.CreatedAt, &cr.UpdatedAt); err != nil {
			return nil, err
		}
		if strings.Contains(kw, strings.ToLower(cr.Keyword)) {
			return &cr, nil
		}
	}
	return nil, nil
}
