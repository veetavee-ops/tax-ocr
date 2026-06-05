package db

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
)

// invoiceCols is the common SELECT column list returned by all invoice queries.
const invoiceCols = `id, tenant_id, branch_id, COALESCE(document_import_id::text,''), file_path, file_hash,
	invoice_no, COALESCE(vendor_name,''), COALESCE(vendor_tax_id,''),
	COALESCE(invoice_doc_no,''), COALESCE(invoice_date,''),
	total_before_vat, vat_amount, total_amount, vat_math_ok, status,
	COALESCE(verified_by::text,''), verified_at,
	created_at, updated_at`

func scanInvoice(scan func(dest ...any) error, inv *Invoice) error {
	return scan(
		&inv.ID, &inv.TenantID, &inv.BranchID, &inv.DocumentImportID, &inv.FilePath, &inv.FileHash,
		&inv.InvoiceNo, &inv.VendorName, &inv.VendorTaxID,
		&inv.InvoiceDocNo, &inv.InvoiceDate,
		&inv.TotalBeforeVat, &inv.VatAmount, &inv.TotalAmount, &inv.VatMathOK, &inv.Status,
		&inv.VerifiedBy, &inv.VerifiedAt,
		&inv.CreatedAt, &inv.UpdatedAt,
	)
}

func (s *Store) ListInvoices(ctx context.Context, tenantID, status string) ([]Invoice, error) {
	query := `SELECT ` + invoiceCols + ` FROM invoices WHERE 1=1`
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
		if err := scanInvoice(rows.Scan, &inv); err != nil {
			return nil, err
		}
		items = append(items, inv)
	}
	return items, nil
}

func (s *Store) UpdateInvoiceItem(ctx context.Context, id string, quantity, unitPrice, totalPrice float64) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE invoice_items SET quantity=$2, unit_price=$3, total_price=$4, updated_at=NOW() WHERE id=$1`,
		id, quantity, unitPrice, totalPrice)
	return err
}

func (s *Store) DeleteInvoiceItemsByInvoiceID(ctx context.Context, invoiceID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM invoice_items WHERE invoice_id = $1`, invoiceID)
	return err
}

// UpdateInvoiceAmounts directly sets total_before_vat, vat_amount, and total_amount (no conditional — always overwrites).
func (s *Store) UpdateInvoiceAmounts(ctx context.Context, id string, totalBeforeVAT, vatAmount, totalAmount float64, vatMathOK bool) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE invoices SET total_before_vat=$2, vat_amount=$3, total_amount=$4, vat_math_ok=$5, updated_at=NOW() WHERE id=$1`,
		id, totalBeforeVAT, vatAmount, totalAmount, vatMathOK)
	return err
}

func (s *Store) GetInvoice(ctx context.Context, id string) (Invoice, error) {
	var inv Invoice
	err := scanInvoice(
		s.pool.QueryRow(ctx, `SELECT `+invoiceCols+` FROM invoices WHERE id = $1`, id).Scan,
		&inv,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, ErrNotFound
	}
	return inv, err
}

func (s *Store) CreateInvoice(ctx context.Context, input Invoice) (Invoice, error) {
	if input.TenantID == "" || input.BranchID == "" || input.FilePath == "" {
		return Invoice{}, ErrInvalidInput
	}
	// RETURNING uses a different column order: invoice_no comes second.
	const retCols = `id, invoice_no, tenant_id, branch_id, COALESCE(document_import_id::text,''), file_path, file_hash,
		COALESCE(vendor_name,''), COALESCE(vendor_tax_id,''),
		COALESCE(invoice_doc_no,''), COALESCE(invoice_date,''),
		total_before_vat, vat_amount, total_amount, vat_math_ok, status,
		COALESCE(verified_by::text,''), verified_at,
		created_at, updated_at`
	var inv Invoice
	err := s.pool.QueryRow(ctx,
		`INSERT INTO invoices (tenant_id, branch_id, document_import_id, file_path, file_hash, vendor_name, vendor_tax_id, invoice_doc_no, invoice_date, total_before_vat, vat_amount, total_amount, vat_math_ok)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		 RETURNING `+retCols,
		input.TenantID, input.BranchID, nullIfEmpty(input.DocumentImportID), input.FilePath, input.FileHash,
		input.VendorName, nullIfEmpty(input.VendorTaxID), nullIfEmpty(input.InvoiceDocNo), nullIfEmpty(input.InvoiceDate),
		input.TotalBeforeVat, input.VatAmount, input.TotalAmount, input.VatMathOK).
		Scan(
			&inv.ID, &inv.InvoiceNo, &inv.TenantID, &inv.BranchID, &inv.DocumentImportID, &inv.FilePath, &inv.FileHash,
			&inv.VendorName, &inv.VendorTaxID,
			&inv.InvoiceDocNo, &inv.InvoiceDate,
			&inv.TotalBeforeVat, &inv.VatAmount, &inv.TotalAmount, &inv.VatMathOK, &inv.Status,
			&inv.VerifiedBy, &inv.VerifiedAt,
			&inv.CreatedAt, &inv.UpdatedAt,
		)
	return inv, err
}

func (s *Store) UpdateInvoiceStatus(ctx context.Context, id, status string) (Invoice, error) {
	var inv Invoice
	err := scanInvoice(
		s.pool.QueryRow(ctx,
			`UPDATE invoices SET status = $2, updated_at = NOW() WHERE id = $1 RETURNING `+invoiceCols,
			id, status).Scan,
		&inv,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, ErrNotFound
	}
	return inv, err
}

func (s *Store) VerifyInvoice(ctx context.Context, id, userID string) (Invoice, error) {
	var inv Invoice
	err := scanInvoice(
		s.pool.QueryRow(ctx,
			`UPDATE invoices SET status = 'verified', verified_by = $2::uuid, verified_at = NOW(), updated_at = NOW()
			 WHERE id = $1 RETURNING `+invoiceCols,
			id, userID).Scan,
		&inv,
	)
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

type InvoiceUpdate struct {
	VendorName     string
	VendorTaxID    string
	InvoiceDocNo   string
	InvoiceDate    string
	TotalBeforeVAT float64
	VATAmount      float64
	TotalAmount    float64
	VATMathOK      bool
	Status         string
}

func (s *Store) UpdateInvoiceData(ctx context.Context, id string, u InvoiceUpdate) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE invoices SET
			vendor_name      = CASE WHEN $2 != '' THEN $2 ELSE vendor_name END,
			vendor_tax_id    = CASE WHEN $3 != '' THEN $3 ELSE vendor_tax_id END,
			invoice_doc_no   = CASE WHEN $4 != '' THEN $4 ELSE invoice_doc_no END,
			invoice_date     = CASE WHEN $5 != '' THEN $5 ELSE invoice_date END,
			total_before_vat = CASE WHEN $6 != 0  THEN $6 ELSE total_before_vat END,
			vat_amount       = CASE WHEN $7 != 0  THEN $7 ELSE vat_amount END,
			total_amount     = CASE WHEN $8 != 0  THEN $8 ELSE total_amount END,
			vat_math_ok      = $9,
			status           = CASE WHEN $10 != '' THEN $10 ELSE status END,
			updated_at       = NOW()
		 WHERE id = $1`,
		id, u.VendorName, u.VendorTaxID, u.InvoiceDocNo, u.InvoiceDate,
		u.TotalBeforeVAT, u.VATAmount, u.TotalAmount, u.VATMathOK, u.Status)
	return err
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

func (s *Store) DeleteInvoice(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM invoices WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
