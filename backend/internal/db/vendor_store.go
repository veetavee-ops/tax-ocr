package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

const vendorCols = `id, tax_id,
	COALESCE(name,''), COALESCE(address,''), COALESCE(branch_code,''), COALESCE(branch_name,''), COALESCE(phone,''),
	verified, COALESCE(verified_by::text,''), verified_at,
	created_at, updated_at`

func scanVendor(scan func(dest ...any) error, v *Vendor) error {
	return scan(
		&v.ID, &v.TaxID,
		&v.Name, &v.Address, &v.BranchCode, &v.BranchName, &v.Phone,
		&v.Verified, &v.VerifiedBy, &v.VerifiedAt,
		&v.CreatedAt, &v.UpdatedAt,
	)
}

// FindVendorByTaxID looks up a vendor by their 13-digit tax ID.
// Returns ErrNotFound when no vendor record exists yet.
func (s *Store) FindVendorByTaxID(ctx context.Context, taxID string) (Vendor, error) {
	var v Vendor
	err := scanVendor(
		s.pool.QueryRow(ctx, `SELECT `+vendorCols+` FROM vendors WHERE tax_id = $1`, taxID).Scan,
		&v,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Vendor{}, ErrNotFound
	}
	return v, err
}

func (s *Store) GetVendor(ctx context.Context, id string) (Vendor, error) {
	var v Vendor
	err := scanVendor(
		s.pool.QueryRow(ctx, `SELECT `+vendorCols+` FROM vendors WHERE id = $1`, id).Scan,
		&v,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Vendor{}, ErrNotFound
	}
	return v, err
}

func (s *Store) ListVendors(ctx context.Context, verified *bool) ([]Vendor, error) {
	query := `SELECT ` + vendorCols + ` FROM vendors WHERE 1=1`
	args := []any{}
	if verified != nil {
		query += ` AND verified = $1`
		args = append(args, *verified)
	}
	query += ` ORDER BY name ASC, created_at DESC`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Vendor
	for rows.Next() {
		var v Vendor
		if err := scanVendor(rows.Scan, &v); err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, nil
}

// UpsertVendorFromOCR inserts a new vendor from OCR data (unverified),
// or returns the existing record if tax_id already exists.
// Never overwrites a verified vendor's name/address.
func (s *Store) UpsertVendorFromOCR(ctx context.Context, taxID, name, address, branchCode string) (Vendor, error) {
	if taxID == "" {
		return Vendor{}, ErrInvalidInput
	}
	var v Vendor
	err := scanVendor(
		s.pool.QueryRow(ctx, `
			INSERT INTO vendors (tax_id, name, address, branch_code)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (tax_id) DO UPDATE SET
				-- Only update name/address if vendor is not yet verified
				name        = CASE WHEN vendors.verified THEN vendors.name        ELSE EXCLUDED.name        END,
				address     = CASE WHEN vendors.verified THEN vendors.address     ELSE EXCLUDED.address     END,
				branch_code = CASE WHEN vendors.verified THEN vendors.branch_code ELSE EXCLUDED.branch_code END,
				updated_at  = NOW()
			RETURNING `+vendorCols,
			taxID, nullIfEmpty(name), nullIfEmpty(address), nullIfEmpty(branchCode),
		).Scan,
		&v,
	)
	return v, err
}

// VerifyVendor marks a vendor as verified and updates their canonical info.
func (s *Store) VerifyVendor(ctx context.Context, id, userID, name, address, branchCode, branchName, phone string) (Vendor, error) {
	var v Vendor
	err := scanVendor(
		s.pool.QueryRow(ctx, `
			UPDATE vendors SET
				name        = CASE WHEN $2 != '' THEN $2 ELSE name END,
				address     = CASE WHEN $3 != '' THEN $3 ELSE address END,
				branch_code = CASE WHEN $4 != '' THEN $4 ELSE branch_code END,
				branch_name = CASE WHEN $5 != '' THEN $5 ELSE branch_name END,
				phone       = CASE WHEN $6 != '' THEN $6 ELSE phone END,
				verified    = TRUE,
				verified_by = $7::uuid,
				verified_at = NOW(),
				updated_at  = NOW()
			WHERE id = $1
			RETURNING `+vendorCols,
			id, name, address, branchCode, branchName, phone, nullIfEmpty(userID),
		).Scan,
		&v,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Vendor{}, ErrNotFound
	}
	return v, err
}

// LinkInvoiceVendor sets vendor_id on an invoice.
func (s *Store) LinkInvoiceVendor(ctx context.Context, invoiceID, vendorID string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE invoices SET vendor_id = $2::uuid, updated_at = NOW() WHERE id = $1`,
		invoiceID, nullIfEmpty(vendorID))
	return err
}
