package api

import (
	"errors"
	"net/http"

	"tax-ocr/backend/internal/db"
)

func (s *server) listVendors(w http.ResponseWriter, r *http.Request) {
	var verified *bool
	if v := r.URL.Query().Get("verified"); v != "" {
		b := v == "true"
		verified = &b
	}
	items, err := s.store.ListVendors(r.Context(), verified)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *server) getVendor(w http.ResponseWriter, r *http.Request) {
	v, err := s.store.GetVendor(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": v})
}

func (s *server) lookupVendorByTaxID(w http.ResponseWriter, r *http.Request) {
	taxID := r.URL.Query().Get("tax_id")
	if taxID == "" {
		writeError(w, http.StatusBadRequest, errors.New("tax_id is required"))
		return
	}
	v, err := s.store.FindVendorByTaxID(r.Context(), taxID)
	if errors.Is(err, db.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"data": nil})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": v})
}

func (s *server) verifyVendor(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Name       string `json:"name"`
		Address    string `json:"address"`
		BranchCode string `json:"branch_code"`
		BranchName string `json:"branch_name"`
		Phone      string `json:"phone"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	userID := userIDFromContext(r.Context())
	v, err := s.store.VerifyVendor(r.Context(), id, userID, body.Name, body.Address, body.BranchCode, body.BranchName, body.Phone)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": v})
}
