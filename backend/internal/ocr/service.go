package ocr

import (
	"context"
	"log"
	"sync"
	"time"
)

type Config struct {
	OpenAIKey string
	GCVKey    string
}

type Service struct {
	mu     sync.RWMutex
	gpt    *gptClient
	vision *visionClient
}

func NewService() *Service {
	return &Service{}
}

func NewServiceWithConfig(cfg Config) *Service {
	svc := &Service{}
	svc.applyConfig(cfg)
	return svc
}

// UpdateConfig hot-reloads API keys at runtime (thread-safe).
func (s *Service) UpdateConfig(cfg Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.applyConfig(cfg)
}

// HasConfig reports whether at least one engine is configured.
func (s *Service) HasConfig() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.gpt != nil || s.vision != nil
}

func (s *Service) applyConfig(cfg Config) {
	if cfg.OpenAIKey != "" {
		s.gpt = newGPTClient(cfg.OpenAIKey)
	} else {
		s.gpt = nil
	}
	if cfg.GCVKey != "" {
		s.vision = newVisionClient(cfg.GCVKey)
	} else {
		s.vision = nil
	}
}

// Extract runs dual-engine OCR; falls back to stub when no keys configured.
func (s *Service) Extract(ctx context.Context, req ExtractionRequest) (ExtractionResult, error) {
	s.mu.RLock()
	gpt := s.gpt
	vision := s.vision
	s.mu.RUnlock()

	if gpt == nil && vision == nil {
		return ExtractionResult{Engine: "stub", ProcessedAt: time.Now().UTC()}, nil
	}
	if !isImage(req.ContentType) {
		return ExtractionResult{Matched: false, Engine: "stub-pdf", ProcessedAt: time.Now().UTC()}, nil
	}

	return s.dualExtract(ctx, req, gpt, vision)
}

// ExtractDebug runs dual-engine and returns both engines' results separately.
func (s *Service) ExtractDebug(ctx context.Context, req ExtractionRequest) (DebugResult, error) {
	s.mu.RLock()
	gpt := s.gpt
	vision := s.vision
	s.mu.RUnlock()

	if gpt == nil && vision == nil {
		return DebugResult{Engine: "stub"}, nil
	}
	if !isImage(req.ContentType) {
		return DebugResult{Engine: "stub-pdf"}, nil
	}

	var rawText string
	var visionData InvoiceData

	if vision != nil {
		text, err := vision.extractText(ctx, req.FileBytes)
		if err != nil {
			return DebugResult{}, err
		}
		rawText = text
		visionData = parseInvoiceFromText(text)
	}

	var gptData InvoiceData
	if gpt != nil {
		var err error
		if len(req.FileBytes) > 0 {
			gptData, err = gpt.extractFromImage(ctx, req.FileBytes, req.ContentType)
		} else {
			gptData, err = gpt.extractFromText(ctx, rawText)
		}
		if err != nil {
			return DebugResult{}, err
		}
	}

	verify := crossVerify(gptData, visionData)
	return DebugResult{
		GPT:     gptData,
		Vision:  visionData,
		Matched: verify.matched,
		RawText: rawText,
		Engine:  "dual",
	}, nil
}

func (s *Service) dualExtract(ctx context.Context, req ExtractionRequest, gpt *gptClient, vision *visionClient) (ExtractionResult, error) {
	now := time.Now().UTC()

	var rawText string
	var visionData InvoiceData

	if vision != nil {
		text, err := vision.extractText(ctx, req.FileBytes)
		if err != nil {
			return ExtractionResult{}, err
		}
		rawText = text
		visionData = parseInvoiceFromText(text)
	}

	// Step 2: Rule-based classification from Vision raw text.
	// Vision reads Thai characters accurately; rules detect doc_type + vat_inclusive
	// without relying on GPT to understand Thai semantics.
	docType, vatInclusive := classifyFromText(rawText)
	log.Printf("[ocr/vision/raw] %s", rawText)
	log.Printf("[ocr/classify] doc_type=%s vat_inclusive=%v", docType, vatInclusive)
	log.Printf("[ocr/vision] tax_id=%q before_vat=%.2f vat=%.2f total=%.2f",
		visionData.VendorTaxID, visionData.TotalBeforeVAT, visionData.VATAmount, visionData.TotalAmount)

	// Step 3: GPT extracts ALL fields. It receives Vision's raw text + pre-classified context
	// so it knows the document type and VAT treatment without reading Thai on its own.
	// GPT is the sole authority for extracted values — no Vision override.
	var gptData InvoiceData
	if gpt != nil {
		var err error
		if rawText != "" {
			gptData, err = gpt.extractFromTextWithContext(ctx, rawText, docType, vatInclusive)
		} else if len(req.FileBytes) > 0 {
			gptData, err = gpt.extractFromImage(ctx, req.FileBytes, req.ContentType)
		}
		if err != nil {
			return ExtractionResult{}, err
		}
	}

	log.Printf("[ocr/gpt] tax_id=%q before_vat=%.2f vat=%.2f total=%.2f items=%d doc_type=%s vat_inc=%v",
		gptData.VendorTaxID, gptData.TotalBeforeVAT, gptData.VATAmount, gptData.TotalAmount,
		len(gptData.Items), gptData.DocType, gptData.VatInclusive)

	verify := crossVerify(gptData, visionData)
	confidence := 0.6
	if verify.matched {
		confidence = 0.95
	}

	return ExtractionResult{
		Data:        gptData,
		Matched:     verify.matched,
		VATMathOK:   verify.vatMathOK,
		RawText:     rawText,
		Confidence:  confidence,
		Engine:      "dual",
		ProcessedAt: now,
	}, nil
}

func isImage(contentType string) bool {
	return contentType == "image/jpeg" || contentType == "image/png" || contentType == "image/jpg"
}
