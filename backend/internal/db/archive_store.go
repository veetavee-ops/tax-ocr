package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (s *Store) ListArchiveLogs(ctx context.Context, tenantID string) ([]ArchiveLog, error) {
	query := `SELECT id, tenant_id, entity_type, entity_id::text, archived_at, archive_path, status, created_at, updated_at
	           FROM archive_logs WHERE 1=1`
	args := []any{}
	if tenantID != "" {
		query += ` AND tenant_id = $1`
		args = append(args, tenantID)
	}
	query += " ORDER BY archived_at DESC"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ArchiveLog
	for rows.Next() {
		var a ArchiveLog
		if err := rows.Scan(&a.ID, &a.TenantID, &a.EntityType, &a.EntityID,
			&a.ArchivedAt, &a.ArchivePath, &a.Status, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, nil
}

func (s *Store) RestoreArchive(ctx context.Context, id string) (ArchiveLog, error) {
	var a ArchiveLog
	err := s.pool.QueryRow(ctx,
		`UPDATE archive_logs SET status = 'restored', updated_at = NOW() WHERE id = $1
		 RETURNING id, tenant_id, entity_type, entity_id::text, archived_at, archive_path, status, created_at, updated_at`,
		id).
		Scan(&a.ID, &a.TenantID, &a.EntityType, &a.EntityID,
			&a.ArchivedAt, &a.ArchivePath, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ArchiveLog{}, ErrNotFound
	}
	return a, err
}

func (s *Store) ListArchivePolicies(ctx context.Context, tenantID string) ([]ArchivePolicy, error) {
	query := `SELECT id, tenant_id, active_days, archive_days, created_at, updated_at FROM archive_policies WHERE 1=1`
	args := []any{}
	if tenantID != "" {
		query += ` AND tenant_id = $1`
		args = append(args, tenantID)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ArchivePolicy
	for rows.Next() {
		var p ArchivePolicy
		if err := rows.Scan(&p.ID, &p.TenantID, &p.ActiveDays, &p.ArchiveDays, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, nil
}

func (s *Store) CreateArchivePolicy(ctx context.Context, input ArchivePolicy) (ArchivePolicy, error) {
	if input.TenantID == "" {
		return ArchivePolicy{}, ErrInvalidInput
	}
	if input.ActiveDays == 0 {
		input.ActiveDays = 90
	}
	if input.ArchiveDays == 0 {
		input.ArchiveDays = 365
	}
	var p ArchivePolicy
	err := s.pool.QueryRow(ctx,
		`INSERT INTO archive_policies (tenant_id, active_days, archive_days)
		 VALUES ($1, $2, $3)
		 RETURNING id, tenant_id, active_days, archive_days, created_at, updated_at`,
		input.TenantID, input.ActiveDays, input.ArchiveDays).
		Scan(&p.ID, &p.TenantID, &p.ActiveDays, &p.ArchiveDays, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (s *Store) UpdateArchivePolicy(ctx context.Context, id string, input ArchivePolicy) (ArchivePolicy, error) {
	var p ArchivePolicy
	err := s.pool.QueryRow(ctx,
		`UPDATE archive_policies SET
			active_days  = CASE WHEN $2 > 0 THEN $2 ELSE active_days END,
			archive_days = CASE WHEN $3 > 0 THEN $3 ELSE archive_days END,
			updated_at   = NOW()
		 WHERE id = $1
		 RETURNING id, tenant_id, active_days, archive_days, created_at, updated_at`,
		id, input.ActiveDays, input.ArchiveDays).
		Scan(&p.ID, &p.TenantID, &p.ActiveDays, &p.ArchiveDays, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ArchivePolicy{}, ErrNotFound
	}
	return p, err
}
