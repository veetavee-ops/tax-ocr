package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound            = errors.New("record not found")
	ErrInvalidTenant       = errors.New("tenant not found")
	ErrInvalidBranch       = errors.New("branch not found")
	ErrInvalidUser         = errors.New("user not found")
	ErrInvalidInvoice      = errors.New("invoice not found")
	ErrInvalidReviewer     = errors.New("reviewer not found")
	ErrInvalidInput        = errors.New("invalid input")
	ErrDuplicateTaxID      = errors.New("duplicate tax_id")
	ErrDuplicateKeyword    = errors.New("duplicate keyword for tenant")
	ErrDuplicateLineUserID = errors.New("duplicate line_user_id")
	ErrDuplicateInvoice    = errors.New("duplicate invoice: same vendor and invoice number already exists")
)

type Tenant struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	TaxID        string    `json:"tax_id"`
	Address      string    `json:"address,omitempty"`
	Status       string    `json:"status"`
	BusinessType string    `json:"business_type"` // trading / service / construction
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Branch struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	Address   string    `json:"address,omitempty"`
	Phone     string    `json:"phone,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type User struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	LineUserID   string    `json:"line_user_id"`
	Role         string    `json:"role"`
	Status       string    `json:"status"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type DocumentImport struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	BranchID       string    `json:"branch_id"`
	UserID         string    `json:"user_id"`
	SourceType     string    `json:"source_type"`
	SourceURL      string    `json:"source_url,omitempty"`
	TotalFiles     int       `json:"total_files"`
	ProcessedFiles int       `json:"processed_files"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Invoice struct {
	ID               string     `json:"id"`
	InvoiceNo        int        `json:"invoice_no"`
	TenantID         string     `json:"tenant_id"`
	BranchID         string     `json:"branch_id"`
	DocumentImportID string     `json:"document_import_id,omitempty"`
	FilePath         string     `json:"file_path"`
	FileHash         string     `json:"file_hash"`
	// Document classification
	DocType      string `json:"doc_type,omitempty"`
	VatInclusive bool   `json:"vat_inclusive"`
	VatRate      float64 `json:"vat_rate"`
	// Seller info
	VendorName        string `json:"vendor_name,omitempty"`
	VendorTaxID       string `json:"vendor_tax_id,omitempty"`
	VendorAddress     string `json:"vendor_address,omitempty"`
	VendorBranchCode  string `json:"vendor_branch_code,omitempty"`
	// Buyer info
	BuyerName        string `json:"buyer_name,omitempty"`
	BuyerTaxID       string `json:"buyer_tax_id,omitempty"`
	BuyerAddress     string `json:"buyer_address,omitempty"`
	BuyerBranchCode  string `json:"buyer_branch_code,omitempty"`
	// Document reference
	InvoiceDocNo  string `json:"invoice_doc_no,omitempty"`
	InvoiceDate   string `json:"invoice_date,omitempty"`
	// Parsed date parts from the document itself (CE year)
	InvoiceYear  int `json:"invoice_year,omitempty"`
	InvoiceMonth int `json:"invoice_month,omitempty"`
	InvoiceDay   int `json:"invoice_day,omitempty"`
	// Accounting period: which VAT return month this document is claimed in
	AccountingYear  int `json:"accounting_year,omitempty"`
	AccountingMonth int `json:"accounting_month,omitempty"`
	// Vendor registry link (set after vendor lookup in worker)
	VendorID string `json:"vendor_id,omitempty"`
	// Duplicate detection
	DuplicateOf string `json:"duplicate_of,omitempty"`
	// Financial summary
	VatExemptAmount      float64 `json:"vat_exempt_amount"`
	VatInclusiveSubtotal float64 `json:"vat_inclusive_subtotal"`
	DiscountAmount       float64 `json:"discount_amount"`
	TotalBeforeVat       float64 `json:"total_before_vat"`
	VatAmount            float64 `json:"vat_amount"`
	TotalAmount          float64 `json:"total_amount"`
	VatMathOK            bool    `json:"vat_math_ok"`
	// Status
	Status        string     `json:"status"`
	InvalidReason string     `json:"invalid_reason,omitempty"`
	VerifiedBy    string     `json:"verified_by,omitempty"`
	VerifiedAt    *time.Time `json:"verified_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type Vendor struct {
	ID         string     `json:"id"`
	TaxID      string     `json:"tax_id"`
	Name       string     `json:"name,omitempty"`
	Address    string     `json:"address,omitempty"`
	BranchCode string     `json:"branch_code,omitempty"`
	BranchName string     `json:"branch_name,omitempty"`
	Phone      string     `json:"phone,omitempty"`
	Verified   bool       `json:"verified"`
	VerifiedBy string     `json:"verified_by,omitempty"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type InvoiceItem struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	BranchID    string    `json:"branch_id"`
	InvoiceID   string    `json:"invoice_id"`
	ProductCode string    `json:"product_code,omitempty"`
	Description string    `json:"description"`
	Unit        string    `json:"unit,omitempty"`
	Quantity    float64   `json:"quantity"`
	UnitPrice   float64   `json:"unit_price"`
	Discount    float64   `json:"discount"`
	TotalPrice  float64   `json:"total_price"`
	AssetType    string    `json:"asset_type"`
	ClassifiedBy string    `json:"classified_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ClassificationRule struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	Keyword    string    `json:"keyword"`
	AssetType  string    `json:"asset_type"`
	Source     string    `json:"source"`
	Confidence float64   `json:"confidence"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type HitlQueueItem struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	InvoiceItemID string    `json:"invoice_item_id"`
	Reason        string    `json:"reason"`
	Status        string    `json:"status"`
	ResolvedBy    string    `json:"resolved_by,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type AuditLog struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	BranchID   string    `json:"branch_id,omitempty"`
	UserID     string    `json:"user_id,omitempty"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type,omitempty"`
	EntityID   string    `json:"entity_id,omitempty"`
	Metadata   any       `json:"metadata,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
	DeviceInfo string    `json:"device_info,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Reviewer struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	LineUserID    string    `json:"line_user_id"`
	ReviewerType  string    `json:"reviewer_type"`
	Status        string    `json:"status"`
	TotalEarned   float64   `json:"total_earned"`
	PendingPayout float64   `json:"pending_payout"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ReviewerTask struct {
	ID           string     `json:"id"`
	HitlQueueID  string     `json:"hitl_queue_id"`
	ReviewerID   string     `json:"reviewer_id"`
	TaskType     string     `json:"task_type"`
	Status       string     `json:"status"`
	RewardAmount float64    `json:"reward_amount"`
	SentAt       time.Time  `json:"sent_at"`
	AcceptedAt   *time.Time `json:"accepted_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	ExpiredAt    *time.Time `json:"expired_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type Conversation struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	BranchID   string    `json:"branch_id,omitempty"`
	UserID     string    `json:"user_id,omitempty"`
	Channel    string    `json:"channel"`
	LineUserID string    `json:"line_user_id,omitempty"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	TenantID       string    `json:"tenant_id"`
	SenderType     string    `json:"sender_type"`
	SenderID       string    `json:"sender_id,omitempty"`
	MessageType    string    `json:"message_type"`
	Content        string    `json:"content"`
	Metadata       any       `json:"metadata,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type TenantStorageConfig struct {
	ID                string    `json:"id"`
	TenantID          string    `json:"tenant_id"`
	StorageType       string    `json:"storage_type"`
	GdriveFolderID    string    `json:"gdrive_folder_id,omitempty"`
	GdriveFolderURL   string    `json:"gdrive_folder_url,omitempty"`
	OnedriveFolderID  string    `json:"onedrive_folder_id,omitempty"`
	OnedriveFolderURL string    `json:"onedrive_folder_url,omitempty"`
	OwnedBy           string    `json:"owned_by"`
	BillingType       string    `json:"billing_type"`
	MonthlyFee        float64   `json:"monthly_fee"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type ArchivePolicy struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	ActiveDays  int       `json:"active_days"`
	ArchiveDays int       `json:"archive_days"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ArchiveLog struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	EntityType  string    `json:"entity_type"`
	EntityID    string    `json:"entity_id"`
	ArchivedAt  time.Time `json:"archived_at"`
	ArchivePath string    `json:"archive_path"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type OcrConfig struct {
	ID        string    `json:"id"`
	Provider  string    `json:"provider"`
	APIKey    string    `json:"api_key,omitempty"`
	Enabled   bool      `json:"enabled"`
	UpdatedBy string    `json:"updated_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RewardConfig struct {
	ID        string    `json:"id"`
	TaskType  string    `json:"task_type"`
	Amount    float64   `json:"amount"`
	Currency  string    `json:"currency"`
	UpdatedBy string    `json:"updated_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Store wraps a PostgreSQL connection pool.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(ctx context.Context, connString string) (*Store, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}
