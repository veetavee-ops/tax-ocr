package api

import (
	"net/http"

	"tax-ocr/backend/internal/db"
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

func (s *server) getInvoiceItems(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListInvoiceItems(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
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
