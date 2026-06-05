package classify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"tax-ocr/backend/internal/db"
)

type ClassificationInput struct {
	TenantID    string
	Description string
	Amount      float64
}

type ClassificationResult struct {
	AssetType    string
	ClassifiedBy string
	Confidence   float64
	Keyword      string
}

type Config struct {
	OpenAIKey string
}

type Service struct {
	store     *db.Store
	openaiKey string
	http      *http.Client
}

func NewService() *Service {
	return &Service{http: &http.Client{}}
}

func NewServiceWithConfig(store *db.Store, cfg Config) *Service {
	return &Service{
		store:     store,
		openaiKey: cfg.OpenAIKey,
		http:      &http.Client{},
	}
}

// Classify applies hardcoded keyword rules only (no DB, no AI).
func (s *Service) Classify(input ClassificationInput) ClassificationResult {
	return s.classifyByKeywords(input.Description)
}

// ClassifyWithDB applies DB rules → hardcoded rules → AI fallback → HITL pending.
// Auto-creates new rules when AI confidence ≥ 0.8 (self-learning).
func (s *Service) ClassifyWithDB(ctx context.Context, input ClassificationInput) (ClassificationResult, error) {
	// 1. DB rules for this tenant
	if s.store != nil && input.TenantID != "" {
		rules, err := s.store.ListRules(ctx, input.TenantID)
		if err == nil {
			for _, rule := range rules {
				if strings.Contains(strings.ToLower(input.Description), strings.ToLower(rule.Keyword)) {
					return ClassificationResult{
						AssetType:    rule.AssetType,
						ClassifiedBy: "rule",
						Confidence:   rule.Confidence,
						Keyword:      rule.Keyword,
					}, nil
				}
			}
		}
	}

	// 2. Hardcoded keyword rules
	result := s.classifyByKeywords(input.Description)
	if result.AssetType != "pending" {
		return result, nil
	}

	// 3. AI fallback
	if s.openaiKey == "" {
		return ClassificationResult{AssetType: "pending", ClassifiedBy: "rule", Confidence: 0.3}, nil
	}

	aiResult, err := s.classifyWithAI(ctx, input.Description)
	if err != nil {
		return ClassificationResult{AssetType: "pending", ClassifiedBy: "rule", Confidence: 0.3}, nil
	}

	// 4. Self-learning: persist rule if confidence is high enough
	if s.store != nil && aiResult.Confidence >= 0.8 && input.TenantID != "" && aiResult.Keyword != "" {
		_, _ = s.store.CreateRule(ctx, db.ClassificationRule{
			TenantID:   input.TenantID,
			Keyword:    aiResult.Keyword,
			AssetType:  aiResult.AssetType,
			Source:     "ai",
			Confidence: aiResult.Confidence,
		})
	}

	return aiResult, nil
}

var assetKeywords = []string{
	"computer", "คอมพิวเตอร์", "notebook", "laptop", "desktop",
	"printer", "เครื่องพิมพ์", "scanner", "projector",
	"server", "switch", "router", "network",
	"monitor", "จอ", "camera", "กล้อง",
	"vehicle", "รถ", "car", "truck",
	"machinery", "เครื่องจักร", "equipment",
	"furniture", "เฟอร์นิเจอร์", "chair", "desk", "table",
	"phone", "โทรศัพท์", "tablet",
	"air", "แอร์", "conditioner",
}

var expenseKeywords = []string{
	"paper", "กระดาษ", "stationery", "เครื่องเขียน",
	"fuel", "น้ำมัน", "gasoline",
	"service", "ค่าบริการ", "maintenance", "ซ่อม",
	"electric", "ไฟฟ้า", "utility", "water", "น้ำ",
	"internet", "อินเตอร์เน็ต",
	"postage", "courier", "ขนส่ง",
	"food", "อาหาร", "meal",
	"advertising", "โฆษณา",
	"insurance", "ประกัน",
}

func (s *Service) classifyByKeywords(desc string) ClassificationResult {
	text := strings.ToLower(desc)

	for _, kw := range assetKeywords {
		if strings.Contains(text, strings.ToLower(kw)) {
			return ClassificationResult{AssetType: "asset", ClassifiedBy: "rule", Confidence: 0.92, Keyword: kw}
		}
	}
	for _, kw := range expenseKeywords {
		if strings.Contains(text, strings.ToLower(kw)) {
			return ClassificationResult{AssetType: "expense", ClassifiedBy: "rule", Confidence: 0.88, Keyword: kw}
		}
	}

	return ClassificationResult{AssetType: "pending", ClassifiedBy: "rule", Confidence: 0.3}
}

func (s *Service) classifyWithAI(ctx context.Context, description string) (ClassificationResult, error) {
	prompt := fmt.Sprintf(
		`Classify this invoice line item for Thai accounting.
"asset" = long-lived items (equipment, machinery, vehicles, furniture, computers, etc.)
"expense" = consumed in operations (supplies, services, utilities, repairs, etc.)

Item: %s

Return JSON only: {"asset_type": "asset or expense", "confidence": 0.0-1.0, "keyword": "main keyword from description"}`,
		description,
	)

	body := map[string]any{
		"model":           "gpt-4o-mini",
		"messages":        []map[string]any{{"role": "user", "content": prompt}},
		"max_tokens":      100,
		"response_format": map[string]string{"type": "json_object"},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return ClassificationResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return ClassificationResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+s.openaiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		return ClassificationResult{}, err
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return ClassificationResult{}, fmt.Errorf("openai %d", resp.StatusCode)
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return ClassificationResult{}, err
	}
	if len(apiResp.Choices) == 0 {
		return ClassificationResult{}, fmt.Errorf("empty response")
	}

	var parsed struct {
		AssetType  string  `json:"asset_type"`
		Confidence float64 `json:"confidence"`
		Keyword    string  `json:"keyword"`
	}
	if err := json.Unmarshal([]byte(apiResp.Choices[0].Message.Content), &parsed); err != nil {
		return ClassificationResult{}, err
	}

	return ClassificationResult{
		AssetType:    parsed.AssetType,
		ClassifiedBy: "ai",
		Confidence:   parsed.Confidence,
		Keyword:      parsed.Keyword,
	}, nil
}
