package ocr

import (
	"context"
	"time"
)

// ExtractionRequest represents a file waiting for OCR.
type ExtractionRequest struct {
	TenantID string
	FilePath string
}

type ExtractionResult struct {
	Text       string
	Confidence float64
	Engine     string
	ProcessedAt time.Time
}

// Service wraps OCR dependencies. External engines can be injected later.
type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Extract(ctx context.Context, req ExtractionRequest) (ExtractionResult, error) {
	select {
	case <-ctx.Done():
		return ExtractionResult{}, ctx.Err()
	default:
	}

	return ExtractionResult{
		Text:       "OCR pending integration",
		Confidence: 0.0,
		Engine:     "stub",
		ProcessedAt: time.Now().UTC(),
	}, nil
}
