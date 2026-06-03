package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (s *Store) GetStorageConfig(ctx context.Context, tenantID string) (TenantStorageConfig, error) {
	var sc TenantStorageConfig
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, storage_type, COALESCE(gdrive_folder_id,''), COALESCE(gdrive_folder_url,''),
		  COALESCE(onedrive_folder_id,''), COALESCE(onedrive_folder_url,''),
		  owned_by, billing_type, monthly_fee, status, created_at, updated_at
		  FROM tenant_storage_config WHERE tenant_id = $1`, tenantID).
		Scan(&sc.ID, &sc.TenantID, &sc.StorageType, &sc.GdriveFolderID, &sc.GdriveFolderURL,
			&sc.OnedriveFolderID, &sc.OnedriveFolderURL,
			&sc.OwnedBy, &sc.BillingType, &sc.MonthlyFee, &sc.Status, &sc.CreatedAt, &sc.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return TenantStorageConfig{}, ErrNotFound
	}
	return sc, err
}

func (s *Store) CreateStorageConfig(ctx context.Context, input TenantStorageConfig) (TenantStorageConfig, error) {
	if input.TenantID == "" {
		return TenantStorageConfig{}, ErrInvalidInput
	}
	if input.Status == "" {
		input.Status = "active"
	}
	var sc TenantStorageConfig
	err := s.pool.QueryRow(ctx,
		`INSERT INTO tenant_storage_config
		  (tenant_id, storage_type, gdrive_folder_id, gdrive_folder_url, onedrive_folder_id, onedrive_folder_url, owned_by, billing_type, monthly_fee, status)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 RETURNING id, tenant_id, storage_type, COALESCE(gdrive_folder_id,''), COALESCE(gdrive_folder_url,''),
		   COALESCE(onedrive_folder_id,''), COALESCE(onedrive_folder_url,''),
		   owned_by, billing_type, monthly_fee, status, created_at, updated_at`,
		input.TenantID, input.StorageType,
		nullIfEmpty(input.GdriveFolderID), nullIfEmpty(input.GdriveFolderURL),
		nullIfEmpty(input.OnedriveFolderID), nullIfEmpty(input.OnedriveFolderURL),
		input.OwnedBy, input.BillingType, input.MonthlyFee, input.Status).
		Scan(&sc.ID, &sc.TenantID, &sc.StorageType, &sc.GdriveFolderID, &sc.GdriveFolderURL,
			&sc.OnedriveFolderID, &sc.OnedriveFolderURL,
			&sc.OwnedBy, &sc.BillingType, &sc.MonthlyFee, &sc.Status, &sc.CreatedAt, &sc.UpdatedAt)
	return sc, err
}

func (s *Store) UpdateStorageConfig(ctx context.Context, tenantID string, input TenantStorageConfig) (TenantStorageConfig, error) {
	var sc TenantStorageConfig
	err := s.pool.QueryRow(ctx,
		`UPDATE tenant_storage_config SET
			storage_type       = COALESCE(NULLIF($2,''), storage_type),
			gdrive_folder_id   = COALESCE(NULLIF($3,''), gdrive_folder_id),
			gdrive_folder_url  = COALESCE(NULLIF($4,''), gdrive_folder_url),
			onedrive_folder_id = COALESCE(NULLIF($5,''), onedrive_folder_id),
			onedrive_folder_url= COALESCE(NULLIF($6,''), onedrive_folder_url),
			owned_by           = COALESCE(NULLIF($7,''), owned_by),
			billing_type       = COALESCE(NULLIF($8,''), billing_type),
			monthly_fee        = CASE WHEN $9 > 0 THEN $9 ELSE monthly_fee END,
			status             = COALESCE(NULLIF($10,''), status),
			updated_at         = NOW()
		 WHERE tenant_id = $1
		 RETURNING id, tenant_id, storage_type, COALESCE(gdrive_folder_id,''), COALESCE(gdrive_folder_url,''),
		   COALESCE(onedrive_folder_id,''), COALESCE(onedrive_folder_url,''),
		   owned_by, billing_type, monthly_fee, status, created_at, updated_at`,
		tenantID, input.StorageType, input.GdriveFolderID, input.GdriveFolderURL,
		input.OnedriveFolderID, input.OnedriveFolderURL,
		input.OwnedBy, input.BillingType, input.MonthlyFee, input.Status).
		Scan(&sc.ID, &sc.TenantID, &sc.StorageType, &sc.GdriveFolderID, &sc.GdriveFolderURL,
			&sc.OnedriveFolderID, &sc.OnedriveFolderURL,
			&sc.OwnedBy, &sc.BillingType, &sc.MonthlyFee, &sc.Status, &sc.CreatedAt, &sc.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return TenantStorageConfig{}, ErrNotFound
	}
	return sc, err
}
