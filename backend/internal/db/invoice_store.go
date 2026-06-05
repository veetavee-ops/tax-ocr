package db

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
)

// invoiceCols is the common SELECT column list returned by all invoice queries.
const invoiceCols = `id, tenant_id, branch_id, COALESCE(document_import_id::text,''), file_path, file_hash,
	invoice_no,
	COALESCE(doc_type,'tax_invoice'), COALESCE(vat_inclusive,false), COALESCE(vat_rate,7.00),
	COALESCE(vendor_name,''), COALESCE(vendor_tax_id,''), COALESCE(vendor_address,''), COALESCE(vendor_branch_code,''),
	COALESCE(buyer_name,''), COALESCE(buyer_tax_id,''), COALESCE(buyer_address,''), COALESCE(buyer_branch_code,''),
	COALESCE(invoice_doc_no,''), COALESCE(invoice_date,''),
	COALESCE(vat_exempt_amount,0), COALESCE(vat_inclusive_subtotal,0), COALESCE(discount_amount,0),
	total_before_vat, vat_amount, total_amount, vat_math_ok, status,
	COALESCE(verified_by::text,''), verified_at,
	created_at, updated_at`

func scanInvoice(scan func(dest ...any) error, inv *Invoice) error {
	return scan(
		&inv.ID, &inv.TenantID, &inv.BranchID, &inv.DocumentImportID, &inv.FilePath, &inv.FileHash,
		&inv.InvoiceNo,
		&inv.DocType, &inv.VatInclusive, &inv.VatRate,
		&inv.VendorName, &inv.VendorTaxID, &inv.VendorAddress, &inv.VendorBranchCode,
		&inv.BuyerName, &inv.BuyerTaxID, &inv.BuyerAddress, &inv.BuyerBranchCode,
		&inv.InvoiceDocNo, &inv.InvoiceDate,
		&inv.VatExemptAmount, &inv.VatInclusiveSubtotal, &inv.DiscountAmount,
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
	var inv Invoice
	err := scanInvoice(
		s.pool.QueryRow(ctx,
			`INSERT INTO invoices (tenant_id, branch_id, document_import_id, file_path, file_hash,
				doc_type, vat_inclusive, vat_rate,
				vendor_name, vendor_tax_id, vendor_address, vendor_branch_code,
				buyer_name, buyer_tax_id, buyer_address, buyer_branch_code,
				invoice_doc_no, invoice_date,
				vat_exempt_amount, vat_inclusive_subtotal, discount_amount,
				total_before_vat, vat_amount, total_amount, vat_math_ok)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)
			 RETURNING `+invoiceCols,
			input.TenantID, input.BranchID, nullIfEmpty(input.DocumentImportID), input.FilePath, input.FileHash,
			nullIfEmpty(input.DocType), input.VatInclusive, input.VatRate,
			input.VendorName, nullIfEmpty(input.VendorTaxID), nullIfEmpty(input.VendorAddress), nullIfEmpty(input.VendorBranchCode),
			nullIfEmpty(input.BuyerName), nullIfEmpty(input.BuyerTaxID), nullIfEmpty(input.BuyerAddress), nullIfEmpty(input.BuyerBranchCode),
			nullIfEmpty(input.InvoiceDocNo), nullIfEmpty(input.InvoiceDate),
			input.VatExemptAmount, input.VatInclusiveSubtotal, input.DiscountAmount,
			input.TotalBeforeVat, input.VatAmount, input.TotalAmount, input.VatMathOK,
		).Scan,
		&inv,
	)
	return inv, err
}

// FullUpdateInvoice overwrites all editable fields (for manual UI edits).
// vatMathOK must be pre-computed by the caller.
func (s *Store) FullUpdateInvoice(ctx context.Context, id string, u InvoiceUpdate) (Invoice, error) {
	var inv Invoice
	err := scanInvoice(
		s.pool.QueryRow(ctx,
			`UPDATE invoices SET
				doc_type=$2, vat_inclusive=$3, vat_rate=$4,
				vendor_name=$5, vendor_tax_id=$6, vendor_address=$7, vendor_branch_code=$8,
				buyer_name=$9, buyer_tax_id=$10, buyer_address=$11, buyer_branch_code=$12,
				invoice_doc_no=$13, invoice_date=$14,
				vat_exempt_amount=$15, vat_inclusive_subtotal=$16, discount_amount=$17,
				total_before_vat=$18, vat_amount=$19, total_amount=$20, vat_math_ok=$21,
				updated_at=NOW()
			 WHERE id=$1 RETURNING `+invoiceCols,
			id,
			nullIfEmpty(u.DocType), u.VatInclusive, u.VatRate,
			u.VendorName, nullIfEmpty(u.VendorTaxID), nullIfEmpty(u.VendorAddress), nullIfEmpty(u.VendorBranchCode),
			nullIfEmpty(u.BuyerName), nullIfEmpty(u.BuyerTaxID), nullIfEmpty(u.BuyerAddress), nullIfEmpty(u.BuyerBranchCode),
			nullIfEmpty(u.InvoiceDocNo), nullIfEmpty(u.InvoiceDate),
			u.VatExemptAmount, u.VatInclusiveSubtotal, u.DiscountAmount,
			u.TotalBeforeVAT, u.VATAmount, u.TotalAmount, u.VATMathOK,
		).Scan,
		&inv,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, ErrNotFound
	}
	return inv, err
}

// UpdateInvoiceAmounts directly sets financial amounts — always overwrites.
func (s *Store) UpdateInvoiceAmounts(ctx context.Context, id string, totalBeforeVAT, vatAmount, totalAmount float64, vatMathOK bool) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE invoices SET total_before_vat=$2, vat_amount=$3, total_amount=$4, vat_math_ok=$5, updated_at=NOW() WHERE id=$1`,
		id, totalBeforeVAT, vatAmount, totalAmount, vatMathOK)
	return err
}

func (s *Store) UpdateInvoiceStatus(ctx context.Context, id, status string) (Invoice, error) {
	var inv Invoice
	err := scanInvoice(
		s.pool.QueryRow(ctx,
			`UPDATE invoices SET status=$2, updated_at=NOW() WHERE id=$1 RETURNING `+invoiceCols,
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
			`UPDATE invoices SET status='verified', verified_by=$2::uuid, verified_at=NOW(), updated_at=NOW()
			 WHERE id=$1 RETURNING `+invoiceCols,
			id, userID).Scan,
		&inv,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, ErrNotFound
	}
	return inv, err
}

// InvoiceUpdate holds OCR-extracted values for UpdateInvoiceData.
type InvoiceUpdate struct {
	DocType              string
	VatInclusive         bool
	VatRate              float64
	VendorName           string
	VendorTaxID          string
	VendorAddress        string
	VendorBranchCode     string
	BuyerName            string
	BuyerTaxID           string
	BuyerAddress         string
	BuyerBranchCode      string
	InvoiceDocNo         string
	InvoiceDate          string
	VatExemptAmount      float64
	VatInclusiveSubtotal float64
	DiscountAmount       float64
	TotalBeforeVAT       float64
	VATAmount            float64
	TotalAmount          float64
	VATMathOK            bool
	Status               string
}

func (s *Store) UpdateInvoiceData(ctx context.Context, id string, u InvoiceUpdate) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE invoices SET
			doc_type              = CASE WHEN $2  != '' THEN $2  ELSE doc_type END,
			vat_inclusive         = $3,
			vat_rate              = CASE WHEN $4  != 0  THEN $4  ELSE vat_rate END,
			vendor_name           = CASE WHEN $5  != '' THEN $5  ELSE vendor_name END,
			vendor_tax_id         = CASE WHEN $6  != '' THEN $6  ELSE vendor_tax_id END,
			vendor_address        = CASE WHEN $7  != '' THEN $7  ELSE vendor_address END,
			vendor_branch_code    = CASE WHEN $8  != '' THEN $8  ELSE vendor_branch_code END,
			buyer_name            = CASE WHEN $9  != '' THEN $9  ELSE buyer_name END,
			buyer_tax_id          = CASE WHEN $10 != '' THEN $10 ELSE buyer_tax_id END,
			buyer_address         = CASE WHEN $11 != '' THEN $11 ELSE buyer_address END,
			buyer_branch_code     = CASE WHEN $12 != '' THEN $12 ELSE buyer_branch_code END,
			invoice_doc_no        = CASE WHEN $13 != '' THEN $13 ELSE invoice_doc_no END,
			invoice_date          = CASE WHEN $14 != '' THEN $14 ELSE invoice_date END,
			vat_exempt_amount     = CASE WHEN $15 != 0  THEN $15 ELSE vat_exempt_amount END,
			vat_inclusive_subtotal= CASE WHEN $16 != 0  THEN $16 ELSE vat_inclusive_subtotal END,
			discount_amount       = CASE WHEN $17 != 0  THEN $17 ELSE discount_amount END,
			total_before_vat      = CASE WHEN $18 != 0  THEN $18 ELSE total_before_vat END,
			vat_amount            = CASE WHEN $19 != 0  THEN $19 ELSE vat_amount END,
			total_amount          = CASE WHEN $20 != 0  THEN $20 ELSE total_amount END,
			vat_math_ok           = $21,
			status                = CASE WHEN $22 != '' THEN $22 ELSE status END,
			updated_at            = NOW()
		 WHERE id = $1`,
		id,
		u.DocType, u.VatInclusive, u.VatRate,
		u.VendorName, u.VendorTaxID, u.VendorAddress, u.VendorBranchCode,
		u.BuyerName, u.BuyerTaxID, u.BuyerAddress, u.BuyerBranchCode,
		u.InvoiceDocNo, u.InvoiceDate,
		u.VatExemptAmount, u.VatInclusiveSubtotal, u.DiscountAmount,
		u.TotalBeforeVAT, u.VATAmount, u.TotalAmount, u.VATMathOK, u.Status)
	return err
}

