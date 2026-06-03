package api

import (
	"net/http"

	"tax-ocr/backend/internal/db"
)

func (s *server) listConversations(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	items, err := s.store.ListConversations(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *server) createConversation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID   string `json:"tenant_id"`
		BranchID   string `json:"branch_id"`
		UserID     string `json:"user_id"`
		Channel    string `json:"channel"`
		LineUserID string `json:"line_user_id"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	conv, err := s.store.CreateConversation(r.Context(), db.Conversation{
		TenantID:   req.TenantID,
		BranchID:   req.BranchID,
		UserID:     req.UserID,
		Channel:    req.Channel,
		LineUserID: req.LineUserID,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": conv})
}

func (s *server) getConversationMessages(w http.ResponseWriter, r *http.Request) {
	msgs, err := s.store.ListMessages(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": msgs})
}

func (s *server) sendMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SenderType  string `json:"sender_type"`
		SenderID    string `json:"sender_id"`
		MessageType string `json:"message_type"`
		Content     string `json:"content"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	msg, err := s.store.CreateMessage(r.Context(), db.Message{
		ConversationID: r.PathValue("id"),
		SenderType:     req.SenderType,
		SenderID:       req.SenderID,
		MessageType:    req.MessageType,
		Content:        req.Content,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": msg})
}
