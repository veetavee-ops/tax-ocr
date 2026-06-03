package api

import (
	"net/http"
)

func (s *server) listHitlQueue(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	status := r.URL.Query().Get("status")
	items, err := s.store.ListHitlQueue(r.Context(), tenantID, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *server) resolveHitlItem(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResolvedBy string `json:"resolved_by"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := s.store.ResolveHitlItem(r.Context(), r.PathValue("id"), req.ResolvedBy)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": item})
}

func (s *server) rejectHitlItem(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.RejectHitlItem(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": item})
}
