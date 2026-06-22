package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/hibiken/asynq"

	"tax-ocr/backend/internal/classify"
	"tax-ocr/backend/internal/db"
	"tax-ocr/backend/internal/ocr"
	"tax-ocr/backend/internal/reviewer"
	"tax-ocr/backend/internal/storage"
)

type WorkerConfig struct {
	RedisAddr   string
	Concurrency int
}

type Worker struct {
	server      *asynq.Server
	ocrSvc      *ocr.Service
	classifySvc *classify.Service
	store       *db.Store
	storage     *storage.Client
	reviewerSvc *reviewer.Service
	lineClient  *reviewer.LineClient
}

func NewWorker(cfg WorkerConfig, ocrSvc *ocr.Service, classifySvc *classify.Service, store *db.Store, storageCli *storage.Client, reviewerSvc *reviewer.Service, lineClient *reviewer.LineClient) *Worker {
	concurrency := cfg.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.RedisAddr},
		asynq.Config{
			Concurrency: concurrency,
			Queues:      map[string]int{"default": 10},
		},
	)

	return &Worker{
		server:      srv,
		ocrSvc:      ocrSvc,
		classifySvc: classifySvc,
		store:       store,
		storage:     storageCli,
		reviewerSvc: reviewerSvc,
		lineClient:  lineClient,
	}
}

func (w *Worker) Start() error {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeProcessInvoice, w.handleProcessInvoice)
	return w.server.Start(mux)
}

func (w *Worker) Shutdown() {
	w.server.Shutdown()
}

// parseInvoiceDate parses Thai/CE date strings from OCR output.
// Returns year (CE), month (1-12), day (1-31); returns 0,0,0 when unparseable.
// Buddhist Era (BE >= 2400) is converted to CE by subtracting 543.
func parseInvoiceDate(s string) (year, month, day int) {
	s = strings.TrimSpace(s)
	if s == "" {
		return
	}

	// DD/MM/YYYY  or  D/M/YYYY  (separator: / - .)
	re1 := regexp.MustCompile(`(\d{1,2})[/\-.](\d{1,2})[/\-.](\d{4})`)
	if m := re1.FindStringSubmatch(s); m != nil {
		day, _ = strconv.Atoi(m[1])
		month, _ = strconv.Atoi(m[2])
		year, _ = strconv.Atoi(m[3])
		if year >= 2400 {
			year -= 543
		}
		return
	}

	// YYYY-MM-DD  or  YYYY/MM/DD
	re2 := regexp.MustCompile(`(\d{4})[/\-.](\d{1,2})[/\-.](\d{1,2})`)
	if m := re2.FindStringSubmatch(s); m != nil {
		year, _ = strconv.Atoi(m[1])
		month, _ = strconv.Atoi(m[2])
		day, _ = strconv.Atoi(m[3])
		if year >= 2400 {
			year -= 543
		}
		return
	}

	// Thai long form: "31 ธันวาคม 2568" or "31 ธ.ค. 2568"
	thaiMonths := map[string]int{
		"มกราคม": 1, "กุมภาพันธ์": 2, "มีนาคม": 3, "เมษายน": 4,
		"พฤษภาคม": 5, "มิถุนายน": 6, "กรกฎาคม": 7, "สิงหาคม": 8,
		"กันยายน": 9, "ตุลาคม": 10, "พฤศจิกายน": 11, "ธันวาคม": 12,
		"ม.ค.": 1, "ก.พ.": 2, "มี.ค.": 3, "เม.ย.": 4,
		"พ.ค.": 5, "มิ.ย.": 6, "ก.ค.": 7, "ส.ค.": 8,
		"ก.ย.": 9, "ต.ค.": 10, "พ.ย.": 11, "ธ.ค.": 12,
	}
	re3 := regexp.MustCompile(`(\d{1,2})\s+(\S+)\s+(\d{4})`)
	if m := re3.FindStringSubmatch(s); m != nil {
		if mo, ok := thaiMonths[m[2]]; ok {
			day, _ = strconv.Atoi(m[1])
			month = mo
			year, _ = strconv.Atoi(m[3])
			if year >= 2400 {
				year -= 543
			}
			return
		}
	}
	return
}

