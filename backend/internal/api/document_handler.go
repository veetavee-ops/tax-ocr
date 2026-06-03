package api

import (
	"net/http"
	"strings"

	"tax-ocr/backend/internal/db"
)

const maxUploadSize = 20 << 20 // 20 MB

func (s *server) uploadDocument(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	tenantID := r.FormValue("tenant_id")
	branchID := r.FormValue("branch_id")
	userID := r.FormValue("user_id")

	if tenantID == "" || branchID == "" || userID == "" {
		writeError(w, http.StatusBadRequest, errMissingFields)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	defer file.Close()

	contentType := detectContentType(header.Filename)

	uploaded, err := s.storage.Upload(r.Context(), tenantID, header.Filename, file, header.Size, contentType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	doc, err := s.store.CreateDocumentImport(r.Context(), db.DocumentImport{
		TenantID:   tenantID,
		BranchID:   branchID,
		UserID:     userID,
		SourceType: "upload",
		TotalFiles: 1,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	inv, err := s.store.CreateInvoice(r.Context(), db.Invoice{
		TenantID:         tenantID,
		BranchID:         branchID,
		DocumentImportID: doc.ID,
		FilePath:         uploaded.Path,
		FileHash:         uploaded.FileHash,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"document_import": doc,
		"invoice":         inv,
	})
}

func (s *server) documentStatus(w http.ResponseWriter, r *http.Request) {
	doc, err := s.store.GetDocumentImport(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": doc})
}

func detectContentType(filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".pdf"):
		return "application/pdf"
	case strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	default:
		return "application/octet-stream"
	}
}
