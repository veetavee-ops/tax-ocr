package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const gptModel = "gpt-4o-mini"
const gptEndpoint = "https://api.openai.com/v1/chat/completions"

type gptClient struct {
	apiKey string
	http   *http.Client
}

func newGPTClient(apiKey string) *gptClient {
	return &gptClient{apiKey: apiKey, http: &http.Client{}}
}

const invoiceJSONSchema = `{
  "vendor_name": "",
  "vendor_tax_id": "",
  "invoice_doc_no": "",
  "invoice_date": "",
  "total_before_vat": 0.00,
  "vat_amount": 0.00,
  "total_amount": 0.00,
  "items": [{"description": "", "quantity": 1.0, "unit_price": 0.00, "total_price": 0.00}]
}`

const gptSystemPrompt = `You are a Thai tax invoice (ใบกำกับภาษี) data extractor. Return ONLY valid JSON — no markdown, no explanation.

Field mapping — find these Thai labels in the invoice:
- vendor_name     : ชื่อบริษัท/ร้านค้าของผู้ขาย (the seller, not the buyer)
- vendor_tax_id   : เลขประจำตัวผู้เสียภาษี — exactly 13 digits, strip spaces/dashes/parentheses
- invoice_doc_no  : เลขที่ / เลขที่ใบกำกับภาษี (e.g. "IV-001", "TAX2568-001", "001/2568")
- invoice_date    : วันที่ออกใบกำกับ → YYYY-MM-DD. Buddhist year (พ.ศ.) subtract 543: e.g. 2568→2025
- total_before_vat: ONLY from the FOOTER summary section — look for "มูลค่าสินค้าก่อนภาษี" / "ฐานภาษี" / "ราคาก่อนภาษี" / "มูลค่าสุทธิก่อนภาษี" / "รวมก่อนภาษีมูลค่าเพิ่ม". WARNING: "มูลค่าที่มีภาษี" is the VAT-INCLUSIVE amount — do NOT use it as total_before_vat.
- vat_amount      : ภาษีมูลค่าเพิ่ม — the baht amount printed AFTER the rate (e.g. "7.00%  26.32" → 26.32, NOT 7.00). Always includes decimals.
- total_amount    : ยอดรวมทั้งสิ้น / รวมจำนวนเงินทั้งสิ้น (grand total including VAT)
- items[].description: ชื่อสินค้า/บริการ
- items[].quantity   : จำนวน
- items[].unit_price : ราคาต่อหน่วย (as printed — may include VAT)
- items[].total_price: จำนวนเงิน per line (as printed — may include VAT)

CRITICAL RULES:
1. Copy numbers EXACTLY as printed — 26.32 stays 26.32, 402.32 stays 402.32. Never round, truncate, or recalculate.
2. Do NOT adjust any number to make arithmetic balance. Report what is physically printed, even if the numbers do not add up. The cross-verification engine will flag mismatches.
3. vendor_tax_id must be exactly 13 digits — remove all spaces, dashes, and parentheses
4. Missing text → "" ; missing number → 0
5. Return raw JSON only, no markdown fences
6. VAT-INCLUSIVE invoices: Some Thai invoices show line item prices that already include VAT (ราคารวมภาษีแล้ว). The footer will show "มูลค่าสินค้าก่อนภาษี" (pre-VAT base, smaller number) separately from "มูลค่าที่มีภาษี" (VAT-inclusive subtotal, larger number). ALWAYS use "มูลค่าสินค้าก่อนภาษี" as total_before_vat — never use "มูลค่าที่มีภาษี".`

func (g *gptClient) extractFromImage(ctx context.Context, imageBytes []byte, contentType string) (InvoiceData, error) {
	b64 := base64.StdEncoding.EncodeToString(imageBytes)

	body := map[string]any{
		"model": gptModel,
		"messages": []map[string]any{
			{"role": "system", "content": gptSystemPrompt},
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "text", "text": "Extract all invoice data from this image. Return JSON:\n" + invoiceJSONSchema},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url":    fmt.Sprintf("data:%s;base64,%s", contentType, b64),
							"detail": "high",
						},
					},
				},
			},
		},
		"max_tokens":      1500,
		"response_format": map[string]string{"type": "json_object"},
	}

	return g.sendRequest(ctx, body)
}

func (g *gptClient) extractFromText(ctx context.Context, rawText string) (InvoiceData, error) {
	userMsg := "Extract all invoice data from this Thai tax invoice text. Return JSON:\n" +
		invoiceJSONSchema + "\n\nInvoice text:\n" + rawText

	body := map[string]any{
		"model": gptModel,
		"messages": []map[string]any{
			{"role": "system", "content": gptSystemPrompt},
			{"role": "user", "content": userMsg},
		},
		"max_tokens":      1500,
		"response_format": map[string]string{"type": "json_object"},
	}

	return g.sendRequest(ctx, body)
}

func (g *gptClient) sendRequest(ctx context.Context, body map[string]any) (InvoiceData, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return InvoiceData{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", gptEndpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return InvoiceData{}, err
	}
	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.http.Do(req)
	if err != nil {
		return InvoiceData{}, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return InvoiceData{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return InvoiceData{}, fmt.Errorf("openai api %d: %s", resp.StatusCode, string(respBytes))
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return InvoiceData{}, err
	}
	if len(apiResp.Choices) == 0 {
		return InvoiceData{}, fmt.Errorf("empty choices from openai")
	}

	content := strings.TrimSpace(apiResp.Choices[0].Message.Content)
	var data InvoiceData
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return InvoiceData{}, fmt.Errorf("parse gpt response: %w", err)
	}
	return data, nil
}
