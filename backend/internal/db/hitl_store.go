package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (s *Store) ListHitlQueue(ctx context.Context, tenantID, status string) ([]HitlQueueItem, error) {
	query := `SELECT id, tenant_id, invoice_item_id, reason, status, COALESCE(resolved_by::text,''), created_at, updated_at
	           FROM hitl_queue WHERE tenant_id = $1`
	args := []any{tenantID}
	if status != "" {
		query += ` AND status = $2`
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []HitlQueueItem
	for rows.Next() {
		var h HitlQueueItem
		if err := rows.Scan(&h.ID, &h.TenantID, &h.InvoiceItemID, &h.Reason, &h.Status, &h.ResolvedBy, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, h)
	}
	return items, nil
}

func (s *Store) GetHitlItem(ctx context.Context, id string) (HitlQueueItem, error) {
	var h HitlQueueItem
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, invoice_item_id, reason, status, COALESCE(resolved_by::text,''), created_at, updated_at
		 FROM hitl_queue WHERE id = $1`, id).
		Scan(&h.ID, &h.TenantID, &h.InvoiceItemID, &h.Reason, &h.Status, &h.ResolvedBy, &h.CreatedAt, &h.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return HitlQueueItem{}, ErrNotFound
	}
	return h, err
}

func (s *Store) CreateHitlItem(ctx context.Context, input HitlQueueItem) (HitlQueueItem, error) {
	if input.TenantID == "" || input.InvoiceItemID == "" {
		return HitlQueueItem{}, ErrInvalidInput
	}
	var h HitlQueueItem
	err := s.pool.QueryRow(ctx,
		`INSERT INTO hitl_queue (tenant_id, invoice_item_id, reason)
		 VALUES ($1, $2, $3)
		 RETURNING id, tenant_id, invoice_item_id, reason, status, COALESCE(resolved_by::text,''), created_at, updated_at`,
		input.TenantID, input.InvoiceItemID, input.Reason).
		Scan(&h.ID, &h.TenantID, &h.InvoiceItemID, &h.Reason, &h.Status, &h.ResolvedBy, &h.CreatedAt, &h.UpdatedAt)
	return h, err
}

func (s *Store) ResolveHitlItem(ctx context.Context, id, resolvedBy string) (HitlQueueItem, error) {
	var h HitlQueueItem
	err := s.pool.QueryRow(ctx,
		`UPDATE hitl_queue SET status = 'resolved', resolved_by = $2, updated_at = NOW() WHERE id = $1
		 RETURNING id, tenant_id, invoice_item_id, reason, status, COALESCE(resolved_by::text,''), created_at, updated_at`,
		id, nullIfEmpty(resolvedBy)).
		Scan(&h.ID, &h.TenantID, &h.InvoiceItemID, &h.Reason, &h.Status, &h.ResolvedBy, &h.CreatedAt, &h.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return HitlQueueItem{}, ErrNotFound
	}
	return h, err
}

func (s *Store) RejectHitlItem(ctx context.Context, id string) (HitlQueueItem, error) {
	var h HitlQueueItem
	err := s.pool.QueryRow(ctx,
		`UPDATE hitl_queue SET status = 'rejected', updated_at = NOW() WHERE id = $1
		 RETURNING id, tenant_id, invoice_item_id, reason, status, COALESCE(resolved_by::text,''), created_at, updated_at`,
		id).
		Scan(&h.ID, &h.TenantID, &h.InvoiceItemID, &h.Reason, &h.Status, &h.ResolvedBy, &h.CreatedAt, &h.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return HitlQueueItem{}, ErrNotFound
	}
	return h, err
}
