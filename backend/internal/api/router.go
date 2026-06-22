package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"tax-ocr/backend/internal/db"
	"tax-ocr/backend/internal/ocr"
	"tax-ocr/backend/internal/queue"
	"tax-ocr/backend/internal/storage"
)

var errMissingFields = errors.New("tenant_id, branch_id and user_id are required")

type ServerConfig struct {
	LineToken  string
	LineSecret string
}

func NewRouter(store *db.Store, storage *storage.Client, queueClient *queue.Client, ocrSvc *ocr.Service, cfg ServerConfig) http.Handler {
	api := &server{store: store, storage: storage, queue: queueClient, ocrSvc: ocrSvc, lineToken: cfg.LineToken, lineSecret: cfg.LineSecret}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /api/v1/status", statusHandler)
	mux.HandleFunc("POST /api/v1/auth/login", api.login)
	mux.HandleFunc("POST /api/v1/auth/setup", api.setup)
	mux.HandleFunc("POST /api/v1/auth/line", api.lineLogin)
	mux.HandleFunc("POST /api/v1/auth/refresh", api.refresh)
	mux.HandleFunc("POST /webhook/line/{tenantId}", api.lineWebhook)

	p := http.NewServeMux()
	p.HandleFunc("GET /api/v1/tenants", api.listTenants)
	p.HandleFunc("POST /api/v1/tenants", api.createTenant)
	p.HandleFunc("GET /api/v1/tenants/{id}", api.getTenant)
	p.HandleFunc("PUT /api/v1/tenants/{id}", api.updateTenant)

	p.HandleFunc("GET /api/v1/tenants/{id}/branches", api.listBranches)
	p.HandleFunc("POST /api/v1/tenants/{id}/branches", api.createBranch)
	p.HandleFunc("PUT /api/v1/tenants/{id}/branches/{branchId}", api.updateBranch)

	p.HandleFunc("GET /api/v1/users", api.listUsers)
	p.HandleFunc("POST /api/v1/users", api.createUser)
	p.HandleFunc("PUT /api/v1/users/{id}", api.updateUser)
	p.HandleFunc("DELETE /api/v1/users/{id}", api.deleteUser)

	p.HandleFunc("POST /api/v1/documents/upload", api.uploadDocument)
	p.HandleFunc("POST /api/v1/documents/zip", api.uploadZip)
	p.HandleFunc("POST /api/v1/documents/link", api.uploadFromLink)
	p.HandleFunc("GET /api/v1/documents/{id}/status", api.documentStatus)
	p.HandleFunc("GET /api/v1/my-documents", api.myDocuments)

	p.HandleFunc("GET /api/v1/invoices", api.listInvoices)
	p.HandleFunc("POST /api/v1/invoices", api.createInvoice)
	p.HandleFunc("GET /api/v1/invoices/{id}", api.getInvoice)
	p.HandleFunc("PUT /api/v1/invoices/{id}", api.updateInvoice)
	p.HandleFunc("DELETE /api/v1/invoices/{id}", api.deleteInvoice)
	p.HandleFunc("GET /api/v1/invoices/{id}/image", api.getInvoiceImage)
	p.HandleFunc("GET /api/v1/invoices/{id}/items", api.getInvoiceItems)
	p.HandleFunc("POST /api/v1/invoices/{id}/verify", api.verifyInvoice)
	p.HandleFunc("POST /api/v1/invoices/{id}/reprocess", api.reprocessInvoice)
	p.HandleFunc("PUT /api/v1/invoices/{id}/accounting-period", api.updateAccountingPeriod)
	p.HandleFunc("PUT /api/v1/invoice-items/{id}", api.updateInvoiceItem)

	p.HandleFunc("GET /api/v1/vendors", api.listVendors)
	p.HandleFunc("GET /api/v1/vendors/lookup", api.lookupVendorByTaxID)
	p.HandleFunc("GET /api/v1/vendors/{id}", api.getVendor)
	p.HandleFunc("POST /api/v1/vendors/{id}/verify", api.verifyVendor)

	p.HandleFunc("GET /api/v1/rules", api.listRules)
	p.HandleFunc("POST /api/v1/rules", api.createRule)
	p.HandleFunc("GET /api/v1/rules/{id}", api.getRule)
	p.HandleFunc("PUT /api/v1/rules/{id}", api.updateRule)
	p.HandleFunc("DELETE /api/v1/rules/{id}", api.deleteRule)
	p.HandleFunc("POST /api/v1/rules/import", api.importRules)
	p.HandleFunc("GET /api/v1/rules/export", api.exportRules)
	p.HandleFunc("POST /api/v1/rules/test", api.testRule)

	p.HandleFunc("GET /api/v1/hitl/queue", api.listHitlQueue)
	p.HandleFunc("POST /api/v1/hitl/{id}/resolve", api.resolveHitlItem)
	p.HandleFunc("POST /api/v1/hitl/{id}/reject", api.rejectHitlItem)

	p.HandleFunc("GET /api/v1/reviewers", api.listReviewers)
	p.HandleFunc("POST /api/v1/reviewers", api.createReviewer)
	p.HandleFunc("PUT /api/v1/reviewers/{id}", api.updateReviewer)
	p.HandleFunc("GET /api/v1/reviewers/{id}/tasks", api.listReviewerTasks)
	p.HandleFunc("GET /api/v1/reviewers/{id}/payouts", api.listReviewerPayouts)
	p.HandleFunc("POST /api/v1/reviewers/payout", api.createPayout)
	p.HandleFunc("POST /api/v1/reviewer-tasks/{id}/accept", api.acceptReviewerTask)
	p.HandleFunc("POST /api/v1/reviewer-tasks/{id}/complete", api.completeReviewerTask)

	p.HandleFunc("GET /api/v1/audit-logs", api.listAuditLogs)
	p.HandleFunc("GET /api/v1/audit-logs/{id}", api.getAuditLog)

	p.HandleFunc("GET /api/v1/conversations", api.listConversations)
	p.HandleFunc("POST /api/v1/conversations", api.createConversation)
	p.HandleFunc("GET /api/v1/conversations/{id}/messages", api.getConversationMessages)
	p.HandleFunc("POST /api/v1/conversations/{id}/messages", api.sendMessage)

	p.HandleFunc("GET /api/v1/storage/config/{tenantId}", api.getStorageConfig)
	p.HandleFunc("POST /api/v1/storage/config", api.createStorageConfig)
	p.HandleFunc("PUT /api/v1/storage/config/{tenantId}", api.updateStorageConfig)

	p.HandleFunc("GET /api/v1/archive", api.listArchiveLogs)
	p.HandleFunc("POST /api/v1/archive/{id}/restore", api.restoreArchive)
	p.HandleFunc("GET /api/v1/archive/policies", api.listArchivePolicies)
	p.HandleFunc("POST /api/v1/archive/policies", api.createArchivePolicy)
	p.HandleFunc("PUT /api/v1/archive/policies/{id}", api.updateArchivePolicy)

	p.HandleFunc("GET /api/v1/reward/config", api.listRewardConfig)
	p.HandleFunc("PUT /api/v1/reward/config/{id}", api.updateRewardConfig)

	p.HandleFunc("GET /api/v1/ocr/config", api.listOCRConfig)
	p.HandleFunc("PUT /api/v1/ocr/config/{provider}", api.updateOCRConfig)
	p.HandleFunc("POST /api/v1/ocr/test", api.testOCR)

	p.HandleFunc("GET /api/v1/auth/me", api.me)
	p.HandleFunc("POST /api/v1/auth/logout", api.logout)

	mux.Handle("/api/v1/", authMiddleware(auditMiddleware(store)(p)))
	return mux
}

