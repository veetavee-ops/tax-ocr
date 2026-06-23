package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

// CompanyData holds extracted company/tenant registration info.
type CompanyData struct {
	Name         string       `json:"name"`
	TaxID        string       `json:"tax_id"`
	Address      string       `json:"address"`
	BusinessType string       `json:"business_type"` // service | trading | construction
	Branches     []BranchData `json:"branches"`
}

// BranchData holds extracted branch info from the document.
type BranchData struct {
	Name    string `json:"name"`
	Code    string `json:"code"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
}

const companySchema = `{
  "name": "",
  "tax_id": "",
  "address": "",
  "business_type": "service",
  "branches": [{"name": "", "code": "", "address": "", "phone": ""}]
}`

const companySystemPrompt = `You are a Thai business document analyzer. Extract company registration information from Thai documents such as:
- หนังสือรับรองบริษัท (Company Certificate)
- ใบทะเบียนภาษีมูลค่าเพิ่ม ภพ.01 / ภพ.20 (VAT Registration)
- หนังสือรับรองการจดทะเบียน (Registration Certificate)

Return ONLY valid JSON — no markdown, no explanation.

Fields to extract:
- name: ชื่อบริษัท/ห้างหุ้นส่วน (full legal name as printed)
- tax_id: เลขประจำตัวผู้เสียภาษี OR เลขทะเบียนนิติบุคคล — exactly 13 digits, strip spaces/dashes
- address: ที่อยู่จดทะเบียน (full registered address as printed)
- business_type: classify as one of: "trading" (ซื้อมาขายไป/ผลิต/นำเข้า), "service" (บริการ/ให้คำปรึกษา), "construction" (รับเหมาก่อสร้าง)
  When unclear, default to "service"
- branches: list of branch locations found (NOT including สำนักงานใหญ่/HQ)
  Each branch:
  - name: ชื่อสาขา (e.g. "สาขาสีลม")
  - code: รหัสสาขา — normalize "สำนักงานใหญ่"/"HQ"/"Head Office" to "00000"; pad numeric codes to 5 digits
  - address: ที่อยู่สาขา
  - phone: เบอร์โทรศัพท์ (empty string if not found)

RULES:
1. tax_id must be exactly 13 digits — strip all spaces, dashes, parentheses
2. Do NOT include สำนักงานใหญ่ as a branch — it is the main office address
3. If no branches found, return empty array []
4. Missing text → empty string ""
5. Return raw JSON only`

// extractTextFromPDF extracts plain text from a digital (selectable-text) PDF.
func extractTextFromPDF(data []byte) (string, error) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("pdf open: %w", err)
	}
	var sb strings.Builder
	for i := 1; i <= r.NumPage(); i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			continue
		}
		sb.WriteString(text)
		sb.WriteString("\n")
	}
	return strings.TrimSpace(sb.String()), nil
}

// ExtractCompanyInfo extracts company registration data from an image or digital PDF.
// PDF path: extract text via Go PDF library → GPT (no Vision needed).
// Image path: Vision → GPT.
func (s *Service) ExtractCompanyInfo(ctx context.Context, fileBytes []byte, contentType string) (CompanyData, error) {
	s.mu.RLock()
	gpt := s.gpt
	vision := s.vision
	s.mu.RUnlock()

	if gpt == nil {
		return CompanyData{}, fmt.Errorf("GPT not configured")
	}

	// Digital PDF: extract text locally, no Vision call needed
	if contentType == "application/pdf" {
		text, err := extractTextFromPDF(fileBytes)
		if err != nil {
			return CompanyData{}, fmt.Errorf("pdf: %w", err)
		}
		if text == "" {
			return CompanyData{}, fmt.Errorf("PDF ไม่มีข้อความ (อาจเป็น scanned PDF — กรุณาใช้ไฟล์รูปแทน)")
		}
		return gpt.extractCompanyInfo(ctx, text, nil, "")
	}

	// Image: Vision → GPT
	var rawText string
	if vision != nil {
		text, err := vision.extractText(ctx, fileBytes)
		if err != nil {
			return CompanyData{}, fmt.Errorf("vision: %w", err)
		}
		rawText = text
	}

	return gpt.extractCompanyInfo(ctx, rawText, fileBytes, contentType)
}

func (g *gptClient) extractCompanyInfo(ctx context.Context, rawText string, imageBytes []byte, contentType string) (CompanyData, error) {
	var userContent []map[string]any

	if rawText != "" {
		userContent = []map[string]any{{
			"type": "text",
			"text": "Extract company registration data from this Thai document text:\n\n" + rawText + "\n\nReturn JSON:\n" + companySchema,
		}}
	} else {
		// Fallback: send image directly when Vision produced no text
		b64 := base64.StdEncoding.EncodeToString(imageBytes)
		userContent = []map[string]any{
			{"type": "text", "text": "Extract company registration data from this Thai document image. Return JSON:\n" + companySchema},
			{"type": "image_url", "image_url": map[string]string{
				"url":    fmt.Sprintf("data:%s;base64,%s", contentType, b64),
				"detail": "high",
			}},
		}
	}

	body := map[string]any{
		"model": gptModel,
		"messages": []map[string]any{
			{"role": "system", "content": companySystemPrompt},
			{"role": "user", "content": userContent},
		},
		"max_tokens":      800,
		"response_format": map[string]string{"type": "json_object"},
	}

	raw, err := g.sendRawRequest(ctx, body)
	if err != nil {
		return CompanyData{}, err
	}

	var result CompanyData
	if err := json.Unmarshal(raw, &result); err != nil {
		return CompanyData{}, fmt.Errorf("parse company JSON: %w — raw: %s", err, string(raw))
	}

	// Sanitize tax_id: digits only
	result.TaxID = strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, result.TaxID)

	if result.BusinessType == "" {
		result.BusinessType = "service"
	}
	if result.Branches == nil {
		result.Branches = []BranchData{}
	}

	return result, nil
}
