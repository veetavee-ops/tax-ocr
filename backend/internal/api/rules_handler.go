package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"tax-ocr/backend/internal/db"
)

func (s *server) listRules(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	items, err := s.store.ListRules(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *server) getRule(w http.ResponseWriter, r *http.Request) {
	rule, err := s.store.GetRule(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": rule})
}

func (s *server) createRule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID   string  `json:"tenant_id"`
		Keyword    string  `json:"keyword"`
		AssetType  string  `json:"asset_type"`
		Source     string  `json:"source"`
		Confidence float64 `json:"confidence"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	rule, err := s.store.CreateRule(r.Context(), db.ClassificationRule{
		TenantID:   req.TenantID,
		Keyword:    req.Keyword,
		AssetType:  req.AssetType,
		Source:     req.Source,
		Confidence: req.Confidence,
	})
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, db.ErrDuplicateKeyword) {
			status = http.StatusConflict
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": rule})
}

func (s *server) updateRule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Keyword    string  `json:"keyword"`
		AssetType  string  `json:"asset_type"`
		Confidence float64 `json:"confidence"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	rule, err := s.store.UpdateRule(r.Context(), r.PathValue("id"), db.ClassificationRule{
		Keyword:    req.Keyword,
		AssetType:  req.AssetType,
		Confidence: req.Confidence,
	})
	if err != nil {
		status := http.StatusNotFound
		if errors.Is(err, db.ErrDuplicateKeyword) {
			status = http.StatusConflict
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": rule})
}

func (s *server) deleteRule(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteRule(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) importRules(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID string `json:"tenant_id"`
		Rules    []struct {
			Keyword   string  `json:"keyword"`
			AssetType string  `json:"asset_type"`
			Source    string  `json:"source"`
			Confidence float64 `json:"confidence"`
		} `json:"rules"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	created := make([]db.ClassificationRule, 0, len(req.Rules))
	for _, item := range req.Rules {
		rule, err := s.store.CreateRule(r.Context(), db.ClassificationRule{
			TenantID:   req.TenantID,
			Keyword:    item.Keyword,
			AssetType:  item.AssetType,
			Source:     item.Source,
			Confidence: item.Confidence,
		})
		if err != nil {
			continue // skip duplicates
		}
		created = append(created, rule)
	}
	writeJSON(w, http.StatusOK, map[string]any{"imported": len(created), "data": created})
}

func (s *server) exportRules(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	rules, _ := s.store.ListRules(r.Context(), tenantID)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=rules.json")
	_ = json.NewEncoder(w).Encode(rules)
}

func (s *server) testRule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID string `json:"tenant_id"`
		Keyword  string `json:"keyword"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	rule, matched := s.store.TestRule(r.Context(), req.TenantID, req.Keyword)
	writeJSON(w, http.StatusOK, map[string]any{
		"matched": matched,
		"rule":    rule,
	})
}