// itemCols is the common SELECT column list for invoice_items.
const itemCols = `id, tenant_id, branch_id, invoice_id,
	COALESCE(product_code,''), description, COALESCE(unit,''),
	quantity, unit_price, COALESCE(discount,0), total_price,
	asset_type, classified_by, created_at, updated_at`

func scanItem(scan func(dest ...any) error, it *InvoiceItem) error {
	return scan(
		&it.ID, &it.TenantID, &it.BranchID, &it.InvoiceID,
		&it.ProductCode, &it.Description, &it.Unit,
		&it.Quantity, &it.UnitPrice, &it.Discount, &it.TotalPrice,
		&it.AssetType, &it.ClassifiedBy, &it.CreatedAt, &it.UpdatedAt,
	)
}

func (s *Store) ListInvoiceItems(ctx context.Context, invoiceID string) ([]InvoiceItem, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+itemCols+` FROM invoice_items WHERE invoice_id=$1 ORDER BY created_at`, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []InvoiceItem
	for rows.Next() {
		var it InvoiceItem
		if err := scanItem(rows.Scan, &it); err != nil {
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
	err := scanItem(
		s.pool.QueryRow(ctx,
			`INSERT INTO invoice_items (tenant_id, branch_id, invoice_id, product_code, description, unit,
				quantity, unit_price, discount, total_price, asset_type, classified_by)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			 RETURNING `+itemCols,
			input.TenantID, input.BranchID, input.InvoiceID,
			nullIfEmpty(input.ProductCode), input.Description, nullIfEmpty(input.Unit),
			input.Quantity, input.UnitPrice, input.Discount, input.TotalPrice,
			input.AssetType, input.ClassifiedBy,
		).Scan,
		&it,
	)
	return it, err
}

func (s *Store) UpdateInvoiceItem(ctx context.Context, id string, quantity, unitPrice, discount, totalPrice float64) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE invoice_items SET quantity=$2, unit_price=$3, discount=$4, total_price=$5, updated_at=NOW() WHERE id=$1`,
		id, quantity, unitPrice, discount, totalPrice)
	return err
}

func (s *Store) DeleteInvoiceItemsByInvoiceID(ctx context.Context, invoiceID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM invoice_items WHERE invoice_id=$1`, invoiceID)
	return err
}

func (s *Store) DeleteInvoice(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM invoices WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
