package db

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
)

func (s *Store) ListInvoices(ctx context.Context, tenantID, status string) ([]Invoice, error) {
	query := `SELECT id, tenant_id, branch_id, COALESCE(document_import_id::text,''), file_path, file_hash,
	           COALESCE(vendor_tax_id,''), total_before_vat, vat_amount, total_amount, status, created_at, updated_at
	           FROM invoices WHERE 1=1`
	args := []any{}
	i := 1
	if tenantID != "" {
		query += ` AND tenant_id = $` + strconv.Itoa(i)
		args = append(args, tenantID)
		i++
	}
	if status != "" {
		query += ` AND status = $` + strconv.Itoa(i)
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Invoice
	for rows.Next() {
		var inv Invoice
		if err := rows.Scan(&inv.ID, &inv.TenantID, &inv.BranchID, &inv.DocumentImportID, &inv.FilePath, &inv.FileHash,
			&inv.VendorTaxID, &inv.TotalBeforeVat, &inv.VatAmount, &inv.TotalAmount, &inv.Status, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, inv)
	}
	return items, nil
}

func (s *Store) GetInvoice(ctx context.Context, id string) (Invoice, error) {
	var inv Invoice
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, branch_id, COALESCE(document_import_id::text,''), file_path, file_hash,
		  COALESCE(vendor_tax_id,''), total_before_vat, vat_amount, total_amount, status, created_at, updated_at
		  FROM invoices WHERE id = $1`, id).
		Scan(&inv.ID, &inv.TenantID, &inv.BranchID, &inv.DocumentImportID, &inv.FilePath, &inv.FileHash,
			&inv.VendorTaxID, &inv.TotalBeforeVat, &inv.VatAmount, &inv.TotalAmount, &inv.Status, &inv.CreatedAt, &inv.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, ErrNotFound
	}
	return inv, err
}

func (s *Store) CreateInvoice(ctx context.Context, input Invoice) (Invoice, error) {
	if input.TenantID == "" || input.BranchID == "" || input.FilePath == "" {
		return Invoice{}, ErrInvalidInput
	}
	var inv Invoice
	err := s.pool.QueryRow(ctx,
		`INSERT INTO invoices (tenant_id, branch_id, document_import_id, file_path, file_hash, vendor_tax_id, total_before_vat, vat_amount, total_amount)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, tenant_id, branch_id, COALESCE(document_import_id::text,''), file_path, file_hash,
		   COALESCE(vendor_tax_id,''), total_before_vat, vat_amount, total_amount, status, created_at, updated_at`,
		input.TenantID, input.BranchID, nullIfEmpty(input.DocumentImportID), input.FilePath, input.FileHash,
		nullIfEmpty(input.VendorTaxID), input.TotalBeforeVat, input.VatAmount, input.TotalAmount).
		Scan(&inv.ID, &inv.TenantID, &inv.BranchID, &inv.DocumentImportID, &inv.FilePath, &inv.FileHash,
			&inv.VendorTaxID, &inv.TotalBeforeVat, &inv.VatAmount, &inv.TotalAmount, &inv.Status, &inv.CreatedAt, &inv.UpdatedAt)
	return inv, err
}

func (s *Store) UpdateInvoiceStatus(ctx context.Context, id, status string) (Invoice, error) {
	var inv Invoice
	err := s.pool.QueryRow(ctx,
		`UPDATE invoices SET status = $2, updated_at = NOW() WHERE id = $1
		 RETURNING id, tenant_id, branch_id, COALESCE(document_import_id::text,''), file_path, file_hash,
		   COALESCE(vendor_tax_id,''), total_before_vat, vat_amount, total_amount, status, created_at, updated_at`,
		id, status).
		Scan(&inv.ID, &inv.TenantID, &inv.BranchID, &inv.DocumentImportID, &inv.FilePath, &inv.FileHash,
			&inv.VendorTaxID, &inv.TotalBeforeVat, &inv.VatAmount, &inv.TotalAmount, &inv.Status, &inv.CreatedAt, &inv.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, ErrNotFound
	}
	return inv, err
}

func (s *Store) ListInvoiceItems(ctx context.Context, invoiceID string) ([]InvoiceItem, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, tenant_id, branch_id, invoice_id, description, quantity, unit_price, total_price, asset_type, classified_by, created_at, updated_at
		 FROM invoice_items WHERE invoice_id = $1 ORDER BY created_at`, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []InvoiceItem
	for rows.Next() {
		var it InvoiceItem
		if err := rows.Scan(&it.ID, &it.TenantID, &it.BranchID, &it.InvoiceID, &it.Description, &it.Quantity,
			&it.UnitPrice, &it.TotalPrice, &it.AssetType, &it.ClassifiedBy, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

func (s *Store) CreateInvoiceItem(ctx context.Context, input InvoiceItem) (InvoiceItem, error) {
	if input.InvoiceID == "" || input.Description == "" {
		return InvoiceItem{}, ErrInvalidInput
	}
	var it InvoiceItem
	err := s.pool.QueryRow(ctx,
		`INSERT INTO invoice_items (tenant_id, branch_id, invoice_id, description, quantity, unit_price, total_price, asset_type, classified_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, tenant_id, branch_id, invoice_id, description, quantity, unit_price, total_price, asset_type, classified_by, created_at, updated_at`,
		input.TenantID, input.BranchID, input.InvoiceID, input.Description, input.Quantity,
		input.UnitPrice, input.TotalPrice, input.AssetType, input.ClassifiedBy).
		Scan(&it.ID, &it.TenantID, &it.BranchID, &it.InvoiceID, &it.Description, &it.Quantity,
			&it.UnitPrice, &it.TotalPrice, &it.AssetType, &it.ClassifiedBy, &it.CreatedAt, &it.UpdatedAt)
	return it, err
}
