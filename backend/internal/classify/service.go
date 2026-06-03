package classify

import "strings"

// ClassificationInput represents an invoice line item waiting for classification.
type ClassificationInput struct {
	Description string
	Amount      float64
}

type ClassificationResult struct {
	AssetType    string
	ClassifiedBy string
	Confidence   float64
}

// Service applies lightweight rules before AI/HITL integration.
type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Classify(input ClassificationInput) ClassificationResult {
	text := strings.ToLower(input.Description)

	if strings.Contains(text, "computer") || strings.Contains(text, "printer") || strings.Contains(text, "server") {
		return ClassificationResult{AssetType: "asset", ClassifiedBy: "rule", Confidence: 0.92}
	}

	if strings.Contains(text, "paper") || strings.Contains(text, "fuel") || strings.Contains(text, "service") {
		return ClassificationResult{AssetType: "expense", ClassifiedBy: "rule", Confidence: 0.88}
	}

	return ClassificationResult{AssetType: "pending", ClassifiedBy: "rule", Confidence: 0.3}
}
