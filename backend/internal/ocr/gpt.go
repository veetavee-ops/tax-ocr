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
  "doc_type": "tax_invoice",
  "vat_inclusive": false,
  "vat_rate": 7.00,
  "vendor_name": "",
  "vendor_tax_id": "",
  "vendor_address": "",
  "vendor_branch_code": "",
  "buyer_name": "",
  "buyer_tax_id": "",
  "buyer_address": "",
  "buyer_branch_code": "",
  "invoice_doc_no": "",
  "invoice_date": "",
  "vat_exempt_amount": 0.00,
  "vat_inclusive_subtotal": 0.00,
  "discount_amount": 0.00,
  "total_before_vat": 0.00,
  "vat_amount": 0.00,
  "total_amount": 0.00,
  "items": [{"product_code": "", "description": "", "unit": "", "quantity": 1.0, "unit_price": 0.00, "discount": 0.00, "total_price": 0.00}]
}`

const gptSystemPrompt = `You are a Thai document analyzer and data extractor. Return ONLY valid JSON — no markdown, no explanation.

## PHASE 1 — Classify the document first

Determine:
- doc_type: "tax_invoice" (ใบกำกับภาษี) | "receipt" (ใบเสร็จรับเงิน) | "delivery_note" (ใบส่งสินค้า) | "unknown"
- vat_inclusive: true if line item prices ALREADY INCLUDE VAT (ราคารวมภาษีแล้ว), false if prices are pre-VAT
  → Clue: if the footer shows "มูลค่าสินค้าก่อนภาษี" that is LOWER than the sum of line items → vat_inclusive = true
- vat_rate: VAT percentage on this document (typically 7.00, or 0.00 if exempt)

## PHASE 2 — Extract all fields based on classification

SELLER (ผู้ขาย — the company issuing the document):
- vendor_name        : ชื่อบริษัท/ร้านค้าผู้ออกเอกสาร
- vendor_tax_id      : เลขประจำตัวผู้เสียภาษี ผู้ขาย — exactly 13 digits
- vendor_address     : ที่อยู่ผู้ขาย (full address as printed, free text)
- vendor_branch_code : รหัสสาขาผู้ขาย เช่น "00027", "สำนักงานใหญ่"

BUYER (ผู้ซื้อ — the company receiving the document):
- buyer_name        : ชื่อบริษัท/ลูกค้าผู้รับเอกสาร
- buyer_tax_id      : เลขประจำตัวผู้เสียภาษี ผู้ซื้อ — exactly 13 digits
- buyer_address     : ที่อยู่ผู้ซื้อ (full address as printed, free text)
- buyer_branch_code : รหัสสาขาผู้ซื้อ เช่น "Head Office", "สำนักงานใหญ่"

DOCUMENT:
- invoice_doc_no : เลขที่เอกสาร / เลขที่ใบกำกับภาษี
- invoice_date   : วันที่ → YYYY-MM-DD. Buddhist year subtract 543 (2568→2025)

FINANCIAL SUMMARY (from footer section only):
- vat_exempt_amount      : มูลค่าที่ยกเว้นภาษี
- vat_inclusive_subtotal : มูลค่าที่มีภาษี (VAT-inclusive subtotal shown in footer)
- discount_amount        : ส่วนลดรวม
- total_before_vat       : มูลค่าสินค้าก่อนภาษี / ฐานภาษี / รวมก่อนภาษี (pre-VAT base from footer)
  → If vat_inclusive=true: use "มูลค่าสินค้าก่อนภาษี" — NOT "มูลค่าที่มีภาษี"
  → If vat_inclusive=false: use "รวมก่อนภาษี" / "ราคาก่อนภาษี"
- vat_amount : ภาษีมูลค่าเพิ่ม — baht amount AFTER the rate% (e.g. "7.00% 136.21" → 136.21)
- total_amount : รวมจำนวนเงินทั้งสิ้น / ยอดรวมทั้งสิ้น

