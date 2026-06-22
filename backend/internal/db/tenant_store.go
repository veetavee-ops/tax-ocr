package db

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const tenantCols = `id, name, tax_id, COALESCE(address,''), status, COALESCE(business_type,'service'), created_at, updated_at`

func scanTenant(scan func(dest ...any) error, t *Tenant) error {
	return scan(&t.ID, &t.Name, &t.TaxID, &t.Address, &t.Status, &t.BusinessType, &t.CreatedAt, &t.UpdatedAt)
}

func (s *Store) ListTenants(ctx context.Context) ([]Tenant, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+tenantCols+` FROM tenants ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Tenant
	for rows.Next() {
		var t Tenant
		if err := scanTenant(rows.Scan, &t); err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	return items, nil
}

func (s *Store) GetTenant(ctx context.Context, id string) (Tenant, error) {
	var t Tenant
	err := scanTenant(s.pool.QueryRow(ctx, `SELECT `+tenantCols+` FROM tenants WHERE id = $1`, id).Scan, &t)
	if errors.Is(err, pgx.ErrNoRows) {
		return Tenant{}, ErrNotFound
	}
	return t, err
}

const branchCols = `id, tenant_id, name, code, COALESCE(address,''), COALESCE(phone,''), status, created_at, updated_at`

func scanBranch(scan func(dest ...any) error, b *Branch) error {
	return scan(&b.ID, &b.TenantID, &b.Name, &b.Code, &b.Address, &b.Phone, &b.Status, &b.CreatedAt, &b.UpdatedAt)
}

func (s *Store) GetBranch(ctx context.Context, id string) (Branch, error) {
	var b Branch
	err := scanBranch(s.pool.QueryRow(ctx, `SELECT `+branchCols+` FROM branches WHERE id = $1`, id).Scan, &b)
	if errors.Is(err, pgx.ErrNoRows) {
		return Branch{}, ErrNotFound
	}
	return b, err
}

func (s *Store) CreateTenant(ctx context.Context, name, taxID, businessType, address string) (Tenant, error) {
	if name == "" || taxID == "" {
		return Tenant{}, ErrInvalidInput
	}
	if businessType == "" {
		businessType = "service"
	}
	var t Tenant
	err := scanTenant(s.pool.QueryRow(ctx,
		`INSERT INTO tenants (name, tax_id, business_type, address) VALUES ($1, $2, $3, $4)
		 RETURNING `+tenantCols,
		name, taxID, businessType, address).Scan, &t)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Tenant{}, ErrDuplicateTaxID
		}
		return Tenant{}, err
	}
	return t, nil
}

func (s *Store) UpdateTenant(ctx context.Context, id, name, address, status, businessType string) (Tenant, error) {
	var t Tenant
	err := scanTenant(s.pool.QueryRow(ctx,
		`UPDATE tenants SET
			name          = COALESCE(NULLIF($2,''), name),
			address       = COALESCE(NULLIF($3,''), address),
			status        = COALESCE(NULLIF($4,''), status),
			business_type = COALESCE(NULLIF($5,''), business_type),
			updated_at    = NOW()
		 WHERE id = $1
		 RETURNING `+tenantCols,
		id, name, address, status, businessType).Scan, &t)
	if errors.Is(err, pgx.ErrNoRows) {
		return Tenant{}, ErrNotFound
	}
	return t, err
}

func (s *Store) ListBranchesByTenant(ctx context.Context, tenantID string) ([]Branch, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+branchCols+` FROM branches WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Branch
	for rows.Next() {
		var b Branch
		if err := scanBranch(rows.Scan, &b); err != nil {
			return nil, err
		}
		items = append(items, b)
	}
	return items, nil
}