// normalizeBranchCode converts Thai HQ synonyms to the standard "00000".
func normalizeBranchCode(code string) string {
	code = strings.TrimSpace(code)
	switch strings.ToUpper(code) {
	case "สำนักงานใหญ่", "สนญ.", "สนญ", "HEAD OFFICE", "HQ", "HEADQUARTER", "00000", "0":
		return "00000"
	}
	// Zero-pad numeric codes to 5 digits
	if matched, _ := regexp.MatchString(`^\d{1,5}$`, code); matched {
		n, _ := strconv.Atoi(code)
		return fmt.Sprintf("%05d", n)
	}
	return code
}

// stringSimilarity returns a 0.0–1.0 ratio using Levenshtein distance on rune slices.
func stringSimilarity(a, b string) float64 {
	ra := []rune(strings.TrimSpace(a))
	rb := []rune(strings.TrimSpace(b))
	la, lb := utf8.RuneCountInString(string(ra)), utf8.RuneCountInString(string(rb))
	if la == 0 && lb == 0 {
		return 1.0
	}
	if la == 0 || lb == 0 {
		return 0.0
	}
	// Levenshtein DP
	prev := make([]int, lb+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr := make([]int, lb+1)
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = min3(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev = curr
	}
	dist := prev[lb]
	maxLen := la
	if lb > maxLen {
		maxLen = lb
	}
	return 1.0 - float64(dist)/float64(maxLen)
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// validateBuyer checks that the buyer info on the invoice matches this tenant/branch.
// Returns ("", "") when all checks pass; returns (status, reason) when invalid.
// Only validates tax_invoice doc_type — other doc types don't carry input VAT.
func (w *Worker) validateBuyer(docType, buyerTaxID, buyerBranchCode, buyerName string, tenant db.Tenant, branch db.Branch) (status, reason string) {
	if docType != "tax_invoice" && docType != "" {
		return "", ""
	}
	// Buyer tax ID must match tenant (exact — no fuzzy)
	if buyerTaxID != "" && buyerTaxID != tenant.TaxID {
		return "invalid", "buyer_tax_id_mismatch"
	}
	// Buyer branch code must match branch (after normalization)
	normBuyer := normalizeBranchCode(buyerBranchCode)
	normBranch := normalizeBranchCode(branch.Code)
	if normBuyer != "" && normBranch != "" && normBuyer != normBranch {
		return "invalid", "buyer_branch_code_mismatch"
	}
	// Buyer name fuzzy match ≥ 85%
	if buyerName != "" && tenant.Name != "" {
		if stringSimilarity(buyerName, tenant.Name) < 0.85 {
			return "invalid", "buyer_name_mismatch"
		}
	}
	return "", ""
}

func (w *Worker) handleProcessInvoice(ctx context.Context, t *asynq.Task) error {
	var p ProcessInvoicePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	log.Printf("[worker] processing invoice %s", p.InvoiceID)

	// Download file from MinIO
	fileBytes, err := w.storage.Download(ctx, p.FilePath)
	if err != nil {

		log.Printf("[worker] download error for invoice %s: %v", p.InvoiceID, err)
		_ = w.store.UpdateInvoiceData(ctx, p.InvoiceID, db.InvoiceUpdate{Status: "conflict"})
		return fmt.Errorf("download: %w", err)
	}

	// Dual-engine OCR
	ocrResult, err := w.ocrSvc.Extract(ctx, ocr.ExtractionRequest{
		TenantID:    p.TenantID,
		FilePath:    p.FilePath,
		FileBytes:   fileBytes,
		ContentType: p.ContentType,
	})
	if err != nil {
		log.Printf("[worker] ocr error for invoice %s: %v", p.InvoiceID, err)
		_ = w.store.UpdateInvoiceData(ctx, p.InvoiceID, db.InvoiceUpdate{Status: "conflict"})
		return fmt.Errorf("ocr: %w", err)
	}

	invoiceStatus := "verified"
	if !ocrResult.Matched {
		invoiceStatus = "conflict"
	}

	d := ocrResult.Data

	// Parse invoice date into accounting period fields (CE year)
	invYear, invMonth, invDay := parseInvoiceDate(d.InvoiceDate)
	if invYear > 0 {
		log.Printf("[worker] invoice %s date parsed: %04d-%02d-%02d", p.InvoiceID, invYear, invMonth, invDay)
	}

	// Buyer validation and late invoice check (tax_invoice only)
	invalidReason := ""
	tenant, tenantErr := w.store.GetTenant(ctx, p.TenantID)
	branch, branchErr := w.store.GetBranch(ctx, p.BranchID)
	if tenantErr == nil && branchErr == nil {
		if s, r := w.validateBuyer(d.DocType, d.BuyerTaxID, d.BuyerBranchCode, d.BuyerName, tenant, branch); s == "invalid" {
			invoiceStatus = "invalid"
			invalidReason = r
			log.Printf("[worker] invoice %s INVALID buyer: %s", p.InvoiceID, r)
		}
	}
	// Late invoice flag: invoice_date > 3 months → cannot claim input VAT on ภพ.30
	if invoiceStatus != "invalid" && invYear > 0 {
		invoiceDate := time.Date(invYear, time.Month(invMonth), invDay, 0, 0, 0, 0, time.UTC)
		if time.Since(invoiceDate) > 90*24*time.Hour {
			invalidReason = "late_invoice_vat_unclaimed"
			log.Printf("[worker] invoice %s late invoice warning: date=%04d-%02d-%02d", p.InvoiceID, invYear, invMonth, invDay)
		}
	}

	// Duplicate invoice detection: same vendor + same invoice_doc_no within tenant is forbidden
	duplicateOf := ""
	if d.VendorTaxID != "" && d.InvoiceDocNo != "" {
		existing, err := w.store.FindDuplicateInvoice(ctx, p.TenantID, d.VendorTaxID, d.InvoiceDocNo, p.InvoiceID)
		if err == nil {
			duplicateOf = existing.ID
			invoiceStatus = "conflict"
			log.Printf("[worker] invoice %s DUPLICATE of %s (vendor=%s doc_no=%s year=%04d month=%02d)",
				p.InvoiceID, existing.ID, d.VendorTaxID, d.InvoiceDocNo, invYear, invMonth)
		}
	}

	// Vendor registry lookup: upsert unverified vendor from OCR data, then link to invoice.
	// If vendor already exists (verified or not), reuse existing record.
	if d.VendorTaxID != "" {
		vendor, err := w.store.UpsertVendorFromOCR(ctx, d.VendorTaxID, d.VendorName, d.VendorAddress, d.VendorBranchCode)
		if err == nil {
			_ = w.store.LinkInvoiceVendor(ctx, p.InvoiceID, vendor.ID)
			if vendor.Verified {
				log.Printf("[worker] invoice %s vendor %s verified (%s)", p.InvoiceID, vendor.TaxID, vendor.Name)
			} else {
				log.Printf("[worker] invoice %s vendor %s UNVERIFIED — awaiting confirmation", p.InvoiceID, vendor.TaxID)
			}
		}
	}

	_ = w.store.UpdateInvoiceData(ctx, p.InvoiceID, db.InvoiceUpdate{
		DocType:              d.DocType,
		VatInclusive:         d.VatInclusive,
		VatRate:              d.VatRate,
		VendorName:           d.VendorName,
		VendorTaxID:          d.VendorTaxID,
		VendorAddress:        d.VendorAddress,
		VendorBranchCode:     d.VendorBranchCode,
		BuyerName:            d.BuyerName,
		BuyerTaxID:           d.BuyerTaxID,
		BuyerAddress:         d.BuyerAddress,
		BuyerBranchCode:      d.BuyerBranchCode,
		InvoiceDocNo:         d.InvoiceDocNo,
		InvoiceDate:          d.InvoiceDate,
		InvoiceYear:          invYear,
		InvoiceMonth:         invMonth,
		InvoiceDay:           invDay,
		DuplicateOf:          duplicateOf,
		InvalidReason:        invalidReason,
		VatExemptAmount:      d.VatExemptAmount,
		VatInclusiveSubtotal: d.VatInclusiveSubtotal,
		DiscountAmount:       d.DiscountAmount,
		TotalBeforeVAT:       d.TotalBeforeVAT,
		VATAmount:            d.VATAmount,
		TotalAmount:          d.TotalAmount,
		VATMathOK:            ocrResult.VATMathOK,
		Status:               invoiceStatus,
	})

	// Clear old items before saving new ones (handles re-run OCR)
	_ = w.store.DeleteInvoiceItemsByInvoiceID(ctx, p.InvoiceID)

	// Skip item classification for duplicate invoices — they are flagged for admin review
	if duplicateOf != "" {
		if p.DocumentImportID != "" {
			_ = w.store.UpdateDocumentImportStatus(ctx, p.DocumentImportID, "done")
		}
		go w.notifyLineUser(p, invoiceStatus, 0)
		log.Printf("[worker] invoice %s skipped classification (duplicate)", p.InvoiceID)
		return nil
	}

	// Classify and save each line item
	for _, item := range ocrResult.Data.Items {
		classResult, err := w.classifySvc.ClassifyWithDB(ctx, classify.ClassificationInput{
			TenantID:    p.TenantID,
			Description: item.Description,
			Amount:      item.TotalPrice,
		})
		if err != nil {
			classResult = classify.ClassificationResult{AssetType: "pending", ClassifiedBy: "rule", Confidence: 0.3}
		}

		saved, err := w.store.CreateInvoiceItem(ctx, db.InvoiceItem{
			TenantID:     p.TenantID,
			BranchID:     p.BranchID,
			InvoiceID:    p.InvoiceID,
			ProductCode:  item.ProductCode,
			Description:  item.Description,
			Unit:         item.Unit,
			Quantity:     item.Quantity,
			UnitPrice:    item.UnitPrice,
			Discount:     item.Discount,
			TotalPrice:   item.TotalPrice,
			AssetType:    classResult.AssetType,
			ClassifiedBy: classResult.ClassifiedBy,
		})
		if err != nil {
			log.Printf("[worker] create invoice item error: %v", err)
			continue
		}

		// Route to HITL when classification is uncertain or OCR didn't verify
		if classResult.AssetType == "pending" || !ocrResult.Matched {
			reason := "classification_needed"
			if !ocrResult.Matched {
				reason = "ocr_mismatch"
			}
			hitlItem, err := w.store.CreateHitlItem(ctx, db.HitlQueueItem{
				TenantID:      p.TenantID,
				InvoiceItemID: saved.ID,
				Reason:        reason,
			})
			if err == nil && w.reviewerSvc != nil {
				w.reviewerSvc.AssignForHitl(ctx, hitlItem)
			}
		}
	}

	// Mark document_import as done
	if p.DocumentImportID != "" {
		_ = w.store.UpdateDocumentImportStatus(ctx, p.DocumentImportID, "done")
	}

	go w.notifyLineUser(p, invoiceStatus, len(ocrResult.Data.Items))

	log.Printf("[worker] invoice %s done: engine=%s status=%s items=%d matched=%v", p.InvoiceID, ocrResult.Engine, invoiceStatus, len(ocrResult.Data.Items), ocrResult.Matched)
	return nil
}

func (w *Worker) notifyLineUser(p ProcessInvoicePayload, status string, itemCount int) {
	if w.lineClient == nil || p.DocumentImportID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	doc, err := w.store.GetDocumentImport(ctx, p.DocumentImportID)
	if err != nil || doc.UserID == "" {
		return
	}
	user, err := w.store.GetUserByID(ctx, doc.UserID)
	if err != nil || user.LineUserID == "" {
		return
	}
	inv, err := w.store.GetInvoice(ctx, p.InvoiceID)
	if err != nil {
		return
	}

	var msg string
	if status == "verified" {
		msg = fmt.Sprintf("✅ ใบกำกับภาษีประมวลผลเสร็จแล้ว (#%d)\n💰 ยอดรวม: %.2f บาท\nรายการสินค้า: %d รายการ", inv.InvoiceNo, inv.TotalAmount, itemCount)
	} else {
		msg = fmt.Sprintf("⚠️ ใบกำกับภาษี (#%d) ต้องการการตรวจสอบเพิ่มเติม\nทีมงานจะดำเนินการและแจ้งผลให้ท่านทราบโดยเร็ว", inv.InvoiceNo)
	}
	if err := w.lineClient.Push(user.LineUserID, msg); err != nil {
		log.Printf("[worker] line push to %s: %v", user.LineUserID, err)
	}
}
