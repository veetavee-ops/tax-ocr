package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"

	"tax-ocr/backend/internal/db"
	"tax-ocr/backend/internal/queue"
)

func (s *server) listInvoices(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	status := r.URL.Query().Get("status")
	items, err := s.store.ListInvoices(r.Context(), tenantID, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *server) getInvoice(w http.ResponseWriter, r *http.Request) {
	inv, err := s.store.GetInvoice(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": inv})
}

func (s *server) getInvoiceImage(w http.ResponseWriter, r *http.Request) {
	inv, err := s.store.GetInvoice(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if inv.FilePath == "" {
		writeError(w, http.StatusNotFound, errors.New("no file associated with this invoice"))
		return
	}
	data, err := s.storage.Download(r.Context(), inv.FilePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	ct := detectContentType(inv.FilePath)
	w.Header().Set("Content-Type", ct)
	w.Header().Set("X-File-Path", inv.FilePath)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	_, _ = w.Write(data)
}

func (s *server) getInvoiceItems(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListInvoiceItems(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *server) updateInvoice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		VendorName     string  `json:"vendor_name"`
		VendorTaxID    string  `json:"vendor_tax_id"`
		InvoiceDocNo   string  `json:"invoice_doc_no"`
		InvoiceDate    string  `json:"invoice_date"`
		TotalBeforeVat float64 `json:"total_before_vat"`
		VatAmount      float64 `json:"vat_amount"`
		TotalAmount    float64 `json:"total_amount"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	log.Printf("[updateInvoice] id=%s vat=%.2f total=%.2f before=%.2f", id, req.VatAmount, req.TotalAmount, req.TotalBeforeVat)

	// Fetch current to compute vat_math_ok with merged amounts.
	cur, err := s.store.GetInvoice(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	before := cur.TotalBeforeVat
	vat := cur.VatAmount
	if req.TotalBeforeVat != 0 {
		before = req.TotalBeforeVat
	}
	if req.VatAmount != 0 {
		vat = req.VatAmount
	}
	vatMathOK := math.Abs(before*0.07-vat) < 0.01

	if err := s.store.UpdateInvoiceData(r.Context(), id, db.InvoiceUpdate{
		VendorName:     req.VendorName,
		VendorTaxID:    req.VendorTaxID,
		InvoiceDocNo:   req.InvoiceDocNo,
		InvoiceDate:    req.InvoiceDate,
		TotalBeforeVAT: req.TotalBeforeVat,
		VATAmount:      req.VatAmount,
		TotalAmount:    req.TotalAmount,
		VATMathOK:      vatMathOK,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	inv, err := s.store.GetInvoice(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": inv})
}

func (s *server) verifyInvoice(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	id := r.PathValue("id")

	// Accept optional corrected values from the wizard.
	// Use a lenient decoder so missing body is not an error.
	var corrections struct {
		TotalBeforeVat float64 `json:"total_before_vat"`
		VatAmount      float64 `json:"vat_amount"`
		TotalAmount    float64 `json:"total_amount"`
	}
	_ = func() error {
		defer r.Body.Close()
		return json.NewDecoder(r.Body).Decode(&corrections)
	}()

	// Apply confirmed values before marking verified (atomic: confirm + verify in one step).
	cur, _ := s.store.GetInvoice(r.Context(), id)
	beforeVAT := cur.TotalBeforeVat
	if corrections.TotalBeforeVat != 0 {
		beforeVAT = corrections.TotalBeforeVat
	}
	vatAmt := cur.VatAmount
	if corrections.VatAmount != 0 {
		vatAmt = corrections.VatAmount
	}
	totalAmt := cur.TotalAmount
	if corrections.TotalAmount != 0 {
		totalAmt = corrections.TotalAmount
	}
	vatMathOK := math.Abs(beforeVAT*0.07-vatAmt) < 0.02
	log.Printf("[verifyInvoice] confirming id=%s before_vat=%.2f vat=%.2f total=%.2f vatMathOK=%v", id, beforeVAT, vatAmt, totalAmt, vatMathOK)
	if err := s.store.UpdateInvoiceAmounts(r.Context(), id, beforeVAT, vatAmt, totalAmt, vatMathOK); err != nil {
		log.Printf("[verifyInvoice] UpdateInvoiceAmounts error: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	inv, err := s.store.VerifyInvoice(r.Context(), id, claims.UserID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, db.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	log.Printf("[verifyInvoice] done id=%s vat=%.2f total=%.2f", id, inv.VatAmount, inv.TotalAmount)
	writeJSON(w, http.StatusOK, map[string]any{"data": inv})
}

func (s *server) updateInvoiceItem(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Quantity   float64 `json:"quantity"`
		UnitPrice  float64 `json:"unit_price"`
		TotalPrice float64 `json:"total_price"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.store.UpdateInvoiceItem(r.Context(), r.PathValue("id"), req.Quantity, req.UnitPrice, req.TotalPrice); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) reprocessInvoice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	inv, err := s.store.GetInvoice(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if s.queue == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("queue ไม่พร้อมใช้งาน"))
		return
	}

	_ = s.store.UpdateInvoiceData(r.Context(), id, db.InvoiceUpdate{Status: "pending"})

	if err := s.queue.EnqueueProcessInvoice(r.Context(), queue.ProcessInvoicePayload{
		InvoiceID:        inv.ID,
		DocumentImportID: inv.DocumentImportID,
		TenantID:         inv.TenantID,
		BranchID:         inv.BranchID,
		FilePath:         inv.FilePath,
		ContentType:      detectContentType(inv.FilePath),
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{"message": "OCR job queued"})
}

func (s *server) deleteInvoice(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteInvoice(r.Context(), r.PathValue("id")); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, db.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) createInvoice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID         string  `json:"tenant_id"`
		BranchID         string  `json:"branch_id"`
		DocumentImportID string  `json:"document_import_id"`
		FilePath         string  `json:"file_path"`
		FileHash         string  `json:"file_hash"`
		VendorTaxID      string  `json:"vendor_tax_id"`
		TotalBeforeVat   float64 `json:"total_before_vat"`
		VatAmount        float64 `json:"vat_amount"`
		TotalAmount      float64 `json:"total_amount"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	inv, err := s.store.CreateInvoice(r.Context(), db.Invoice{
		TenantID:         req.TenantID,
		BranchID:         req.BranchID,
		DocumentImportID: req.DocumentImportID,
		FilePath:         req.FilePath,
		FileHash:         req.FileHash,
		VendorTaxID:      req.VendorTaxID,
		TotalBeforeVat:   req.TotalBeforeVat,
		VatAmount:        req.VatAmount,
		TotalAmount:      req.TotalAmount,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": inv})
}
