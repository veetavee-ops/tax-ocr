package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (s *Store) ListAuditLogs(ctx context.Context, tenantID string) ([]AuditLog, error) {
	query := `SELECT id, tenant_id, COALESCE(branch_id::text,''), COALESCE(user_id::text,''),
	           action, COALESCE(entity_type,''), COALESCE(entity_id::text,''),
	           metadata, COALESCE(ip_address,''), COALESCE(device_info,''), created_at, updated_at
	           FROM audit_logs WHERE 1=1`
	args := []any{}
	if tenantID != "" {
		query += ` AND tenant_id = $1`
		args = append(args, tenantID)
	}
	query += " ORDER BY created_at DESC LIMIT 500"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []AuditLog
	for rows.Next() {
		var a AuditLog
		if err := rows.Scan(&a.ID, &a.TenantID, &a.BranchID, &a.UserID,
			&a.Action, &a.EntityType, &a.EntityID,
			&a.Metadata, &a.IPAddress, &a.DeviceInfo, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, nil
}

func (s *Store) GetAuditLog(ctx context.Context, id string) (AuditLog, error) {
	var a AuditLog
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, COALESCE(branch_id::text,''), COALESCE(user_id::text,''),
		  action, COALESCE(entity_type,''), COALESCE(entity_id::text,''),
		  metadata, COALESCE(ip_address,''), COALESCE(device_info,''), created_at, updated_at
		  FROM audit_logs WHERE id = $1`, id).
		Scan(&a.ID, &a.TenantID, &a.BranchID, &a.UserID,
			&a.Action, &a.EntityType, &a.EntityID,
			&a.Metadata, &a.IPAddress, &a.DeviceInfo, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return AuditLog{}, ErrNotFound
	}
	return a, err
}

func (s *Store) CreateAuditLog(ctx context.Context, input AuditLog) (AuditLog, error) {
	if input.TenantID == "" || input.Action == "" {
		return AuditLog{}, ErrInvalidInput
	}
	var a AuditLog
	err := s.pool.QueryRow(ctx,
		`INSERT INTO audit_logs (tenant_id, branch_id, user_id, action, entity_type, entity_id, metadata, ip_address, device_info)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, tenant_id, COALESCE(branch_id::text,''), COALESCE(user_id::text,''),
		   action, COALESCE(entity_type,''), COALESCE(entity_id::text,''),
		   metadata, COALESCE(ip_address,''), COALESCE(device_info,''), created_at, updated_at`,
		input.TenantID, nullIfEmpty(input.BranchID), nullIfEmpty(input.UserID),
		input.Action, nullIfEmpty(input.EntityType), nullIfEmpty(input.EntityID),
		input.Metadata, nullIfEmpty(input.IPAddress), nullIfEmpty(input.DeviceInfo)).
		Scan(&a.ID, &a.TenantID, &a.BranchID, &a.UserID,
			&a.Action, &a.EntityType, &a.EntityID,
			&a.Metadata, &a.IPAddress, &a.DeviceInfo, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}
