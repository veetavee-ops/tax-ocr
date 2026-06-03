package api

import (
	"errors"
	"net/http"

	"tax-ocr/backend/internal/db"
)

func (s *server) listArchiveLogs(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	items, err := s.store.ListArchiveLogs(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *server) restoreArchive(w http.ResponseWriter, r *http.Request) {
	a, err := s.store.RestoreArchive(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": a})
}

func (s *server) listArchivePolicies(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	items, err := s.store.ListArchivePolicies(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *server) createArchivePolicy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID    string `json:"tenant_id"`
		ActiveDays  int    `json:"active_days"`
		ArchiveDays int    `json:"archive_days"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	p, err := s.store.CreateArchivePolicy(r.Context(), db.ArchivePolicy{
		TenantID:    req.TenantID,
		ActiveDays:  req.ActiveDays,
		ArchiveDays: req.ArchiveDays,
	})
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, db.ErrInvalidTenant) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": p})
}

func (s *server) updateArchivePolicy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ActiveDays  int `json:"active_days"`
		ArchiveDays int `json:"archive_days"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	p, err := s.store.UpdateArchivePolicy(r.Context(), r.PathValue("id"), db.ArchivePolicy{
		ActiveDays:  req.ActiveDays,
		ArchiveDays: req.ArchiveDays,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": p})
}