LINE ITEMS (each row in the product/service table):
- items[].product_code : รหัสสินค้า / บาร์โค้ด
- items[].description  : ชื่อสินค้า/บริการ
- items[].unit         : หน่วย (Piece, กล่อง, ชิ้น, etc.)
- items[].quantity     : จำนวน
- items[].unit_price   : ราคาต่อหน่วย (as printed)
- items[].discount     : ส่วนลดต่อบรรทัด (0 if none)
- items[].total_price  : จำนวนเงิน per line (as printed)

CRITICAL RULES:
1. Copy numbers EXACTLY as printed — never round, truncate, or recalculate.
2. Do NOT adjust numbers to make arithmetic balance. Report what is printed.
3. Tax IDs must be exactly 13 digits — strip spaces, dashes, parentheses.
4. Missing text → ""; missing number → 0
5. Return raw JSON only, no markdown fences.
6. discount_amount is the TRADE DISCOUNT row only (usually 0). It is NOT the pre-VAT base.
   If discount_amount ≈ total_before_vat that means you made an error — set discount_amount = 0.
7. The footer table often has 3-column headers: มูลค่าที่ยกเว้นภาษี | มูลค่าที่มีภาษี | ส่วนลดรวม
   and 3-column values below: มูลค่าสินค้าก่อนภาษี | ภาษีมูลค่าเพิ่ม | รวมจำนวนเงินทั้งสิ้น
   Do NOT cross-map headers from row 1 to values from row 2. Each label belongs to its own value on the same row.`

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
	return g.extractFromTextWithContext(ctx, rawText, "", false, InvoiceData{})
}

// extractFromTextWithContext injects pre-classified doc_type and vat_inclusive (detected by Vision
// keyword rules) into the prompt so GPT does not need to figure these out from Thai text.
// visionHints carries values Vision regex already extracted — used as math anchors when vat_inclusive=true.
func (g *gptClient) extractFromTextWithContext(ctx context.Context, rawText, docType string, vatInclusive bool, visionHints InvoiceData) (InvoiceData, error) {
	vatDesc := "ราคาสินค้าในตาราง EXCLUDE VAT (ยังไม่รวม VAT) — total_before_vat คือ sum of line items"
	if vatInclusive {
		vatDesc = "ราคาสินค้าในตาราง INCLUDE VAT แล้ว — total_before_vat ต้องใช้ค่าจากป้าย มูลค่าสินค้าก่อนภาษี ใน footer เท่านั้น (ไม่ใช่ มูลค่าที่มีภาษี)"
	}

	ctx0 := ""
	if docType != "" {
		ctx0 = "PRE-CLASSIFIED (confirmed from document text — do not override):\n" +
			"  doc_type: " + docType + "\n" +
			"  vat_inclusive: " + boolStr(vatInclusive) + " — " + vatDesc + "\n"

		// When vat_inclusive=true and Vision found the inclusive subtotal, provide explicit math
		// so GPT doesn't confuse มูลค่าที่มีภาษี (2082) with total_before_vat (1945.79).
		if vatInclusive && visionHints.VatInclusiveSubtotal > 0 {
			preVAT := visionHints.VatInclusiveSubtotal / 1.07
			vatAmt := visionHints.VatInclusiveSubtotal - preVAT
			ctx0 += fmt.Sprintf(
				"  VISION HINT: มูลค่าที่มีภาษี=%.2f → vat_inclusive_subtotal=%.2f, total_before_vat≈%.2f, vat_amount≈%.2f\n"+
					"  → total_before_vat MUST be ~%.2f (NOT %.2f)\n",
				visionHints.VatInclusiveSubtotal,
				visionHints.VatInclusiveSubtotal,
				preVAT, vatAmt, preVAT,
				visionHints.VatInclusiveSubtotal,
			)
		} else if vatInclusive && visionHints.TotalAmount > 0 {
			// Fallback: derive from total_amount if subtotal not found
			preVAT := visionHints.TotalAmount / 1.07
			vatAmt := visionHints.TotalAmount - preVAT
			ctx0 += fmt.Sprintf(
				"  VISION HINT: total_amount=%.2f → estimated total_before_vat≈%.2f, vat_amount≈%.2f\n",
				visionHints.TotalAmount, preVAT, vatAmt,
			)
		}

		ctx0 += "\n"
	}

	userMsg := ctx0 + "Extract all invoice data from this Thai invoice text. Return JSON:\n" +
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

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
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
