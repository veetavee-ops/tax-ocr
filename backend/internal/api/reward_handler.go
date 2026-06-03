package api

import (
	"net/http"
)

func (s *server) listRewardConfig(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListRewardConfigs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *server) updateRewardConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Amount    float64 `json:"amount"`
		UpdatedBy string  `json:"updated_by"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	rc, err := s.store.UpdateRewardConfig(r.Context(), r.PathValue("id"), req.Amount, req.UpdatedBy)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": rc})
}