func (s *Store) CreateBranch(ctx context.Context, tenantID, name, code, address, phone string) (Branch, error) {
	if tenantID == "" || name == "" {
		return Branch{}, ErrInvalidInput
	}
	var b Branch
	err := scanBranch(s.pool.QueryRow(ctx,
		`INSERT INTO branches (tenant_id, name, code, address, phone) VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+branchCols,
		tenantID, name, nullIfEmpty(code), nullIfEmpty(address), nullIfEmpty(phone)).Scan, &b)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return Branch{}, ErrInvalidTenant
		}
		return Branch{}, err
	}
	return b, nil
}

func (s *Store) UpdateBranch(ctx context.Context, tenantID, branchID, name, address, phone, status string) (Branch, error) {
	var b Branch
	err := scanBranch(s.pool.QueryRow(ctx,
		`UPDATE branches SET
			name    = COALESCE(NULLIF($3,''), name),
			address = COALESCE(NULLIF($4,''), address),
			phone   = COALESCE(NULLIF($5,''), phone),
			status  = COALESCE(NULLIF($6,''), status),
			updated_at = NOW()
		 WHERE id = $2 AND tenant_id = $1
		 RETURNING `+branchCols,
		tenantID, branchID, name, address, phone, status).Scan, &b)
	if errors.Is(err, pgx.ErrNoRows) {
		return Branch{}, ErrNotFound
	}
	return b, err
}

func (s *Store) ListUsers(ctx context.Context, tenantID string) ([]User, error) {
	query := `SELECT id, tenant_id, name, email, phone, line_user_id, role, status, created_at, updated_at FROM users`
	args := []any{}
	if tenantID != "" {
		query += " WHERE tenant_id = $1"
		args = append(args, tenantID)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.TenantID, &u.Name, &u.Email, &u.Phone, &u.LineUserID, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, u)
	}
	return items, nil
}

func (s *Store) HasAnyUser(ctx context.Context) (bool, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count > 0, err
}

func (s *Store) GetUserByID(ctx context.Context, id string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, email, phone, line_user_id, role, status, password_hash, created_at, updated_at
		 FROM users WHERE id = $1 AND status = 'active'`, id).
		Scan(&u.ID, &u.TenantID, &u.Name, &u.Email, &u.Phone, &u.LineUserID, &u.Role, &u.Status, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, email, phone, line_user_id, role, status, password_hash, created_at, updated_at
		 FROM users WHERE email = $1 AND status = 'active'`, email).
		Scan(&u.ID, &u.TenantID, &u.Name, &u.Email, &u.Phone, &u.LineUserID, &u.Role, &u.Status, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (s *Store) CreateUser(ctx context.Context, input User) (User, error) {
	if input.TenantID == "" || input.Name == "" || input.Role == "" {
		return User{}, ErrInvalidInput
	}
	var u User
	err := s.pool.QueryRow(ctx,
		`INSERT INTO users (tenant_id, name, email, phone, line_user_id, role, password_hash)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, tenant_id, name, email, phone, line_user_id, role, status, created_at, updated_at`,
		input.TenantID, input.Name, input.Email, input.Phone, input.LineUserID, input.Role, input.PasswordHash).
		Scan(&u.ID, &u.TenantID, &u.Name, &u.Email, &u.Phone, &u.LineUserID, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23503" {
				return User{}, ErrInvalidTenant
			}
		}
		return User{}, err
	}
	return u, nil
}

func (s *Store) UpdateUser(ctx context.Context, id string, input User) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`UPDATE users SET
			name        = COALESCE(NULLIF($2,''), name),
			email       = COALESCE(NULLIF($3,''), email),
			phone       = COALESCE(NULLIF($4,''), phone),
			line_user_id= COALESCE(NULLIF($5,''), line_user_id),
			role        = COALESCE(NULLIF($6,''), role),
			status      = COALESCE(NULLIF($7,''), status),
			updated_at  = NOW()
		 WHERE id = $1
		 RETURNING id, tenant_id, name, email, phone, line_user_id, role, status, created_at, updated_at`,
		id, input.Name, input.Email, input.Phone, input.LineUserID, input.Role, input.Status).
		Scan(&u.ID, &u.TenantID, &u.Name, &u.Email, &u.Phone, &u.LineUserID, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (s *Store) DeleteUser(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CreateDocumentImport(ctx context.Context, input DocumentImport) (DocumentImport, error) {
	if input.TenantID == "" || input.BranchID == "" || input.UserID == "" || input.SourceType == "" {
		return DocumentImport{}, ErrInvalidInput
	}
	var d DocumentImport
	err := s.pool.QueryRow(ctx,
		`INSERT INTO document_imports (tenant_id, branch_id, user_id, source_type, source_url, total_files)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, tenant_id, branch_id, user_id, source_type, COALESCE(source_url,''), total_files, processed_files, status, created_at, updated_at`,
		input.TenantID, input.BranchID, input.UserID, input.SourceType, nullIfEmpty(input.SourceURL), input.TotalFiles).
		Scan(&d.ID, &d.TenantID, &d.BranchID, &d.UserID, &d.SourceType, &d.SourceURL, &d.TotalFiles, &d.ProcessedFiles, &d.Status, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			msg := pgErr.Message
			if strings.Contains(msg, "branch") {
				return DocumentImport{}, ErrInvalidBranch
			}
			if strings.Contains(msg, "user") {
				return DocumentImport{}, ErrInvalidUser
			}
			return DocumentImport{}, ErrInvalidTenant
		}
		return DocumentImport{}, err
	}
	return d, nil
}

func (s *Store) GetDocumentImport(ctx context.Context, id string) (DocumentImport, error) {
	var d DocumentImport
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, branch_id, user_id, source_type, COALESCE(source_url,''), total_files, processed_files, status, created_at, updated_at
		 FROM document_imports WHERE id = $1`, id).
		Scan(&d.ID, &d.TenantID, &d.BranchID, &d.UserID, &d.SourceType, &d.SourceURL, &d.TotalFiles, &d.ProcessedFiles, &d.Status, &d.CreatedAt, &d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return DocumentImport{}, ErrNotFound
	}
	return d, err
}

func (s *Store) UpdateDocumentImportStatus(ctx context.Context, id, status string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE document_imports SET status = $2, updated_at = NOW() WHERE id = $1`,
		id, status)
	return err
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (s *Store) GetUserByLineID(ctx context.Context, lineUserID, tenantID string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, email, phone, line_user_id, role, status, created_at, updated_at
		 FROM users WHERE line_user_id = $1 AND tenant_id = $2 AND status = 'active'`,
		lineUserID, tenantID).
		Scan(&u.ID, &u.TenantID, &u.Name, &u.Email, &u.Phone, &u.LineUserID, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (s *Store) GetOrCreateLiffUser(ctx context.Context, lineUserID, name, tenantID string) (User, error) {
	u, err := s.GetUserByLineID(ctx, lineUserID, tenantID)
	if err == nil {
		return u, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return User{}, err
	}
	return s.CreateUser(ctx, User{
		TenantID:   tenantID,
		Name:       name,
		LineUserID: lineUserID,
		Role:       "staff",
	})
}

func (s *Store) ListDocumentImportsByUser(ctx context.Context, userID string) ([]DocumentImport, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, tenant_id, branch_id, user_id, source_type, COALESCE(source_url,''),
		        total_files, processed_files, status, created_at, updated_at
		 FROM document_imports WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DocumentImport
	for rows.Next() {
		var d DocumentImport
		if err := rows.Scan(&d.ID, &d.TenantID, &d.BranchID, &d.UserID, &d.SourceType,
			&d.SourceURL, &d.TotalFiles, &d.ProcessedFiles, &d.Status, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, d)
	}
	return items, nil
}
