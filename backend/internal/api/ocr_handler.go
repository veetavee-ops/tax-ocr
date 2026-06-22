package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"tax-ocr/backend/internal/ocr"
)

var errOCRNotConfigured = errors.New("OCR engine ยังไม่ได้ตั้งค่า API key")

// GET /api/v1/ocr/config
func (s *server) listOCRConfig(w http.ResponseWriter, r *http.Request) {
	configs, err := s.store.ListOCRConfigs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	type view struct {
		ID           string `json:"id"`
		Provider     string `json:"provider"`
		APIKeyMasked string `json:"api_key_masked"`
		Enabled      bool   `json:"enabled"`
	}
	views := make([]view, 0, len(configs))
	for _, c := range configs {
		views = append(views, view{
			ID:           c.ID,
			Provider:     c.Provider,
			APIKeyMasked: maskKey(c.APIKey),
			Enabled:      c.Enabled,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": views})
}

// PUT /api/v1/ocr/config/{provider}
func (s *server) updateOCRConfig(w http.ResponseWriter, r *http.Request) {
	provider := r.PathValue("provider")

	var req struct {
		APIKey  string `json:"api_key"`
		Enabled bool   `json:"enabled"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	userID := ""
	if c := claimsFromContext(r.Context()); c != nil {
		userID = c.UserID
	}

	saved, err := s.store.UpsertOCRConfig(r.Context(), provider, req.APIKey, req.Enabled, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Reload OCR service with latest config from DB
	reloadOCRService(r.Context(), s)

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id":             saved.ID,
			"provider":       saved.Provider,
			"api_key_masked": maskKey(saved.APIKey),
			"enabled":        saved.Enabled,
		},
	})
}

// POST /api/v1/ocr/test  (multipart: file)
func (s *server) testOCR(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	defer file.Close()

	if s.ocrSvc == nil || !s.ocrSvc.HasConfig() {
		writeError(w, http.StatusServiceUnavailable, errOCRNotConfigured)
		return
	}

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	contentType := detectContentType(header.Filename)

	result, err := s.ocrSvc.ExtractDebug(r.Context(), ocr.ExtractionRequest{
		FileBytes:   fileBytes,
		ContentType: contentType,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}

// POST /api/v1/ocr/extract-company
// Accepts a multipart file (image/PDF of company registration doc) and returns
// extracted tenant + branch data for auto-filling the create form.
func (s *server) extractCompanyInfo(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	defer file.Close()

	if s.ocrSvc == nil || !s.ocrSvc.HasConfig() {
		writeError(w, http.StatusServiceUnavailable, errOCRNotConfigured)
		return
	}

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	contentType := detectContentType(header.Filename)
	result, err := s.ocrSvc.ExtractCompanyInfo(r.Context(), fileBytes, contentType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}

// reloadOCRService reads latest keys from DB and updates the live OCR service.
func reloadOCRService(ctx context.Context, s *server) {
	if s.ocrSvc == nil {
		return
	}
	configs, err := s.store.ListOCRConfigs(ctx)
	if err != nil {
		return
	}
	cfg := ocr.Config{}
	for _, c := range configs {
		if !c.Enabled || c.APIKey == "" {
			continue
		}
		switch c.Provider {
		case "openai":
			cfg.OpenAIKey = c.APIKey
		case "gcv":
			cfg.GCVKey = c.APIKey
		}
	}
	s.ocrSvc.UpdateConfig(cfg)
}

func maskKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:7] + "..." + key[len(key)-4:]
}
