package api

import (
	"net/http"
)

func (s *server) listAuditLogs(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	items, err := s.store.ListAuditLogs(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *server) getAuditLog(w http.ResponseWriter, r *http.Request) {
	log, err := s.store.GetAuditLog(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": log})
}
