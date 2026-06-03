package api

import (
	"errors"
	"net/http"

	"tax-ocr/backend/internal/db"
)

func (s *server) listReviewers(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListReviewers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *server) createReviewer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		LineUserID   string `json:"line_user_id"`
		ReviewerType string `json:"reviewer_type"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	reviewer, err := s.store.CreateReviewer(r.Context(), db.Reviewer{
		Name:         req.Name,
		LineUserID:   req.LineUserID,
		ReviewerType: req.ReviewerType,
	})
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, db.ErrDuplicateLineUserID) {
			status = http.StatusConflict
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": reviewer})
}

func (s *server) updateReviewer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		ReviewerType string `json:"reviewer_type"`
		Status       string `json:"status"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	reviewer, err := s.store.UpdateReviewer(r.Context(), r.PathValue("id"), db.Reviewer{
		Name:         req.Name,
		ReviewerType: req.ReviewerType,
		Status:       req.Status,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": reviewer})
}

func (s *server) listReviewerTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.store.ListReviewerTasks(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": tasks})
}

func (s *server) listReviewerPayouts(w http.ResponseWriter, r *http.Request) {
	// placeholder — payout history per reviewer
	writeJSON(w, http.StatusOK, map[string]any{"data": []any{}})
}

func (s *server) createPayout(w http.ResponseWriter, _ *http.Request) {
	// placeholder — trigger payout batch
	writeJSON(w, http.StatusAccepted, map[string]any{"message": "payout queued"})
}
