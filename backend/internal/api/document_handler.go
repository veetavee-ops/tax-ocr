package api

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"tax-ocr/backend/internal/db"
	"tax-ocr/backend/internal/queue"
)

var safeHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		return validateURL(req.URL.String())
	},
}

var privateRanges []*net.IPNet

func init() {
	for _, cidr := range []string{
		"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16",
		"127.0.0.0/8", "169.254.0.0/16", "100.64.0.0/10",
		"::1/128", "fc00::/7", "fe80::/10",
	} {
		_, network, _ := net.ParseCIDR(cidr)
		if network != nil {
			privateRanges = append(privateRanges, network)
		}
	}
}

func validateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("only http/https URLs are allowed")
	}
	addrs, err := net.LookupHost(u.Hostname())
	if err != nil {
		return fmt.Errorf("DNS lookup failed: %w", err)
	}
	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip == nil {
			continue
		}
		for _, network := range privateRanges {
			if network.Contains(ip) {
				return errors.New("URL resolves to a private or reserved IP address")
			}
		}
	}
	return nil
}

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

	// Enqueue background OCR processing
	if s.queue != nil {
		err := s.queue.EnqueueProcessInvoice(r.Context(), queue.ProcessInvoicePayload{
			InvoiceID:        inv.ID,
			DocumentImportID: doc.ID,
			TenantID:         tenantID,
			BranchID:         branchID,
			FilePath:         uploaded.Path,
			ContentType:      contentType,
		})
		if err != nil {
			log.Printf("[upload] enqueue failed for invoice %s: %v", inv.ID, err)
		}
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

func (s *server) myDocuments(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	items, err := s.store.ListDocumentImportsByUser(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if items == nil {
		items = []db.DocumentImport{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

// uploadZip processes a ZIP file: extracts each image/PDF and enqueues OCR for each.
func (s *server) uploadZip(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadSize * 10); err != nil {
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

	if !strings.HasSuffix(strings.ToLower(header.Filename), ".zip") {
		writeError(w, http.StatusBadRequest, errors.New("file must be a .zip"))
		return
	}

	// Read zip into memory (max 100 MB)
	zipData, err := io.ReadAll(io.LimitReader(file, 100<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid zip: %w", err))
		return
	}

	// Count processable files
	var processable []*zip.File
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		ct := detectContentType(f.Name)
		if ct == "application/octet-stream" {
			continue
		}
		processable = append(processable, f)
	}
	if len(processable) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("zip contains no supported image/PDF files"))
		return
	}

	doc, err := s.store.CreateDocumentImport(r.Context(), db.DocumentImport{
		TenantID:   tenantID,
		BranchID:   branchID,
		UserID:     userID,
		SourceType: "zip",
		TotalFiles: len(processable),
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var invoices []db.Invoice
	for _, zf := range processable {
		rc, err := zf.Open()
		if err != nil {
			log.Printf("[zip] open %s: %v", zf.Name, err)
			continue
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}

		ct := detectContentType(zf.Name)
		uploaded, err := s.storage.Upload(r.Context(), tenantID, zf.Name,
			bytes.NewReader(data), int64(len(data)), ct)
		if err != nil {
			log.Printf("[zip] upload %s: %v", zf.Name, err)
			continue
		}

		inv, err := s.store.CreateInvoice(r.Context(), db.Invoice{
			TenantID:         tenantID,
			BranchID:         branchID,
			DocumentImportID: doc.ID,
			FilePath:         uploaded.Path,
			FileHash:         uploaded.FileHash,
		})
		if err != nil {
			log.Printf("[zip] create invoice %s: %v", zf.Name, err)
			continue
		}
		invoices = append(invoices, inv)

		if s.queue != nil {
			_ = s.queue.EnqueueProcessInvoice(r.Context(), queue.ProcessInvoicePayload{
				InvoiceID:        inv.ID,
				DocumentImportID: doc.ID,
				TenantID:         tenantID,
				BranchID:         branchID,
				FilePath:         uploaded.Path,
				ContentType:      ct,
			})
		}
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"document_import": doc,
		"invoices":        invoices,
	})
}

var gdriveIDPattern = regexp.MustCompile(`(?:/d/|id=)([a-zA-Z0-9_-]{25,})`)

// uploadFromLink downloads a file from a public URL (Google Drive / direct link) and enqueues OCR.
func (s *server) uploadFromLink(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID string `json:"tenant_id"`
		BranchID string `json:"branch_id"`
		UserID   string `json:"user_id"`
		URL      string `json:"url"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.TenantID == "" || req.BranchID == "" || req.UserID == "" || req.URL == "" {
		writeError(w, http.StatusBadRequest, errors.New("tenant_id, branch_id, user_id and url are required"))
		return
	}

	downloadURL := resolveDownloadURL(req.URL)

	if err := validateURL(downloadURL); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("blocked URL: %w", err))
		return
	}
	httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, downloadURL, nil)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid URL: %w", err))
		return
	}
	httpResp, err := safeHTTPClient.Do(httpReq)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("download failed: %w", err))
		return
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		writeError(w, http.StatusBadRequest, fmt.Errorf("download returned HTTP %d", httpResp.StatusCode))
		return
	}

	data, err := io.ReadAll(io.LimitReader(httpResp.Body, maxUploadSize))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	ct := httpResp.Header.Get("Content-Type")
	if ct == "" {
		ct = "image/jpeg"
	}
	ct = strings.Split(ct, ";")[0]

	filename := extractFilenameFromURL(req.URL)
	uploaded, err := s.storage.Upload(r.Context(), req.TenantID, filename,
		bytes.NewReader(data), int64(len(data)), ct)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	doc, err := s.store.CreateDocumentImport(r.Context(), db.DocumentImport{
		TenantID:   req.TenantID,
		BranchID:   req.BranchID,
		UserID:     req.UserID,
		SourceType: "gdrive",
		SourceURL:  req.URL,
		TotalFiles: 1,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	inv, err := s.store.CreateInvoice(r.Context(), db.Invoice{
		TenantID:         req.TenantID,
		BranchID:         req.BranchID,
		DocumentImportID: doc.ID,
		FilePath:         uploaded.Path,
		FileHash:         uploaded.FileHash,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	if s.queue != nil {
		_ = s.queue.EnqueueProcessInvoice(r.Context(), queue.ProcessInvoicePayload{
			InvoiceID:        inv.ID,
			DocumentImportID: doc.ID,
			TenantID:         req.TenantID,
			BranchID:         req.BranchID,
			FilePath:         uploaded.Path,
			ContentType:      ct,
		})
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"document_import": doc,
		"invoice":         inv,
	})
}

// resolveDownloadURL converts a Google Drive share URL to a direct download URL.
func resolveDownloadURL(rawURL string) string {
	if m := gdriveIDPattern.FindStringSubmatch(rawURL); len(m) > 1 {
		return fmt.Sprintf("https://drive.google.com/uc?export=download&confirm=t&id=%s", m[1])
	}
	return rawURL
}

func extractFilenameFromURL(rawURL string) string {
	parts := strings.Split(rawURL, "/")
	name := parts[len(parts)-1]
	if idx := strings.Index(name, "?"); idx != -1 {
		name = name[:idx]
	}
	if name == "" {
		return "download.jpg"
	}
	return name
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
