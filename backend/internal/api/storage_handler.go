package api

import (
	"errors"
	"net/http"

	"tax-ocr/backend/internal/db"
)

func (s *server) getStorageConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.store.GetStorageConfig(r.Context(), r.PathValue("tenantId"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": cfg})
}

func (s *server) createStorageConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID          string  `json:"tenant_id"`
		StorageType       string  `json:"storage_type"`
		GdriveFolderID    string  `json:"gdrive_folder_id"`
		GdriveFolderURL   string  `json:"gdrive_folder_url"`
		OnedriveFolderID  string  `json:"onedrive_folder_id"`
		OnedriveFolderURL string  `json:"onedrive_folder_url"`
		OwnedBy           string  `json:"owned_by"`
		BillingType       string  `json:"billing_type"`
		MonthlyFee        float64 `json:"monthly_fee"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	cfg, err := s.store.CreateStorageConfig(r.Context(), db.TenantStorageConfig{
		TenantID:          req.TenantID,
		StorageType:       req.StorageType,
		GdriveFolderID:    req.GdriveFolderID,
		GdriveFolderURL:   req.GdriveFolderURL,
		OnedriveFolderID:  req.OnedriveFolderID,
		OnedriveFolderURL: req.OnedriveFolderURL,
		OwnedBy:           req.OwnedBy,
		BillingType:       req.BillingType,
		MonthlyFee:        req.MonthlyFee,
	})
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, db.ErrInvalidTenant) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": cfg})
}

func (s *server) updateStorageConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StorageType       string  `json:"storage_type"`
		GdriveFolderID    string  `json:"gdrive_folder_id"`
		GdriveFolderURL   string  `json:"gdrive_folder_url"`
		OnedriveFolderID  string  `json:"onedrive_folder_id"`
		OnedriveFolderURL string  `json:"onedrive_folder_url"`
		OwnedBy           string  `json:"owned_by"`
		BillingType       string  `json:"billing_type"`
		MonthlyFee        float64 `json:"monthly_fee"`
		Status            string  `json:"status"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	cfg, err := s.store.UpdateStorageConfig(r.Context(), r.PathValue("tenantId"), db.TenantStorageConfig{
		StorageType:       req.StorageType,
		GdriveFolderID:    req.GdriveFolderID,
		GdriveFolderURL:   req.GdriveFolderURL,
		OnedriveFolderID:  req.OnedriveFolderID,
		OnedriveFolderURL: req.OnedriveFolderURL,
		OwnedBy:           req.OwnedBy,
		BillingType:       req.BillingType,
		MonthlyFee:        req.MonthlyFee,
		Status:            req.Status,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": cfg})
}
