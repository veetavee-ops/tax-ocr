package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"tax-ocr/backend/internal/db"
	"tax-ocr/backend/internal/queue"
	rev "tax-ocr/backend/internal/reviewer"
)

type lineWebhookEvent struct {
	Type    string `json:"type"`
	Message struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"message"`
	Source struct {
		UserID string `json:"userId"`
	} `json:"source"`
	ReplyToken string `json:"replyToken"`
}

type lineWebhookBody struct {
	Events []lineWebhookEvent `json:"events"`
}

func (s *server) lineWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenantId")
	if tenantID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	rawBody, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if s.lineSecret != "" {
		sig := r.Header.Get("X-Line-Signature")
		if !verifyLineSignature(rawBody, s.lineSecret, sig) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	var body lineWebhookBody
	if err := json.Unmarshal(rawBody, &body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	lc := rev.NewLineClient(s.lineToken)
	for _, event := range body.Events {
		ev := event
		go s.dispatchLineEvent(tenantID, ev, lc)
	}

	w.WriteHeader(http.StatusOK)
}

func (s *server) dispatchLineEvent(tenantID string, event lineWebhookEvent, lc *rev.LineClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch event.Type {
	case "follow":
		s.onLineFollow(ctx, tenantID, event.Source.UserID, event.ReplyToken, lc)
	case "message":
		switch event.Message.Type {
		case "image":
			s.onLineImage(ctx, tenantID, event.Source.UserID, event.Message.ID, event.ReplyToken, lc)
		case "text":
			s.onLineText(ctx, tenantID, event.Source.UserID, event.Message.Text)
		}
	}
}

func (s *server) onLineFollow(ctx context.Context, tenantID, lineUserID, replyToken string, lc *rev.LineClient) {
	_, _ = s.store.GetOrCreateLiffUser(ctx, lineUserID, "LINE User", tenantID)
	branchID := s.firstActiveBranch(ctx, tenantID)
	_, _ = s.store.FindOrCreateConversationByLineUser(ctx, tenantID, lineUserID, branchID)
	_ = lc.Reply(replyToken, "ยินดีต้อนรับ! 🎉\nส่งรูปใบกำกับภาษีได้เลยครับ ระบบจะประมวลผล OCR ให้อัตโนมัติ")
}

func (s *server) onLineImage(ctx context.Context, tenantID, lineUserID, messageID, replyToken string, lc *rev.LineClient) {
	data, contentType, err := lc.GetContent(messageID)
	if err != nil {
		log.Printf("[line] get content %s: %v", messageID, err)
		_ = lc.Reply(replyToken, "ขออภัย ไม่สามารถรับไฟล์ได้ กรุณาลองใหม่")
		return
	}

	user, _ := s.store.GetOrCreateLiffUser(ctx, lineUserID, "LINE User", tenantID)
	branchID := s.firstActiveBranch(ctx, tenantID)
	if branchID == "" {
		log.Printf("[line] no active branch for tenant %s", tenantID)
		return
	}

	ext := lineContentExt(contentType)
	filename := fmt.Sprintf("line_%s%s", messageID, ext)

	uploaded, err := s.storage.Upload(ctx, tenantID, filename,
		bytes.NewReader(data), int64(len(data)), contentType)
	if err != nil {
		log.Printf("[line] upload: %v", err)
		_ = lc.Reply(replyToken, "ขออภัย บันทึกไฟล์ไม่สำเร็จ กรุณาลองใหม่")
		return
	}

	doc, err := s.store.CreateDocumentImport(ctx, db.DocumentImport{
		TenantID: tenantID, BranchID: branchID, UserID: user.ID,
		SourceType: "upload", TotalFiles: 1,
	})
	if err != nil {
		log.Printf("[line] create doc import: %v", err)
		return
	}

	inv, err := s.store.CreateInvoice(ctx, db.Invoice{
		TenantID: tenantID, BranchID: branchID,
		DocumentImportID: doc.ID,
		FilePath:         uploaded.Path,
		FileHash:         uploaded.FileHash,
	})
	if err != nil {
		log.Printf("[line] create invoice: %v", err)
		return
	}

	if s.queue != nil {
		_ = s.queue.EnqueueProcessInvoice(ctx, queue.ProcessInvoicePayload{
			InvoiceID: inv.ID, DocumentImportID: doc.ID,
			TenantID: tenantID, BranchID: branchID,
			FilePath: uploaded.Path, ContentType: contentType,
		})
	}

	_ = lc.Reply(replyToken, fmt.Sprintf("✅ รับใบกำกับภาษีแล้ว (#%d)\nกำลังประมวลผล OCR กรุณารอสักครู่...", inv.InvoiceNo))
}

func (s *server) onLineText(ctx context.Context, tenantID, lineUserID, text string) {
	branchID := s.firstActiveBranch(ctx, tenantID)
	conv, err := s.store.FindOrCreateConversationByLineUser(ctx, tenantID, lineUserID, branchID)
	if err != nil {
		log.Printf("[line] find conversation: %v", err)
		return
	}
	userID := ""
	if u, err := s.store.GetUserByLineID(ctx, lineUserID, tenantID); err == nil {
		userID = u.ID
	}
	_, _ = s.store.CreateMessage(ctx, db.Message{
		ConversationID: conv.ID,
		SenderType:     "customer",
		SenderID:       userID,
		MessageType:    "text",
		Content:        text,
	})
}

func (s *server) firstActiveBranch(ctx context.Context, tenantID string) string {
	branches, err := s.store.ListBranchesByTenant(ctx, tenantID)
	if err != nil {
		return ""
	}
	for _, b := range branches {
		if b.Status == "active" {
			return b.ID
		}
	}
	return ""
}

func verifyLineSignature(body []byte, secret, sig string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sig))
}

func lineContentExt(ct string) string {
	switch strings.TrimSpace(strings.Split(ct, ";")[0]) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "application/pdf":
		return ".pdf"
	default:
		return ".jpg"
	}
}
