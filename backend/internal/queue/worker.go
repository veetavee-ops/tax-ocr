package queue

import (
	"context"

	"tax-ocr/backend/internal/classify"
	"tax-ocr/backend/internal/ocr"
)

// WorkerConfig captures the queue dependencies that will be wired in the MVP.
type WorkerConfig struct {
	RedisAddr string
}

type Worker struct {
	config      WorkerConfig
	ocrService  *ocr.Service
	classifySvc *classify.Service
}

func NewWorker(cfg WorkerConfig, ocrService *ocr.Service, classifySvc *classify.Service) *Worker {
	if ocrService == nil {
		ocrService = ocr.NewService()
	}
	if classifySvc == nil {
		classifySvc = classify.NewService()
	}

	return &Worker{
		config:      cfg,
		ocrService:  ocrService,
		classifySvc: classifySvc,
	}
}

type ProcessResult struct {
	OCRText       string
	OCRConfidence float64
	AssetType     string
	ClassifiedBy  string
}

func (w *Worker) ProcessDocument(ctx context.Context, req ocr.ExtractionRequest) (ProcessResult, error) {
	ocrResult, err := w.ocrService.Extract(ctx, req)
	if err != nil {
		return ProcessResult{}, err
	}

	classifyResult := w.classifySvc.Classify(classify.ClassificationInput{
		Description: ocrResult.Text,
		Amount:      0,
	})

	return ProcessResult{
		OCRText:       ocrResult.Text,
		OCRConfidence: ocrResult.Confidence,
		AssetType:     classifyResult.AssetType,
		ClassifiedBy:  classifyResult.ClassifiedBy,
	}, nil
}
