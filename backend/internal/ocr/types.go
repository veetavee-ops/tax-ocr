package ocr

import "time"

type InvoiceData struct {
	// Document classification (Phase 1)
	DocType      string `json:"doc_type"`      // tax_invoice / receipt / delivery_note / unknown
	VatInclusive bool   `json:"vat_inclusive"` // true = line item prices already include VAT
	VatRate      float64 `json:"vat_rate"`     // typically 7.00 or 0.00
	// Seller info
	VendorName       string `json:"vendor_name"`
	VendorTaxID      string `json:"vendor_tax_id"`
	VendorAddress    string `json:"vendor_address"`
	VendorBranchCode string `json:"vendor_branch_code"`
	// Buyer info
	BuyerName       string `json:"buyer_name"`
	BuyerTaxID      string `json:"buyer_tax_id"`
	BuyerAddress    string `json:"buyer_address"`
	BuyerBranchCode string `json:"buyer_branch_code"`
	// Document reference
	InvoiceDocNo string `json:"invoice_doc_no"`
	InvoiceDate  string `json:"invoice_date"`
	// Financial summary
	VatExemptAmount      float64    `json:"vat_exempt_amount"`
	VatInclusiveSubtotal float64    `json:"vat_inclusive_subtotal"`
	DiscountAmount       float64    `json:"discount_amount"`
	TotalBeforeVAT       float64    `json:"total_before_vat"`
	VATAmount            float64    `json:"vat_amount"`
	TotalAmount          float64    `json:"total_amount"`
	Items                []LineItem `json:"items"`
}

type LineItem struct {
	ProductCode string  `json:"product_code"`
	Description string  `json:"description"`
	Unit        string  `json:"unit"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Discount    float64 `json:"discount"`
	TotalPrice  float64 `json:"total_price"`
}

type ExtractionRequest struct {
	TenantID    string
	FilePath    string
	FileBytes   []byte
	ContentType string
}

type ExtractionResult struct {
	Data        InvoiceData
	Matched     bool
	VATMathOK   bool    // stated VAT ≈ total_before_vat × 7%
	RawText     string
	Confidence  float64
	Engine      string
	ProcessedAt time.Time
}

// DebugResult exposes both engines' outputs side-by-side for the admin test panel.
type DebugResult struct {
	GPT     InvoiceData `json:"gpt"`
	Vision  InvoiceData `json:"vision"`
	Matched bool        `json:"matched"`
	RawText string      `json:"raw_text"`
	Engine  string      `json:"engine"`
}