type server struct {
	store      *db.Store
	storage    *storage.Client
	queue      *queue.Client
	ocrSvc     *ocr.Service
	lineToken  string
	lineSecret string
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func statusHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"service": "tax-ocr-backend",
		"version": "dev",
	})
}

func (s *server) listTenants(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListTenants(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *server) createTenant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		TaxID        string `json:"tax_id"`
		BusinessType string `json:"business_type"`
	}

	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	tenant, err := s.store.CreateTenant(r.Context(), req.Name, req.TaxID, req.BusinessType)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, db.ErrDuplicateTaxID) {
			status = http.StatusConflict
		}
		writeError(w, status, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": tenant})
}

func (s *server) getTenant(w http.ResponseWriter, r *http.Request) {
	tenant, err := s.store.GetTenant(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": tenant})
}

func (s *server) updateTenant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		Address      string `json:"address"`
		Status       string `json:"status"`
		BusinessType string `json:"business_type"`
	}

	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	tenant, err := s.store.UpdateTenant(r.Context(), r.PathValue("id"), req.Name, req.Address, req.Status, req.BusinessType)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": tenant})
}

func (s *server) listBranches(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListBranchesByTenant(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *server) createBranch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		Code    string `json:"code"`
		Address string `json:"address"`
		Phone   string `json:"phone"`
	}

	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	branch, err := s.store.CreateBranch(r.Context(), r.PathValue("id"), req.Name, req.Code, req.Address, req.Phone)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, db.ErrInvalidTenant) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": branch})
}

func (s *server) updateBranch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		Address string `json:"address"`
		Phone   string `json:"phone"`
		Status  string `json:"status"`
	}

	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	branch, err := s.store.UpdateBranch(r.Context(), r.PathValue("id"), r.PathValue("branchId"), req.Name, req.Address, req.Phone, req.Status)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": branch})
}

func (s *server) listUsers(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	items, err := s.store.ListUsers(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *server) createUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID   string `json:"tenant_id"`
		Name       string `json:"name"`
		Email      string `json:"email"`
		Phone      string `json:"phone"`
		LineUserID string `json:"line_user_id"`
		Role       string `json:"role"`
		Password   string `json:"password"`
	}

	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if req.Password == "" {
		req.Password = "changeme"
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	user, err := s.store.CreateUser(r.Context(), db.User{
		TenantID:     req.TenantID,
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		LineUserID:   req.LineUserID,
		Role:         req.Role,
		PasswordHash: string(hashed),
	})
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, db.ErrInvalidTenant) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": user})
}

func (s *server) updateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string `json:"name"`
		Email      string `json:"email"`
		Phone      string `json:"phone"`
		LineUserID string `json:"line_user_id"`
		Role       string `json:"role"`
		Status     string `json:"status"`
	}

	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	user, err := s.store.UpdateUser(r.Context(), r.PathValue("id"), db.User{
		Name:       req.Name,
		Email:      req.Email,
		Phone:      req.Phone,
		LineUserID: req.LineUserID,
		Role:       req.Role,
		Status:     req.Status,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": user})
}

func (s *server) deleteUser(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteUser(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func readJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{
		"error": err.Error(),
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
