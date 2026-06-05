package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"

	"tax-ocr/backend/internal/classify"
	"tax-ocr/backend/internal/db"
	"tax-ocr/backend/internal/ocr"
	"tax-ocr/backend/internal/reviewer"
	"tax-ocr/backend/internal/storage"
)

type WorkerConfig struct {
	RedisAddr   string
	Concurrency int
}

type Worker struct {
	server      *asynq.Server
	ocrSvc      *ocr.Service
	classifySvc *classify.Service
	store       *db.Store
	storage     *storage.Client
	reviewerSvc *reviewer.Service
	lineClient  *reviewer.LineClient
}

func NewWorker(cfg WorkerConfig, ocrSvc *ocr.Service, classifySvc *classify.Service, store *db.Store, storageCli *storage.Client, reviewerSvc *reviewer.Service, lineClient *reviewer.LineClient) *Worker {
	concurrency := cfg.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.RedisAddr},
		asynq.Config{
			Concurrency: concurrency,
			Queues:      map[string]int{"default": 10},
		},
	)

	return &Worker{
		server:      srv,
		ocrSvc:      ocrSvc,
		classifySvc: classifySvc,
		store:       store,
		storage:     storageCli,
		reviewerSvc: reviewerSvc,
		lineClient:  lineClient,
	}
}

func (w *Worker) Start() error {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeProcessInvoice, w.handleProcessInvoice)
	return w.server.Start(mux)
}

func (w *Worker) Shutdown() {
	w.server.Shutdown()
}

func (w *Worker) handleProcessInvoice(ctx context.Context, t *asynq.Task) error {
	var p ProcessInvoicePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	log.Printf("[worker] processing invoice %s", p.InvoiceID)

	// Download file from MinIO
	fileBytes, err := w.storage.Download(ctx, p.FilePath)
	if err != nil {
		log.Printf("[worker] download error for invoice %s: %v", p.InvoiceID, err)
		_ = w.store.UpdateInvoiceData(ctx, p.InvoiceID, db.InvoiceUpdate{Status: "conflict"})
		return fmt.Errorf("download: %w", err)
	}

	// Dual-engine OCR
	ocrResult, err := w.ocrSvc.Extract(ctx, ocr.ExtractionRequest{
		TenantID:    p.TenantID,
		FilePath:    p.FilePath,
		FileBytes:   fileBytes,
		ContentType: p.ContentType,
	})
	if err != nil {
		log.Printf("[worker] ocr error for invoice %s: %v", p.InvoiceID, err)
		_ = w.store.UpdateInvoiceData(ctx, p.InvoiceID, db.InvoiceUpdate{Status: "conflict"})
		return fmt.Errorf("ocr: %w", err)
	}

	invoiceStatus := "verified"
	if !ocrResult.Matched {
		invoiceStatus = "conflict"
	}

	d := ocrResult.Data
	_ = w.store.UpdateInvoiceData(ctx, p.InvoiceID, db.InvoiceUpdate{
		DocType:              d.DocType,
		VatInclusive:         d.VatInclusive,
		VatRate:              d.VatRate,
		VendorName:           d.VendorName,
		VendorTaxID:          d.VendorTaxID,
		VendorAddress:        d.VendorAddress,
		VendorBranchCode:     d.VendorBranchCode,
		BuyerName:            d.BuyerName,
		BuyerTaxID:           d.BuyerTaxID,
		BuyerAddress:         d.BuyerAddress,
		BuyerBranchCode:      d.BuyerBranchCode,
		InvoiceDocNo:         d.InvoiceDocNo,
		InvoiceDate:          d.InvoiceDate,
		VatExemptAmount:      d.VatExemptAmount,
		VatInclusiveSubtotal: d.VatInclusiveSubtotal,
		DiscountAmount:       d.DiscountAmount,
		TotalBeforeVAT:       d.TotalBeforeVAT,
		VATAmount:            d.VATAmount,
		TotalAmount:          d.TotalAmount,
		VATMathOK:            ocrResult.VATMathOK,
		Status:               invoiceStatus,
	})

	// Clear old items before saving new ones (handles re-run OCR)
	_ = w.store.DeleteInvoiceItemsByInvoiceID(ctx, p.InvoiceID)

	// Classify and save each line item
	for _, item := range ocrResult.Data.Items {
		classResult, err := w.classifySvc.ClassifyWithDB(ctx, classify.ClassificationInput{
			TenantID:    p.TenantID,
			Description: item.Description,
			Amount:      item.TotalPrice,
		})
		if err != nil {
			classResult = classify.ClassificationResult{AssetType: "pending", ClassifiedBy: "rule", Confidence: 0.3}
		}

		saved, err := w.store.CreateInvoiceItem(ctx, db.InvoiceItem{
			TenantID:    p.TenantID,
			BranchID:    p.BranchID,
			InvoiceID:   p.InvoiceID,
			ProductCode: item.ProductCode,
			Description: item.Description,
			Unit:        item.Unit,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			Discount:    item.Discount,
			TotalPrice:  item.TotalPrice,
			AssetType:    classResult.AssetType,
			ClassifiedBy: classResult.ClassifiedBy,
		})
		if err != nil {
			log.Printf("[worker] create invoice item error: %v", err)
			continue
		}

		// Route to HITL when classification is uncertain or OCR didn't verify
		if classResult.AssetType == "pending" || !ocrResult.Matched {
			reason := "classification_needed"
			if !ocrResult.Matched {
				reason = "ocr_mismatch"
			}
			hitlItem, err := w.store.CreateHitlItem(ctx, db.HitlQueueItem{
				TenantID:      p.TenantID,
				InvoiceItemID: saved.ID,
				Reason:        reason,
			})
			if err == nil && w.reviewerSvc != nil {
				w.reviewerSvc.AssignForHitl(ctx, hitlItem)
			}
		}
	}

	// Mark document_import as done
	if p.DocumentImportID != "" {
		_ = w.store.UpdateDocumentImportStatus(ctx, p.DocumentImportID, "done")
	}

	go w.notifyLineUser(p, invoiceStatus, len(ocrResult.Data.Items))

	log.Printf("[worker] invoice %s done: engine=%s status=%s items=%d matched=%v", p.InvoiceID, ocrResult.Engine, invoiceStatus, len(ocrResult.Data.Items), ocrResult.Matched)
	return nil
}

func (w *Worker) notifyLineUser(p ProcessInvoicePayload, status string, itemCount int) {
	if w.lineClient == nil || p.DocumentImportID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	doc, err := w.store.GetDocumentImport(ctx, p.DocumentImportID)
	if err != nil || doc.UserID == "" {
		return
	}
	user, err := w.store.GetUserByID(ctx, doc.UserID)
	if err != nil || user.LineUserID == "" {
		return
	}
	inv, err := w.store.GetInvoice(ctx, p.InvoiceID)
	if err != nil {
		return
	}

	var msg string
	if status == "verified" {
		msg = fmt.Sprintf("✅ ใบกำกับภาษีประมวลผลเสร็จแล้ว (#%d)\n💰 ยอดรวม: %.2f บาท\nรายการสินค้า: %d รายการ", inv.InvoiceNo, inv.TotalAmount, itemCount)
	} else {
		msg = fmt.Sprintf("⚠️ ใบกำกับภาษี (#%d) ต้องการการตรวจสอบเพิ่มเติม\nทีมงานจะดำเนินการและแจ้งผลให้ท่านทราบโดยเร็ว", inv.InvoiceNo)
	}
	if err := w.lineClient.Push(user.LineUserID, msg); err != nil {
		log.Printf("[worker] line push to %s: %v", user.LineUserID, err)
	}
}
