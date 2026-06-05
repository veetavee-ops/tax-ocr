package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const gcvEndpoint = "https://vision.googleapis.com/v1/images:annotate"

type visionClient struct {
	apiKey string
	http   *http.Client
}

func newVisionClient(apiKey string) *visionClient {
	return &visionClient{apiKey: apiKey, http: &http.Client{}}
}

func (v *visionClient) extractText(ctx context.Context, imageBytes []byte) (string, error) {
	b64 := base64.StdEncoding.EncodeToString(imageBytes)

	body := map[string]any{
		"requests": []map[string]any{
			{
				"image":    map[string]any{"content": b64},
				"features": []map[string]any{{"type": "DOCUMENT_TEXT_DETECTION"}},
			},
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s?key=%s", gcvEndpoint, v.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gcv api %d: %s", resp.StatusCode, string(respBytes))
	}

	var apiResp struct {
		Responses []struct {
			FullTextAnnotation struct {
				Text string `json:"text"`
			} `json:"fullTextAnnotation"`
		} `json:"responses"`
	}
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return "", err
	}
	if len(apiResp.Responses) == 0 {
		return "", nil
	}
	return apiResp.Responses[0].FullTextAnnotation.Text, nil
}

var (
	taxIDRegex     = regexp.MustCompile(`\b\d{13}\b`)
	floatRegex     = regexp.MustCompile(`[\d,]+\.\d{2}`)
	vatAmtRegex    = regexp.MustCompile(`(?:ภาษีมูลค่าเพิ่ม|VAT|ภาษี\s*[\d.]+\s*%)[^\d]{0,40}([\d,]+\.\d{2})`)
	beforeVATRegex = regexp.MustCompile(`(?:มูลค่าสินค้าก่อนภาษี|มูลค่าสุทธิก่อนภาษี|ฐานภาษี|ราคาสินค้า|มูลค่าสินค้า|รวมก่อนภาษี|ราคารวมก่อน)[^\d]{0,40}([\d,]+\.\d{2})`)
	totalAmtRegex  = regexp.MustCompile(`(?:รวมจำนวนเงินทั้งสิ้น|รวมทั้งสิ้น|ยอดรวมทั้งสิ้น|จำนวนเงินรวม|รวมเงิน)[^\d]{0,40}([\d,]+\.\d{2})`)
	invoiceNoRegex = regexp.MustCompile(`(?:เลขที่(?:ใบกำกับ(?:ภาษี)?)?)[^\w\n]{0,10}([\w\-/\.]+)`)
	numDateRegex   = regexp.MustCompile(`(\d{1,2})[/\-](\d{1,2})[/\-](\d{4})`)
	thaiDateRegex  = regexp.MustCompile(`(\d{1,2})\s+(มกราคม|กุมภาพันธ์|มีนาคม|เมษายน|พฤษภาคม|มิถุนายน|กรกฎาคม|สิงหาคม|กันยายน|ตุลาคม|พฤศจิกายน|ธันวาคม)\s+(\d{4})`)
)

var thaiMonthNum = map[string]int{
	"มกราคม": 1, "กุมภาพันธ์": 2, "มีนาคม": 3, "เมษายน": 4,
	"พฤษภาคม": 5, "มิถุนายน": 6, "กรกฎาคม": 7, "สิงหาคม": 8,
	"กันยายน": 9, "ตุลาคม": 10, "พฤศจิกายน": 11, "ธันวาคม": 12,
}

var (
	// VAT-inclusive: footer has "มูลค่าที่มีภาษี" column — only present when line-item prices include VAT
	vatInclusiveMarker = regexp.MustCompile(`มูลค่าที่มีภาษี|ราคารวมภาษี|รวมภาษีแล้ว`)
	docTypeTax         = regexp.MustCompile(`ใบกำกับภาษี`)
	docTypeReceipt     = regexp.MustCompile(`ใบเสร็จรับเงิน`)
	docTypeDelivery    = regexp.MustCompile(`ใบส่งสินค้า`)
)

// classifyFromText uses keyword detection on Vision raw text to determine document type and
// whether line-item prices already include VAT. Returns (docType, vatInclusive).
// Called BEFORE GPT so the classification can be injected as context into the GPT prompt.
func classifyFromText(text string) (string, bool) {
	vatInclusive := vatInclusiveMarker.MatchString(text)

	docType := "unknown"
	switch {
	case docTypeTax.MatchString(text):
		docType = "tax_invoice"
	case docTypeReceipt.MatchString(text):
		docType = "receipt"
	case docTypeDelivery.MatchString(text):
		docType = "delivery_note"
	}

	return docType, vatInclusive
}

// parseInvoiceFromText heuristically extracts invoice header fields from raw OCR text.
// Used as cross-verify reference — not the primary extraction path.
func parseInvoiceFromText(text string) InvoiceData {
	data := InvoiceData{}

	// Tax ID (13 digits)
	if m := taxIDRegex.FindString(text); m != "" {
		data.VendorTaxID = m
	}

	// Invoice number
	if m := invoiceNoRegex.FindStringSubmatch(text); len(m) > 1 {
		data.InvoiceDocNo = strings.TrimSpace(m[1])
	}

	// Invoice date — numeric format dd/mm/yyyy or dd-mm-yyyy
	if m := numDateRegex.FindStringSubmatch(text); len(m) > 3 {
		day, _ := strconv.Atoi(m[1])
		month, _ := strconv.Atoi(m[2])
		year, _ := strconv.Atoi(m[3])
		if year > 2400 { // Buddhist year
			year -= 543
		}
		if year > 1900 && month >= 1 && month <= 12 && day >= 1 && day <= 31 {
			data.InvoiceDate = fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		}
	} else if m := thaiDateRegex.FindStringSubmatch(text); len(m) > 3 {
		// Thai full month name: "1 มิถุนายน 2568"
		day, _ := strconv.Atoi(m[1])
		monthNum := thaiMonthNum[m[2]]
		year, _ := strconv.Atoi(m[3])
		if year > 2400 {
			year -= 543
		}
		if year > 1900 && monthNum >= 1 {
			data.InvoiceDate = fmt.Sprintf("%04d-%02d-%02d", year, monthNum, day)
		}
	}

	// VAT amount (look near ภาษีมูลค่าเพิ่ม / VAT label)
	if m := vatAmtRegex.FindStringSubmatch(text); len(m) > 1 {
		if f, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", ""), 64); err == nil {
			data.VATAmount = f
		}
	}

	// Total before VAT (look near ราคาสินค้า / มูลค่าสินค้า label)
	if m := beforeVATRegex.FindStringSubmatch(text); len(m) > 1 {
		if f, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", ""), 64); err == nil {
			data.TotalBeforeVAT = f
		}
	}

	// Total amount — prefer labeled footer (รวมจำนวนเงินทั้งสิ้น), fallback to largest number
	if m := totalAmtRegex.FindStringSubmatch(text); len(m) > 1 {
		if f, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", ""), 64); err == nil {
			data.TotalAmount = f
		}
	}
	if data.TotalAmount == 0 {
		var amounts []float64
		for _, m := range floatRegex.FindAllString(text, -1) {
			if f, err := strconv.ParseFloat(strings.ReplaceAll(m, ",", ""), 64); err == nil {
				amounts = append(amounts, f)
			}
		}
		for _, a := range amounts {
			if a > data.TotalAmount {
				data.TotalAmount = a
			}
		}
	}

	return data
}
