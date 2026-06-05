package ocr

import "math"

const amountTolerance = 0.5  // tolerance for GPT vs Vision cross-check
const vatMathTolerance = 0.02 // tighter tolerance for stated VAT vs 7% calculation
const vatRate = 0.07

type verifyResult struct {
	taxIDMatch  bool
	totalsMatch bool
	vatMathOK   bool // stated VAT ≈ total_before_vat × 7%
	matched     bool
}

func crossVerify(gpt, vision InvoiceData) verifyResult {
	r := verifyResult{}

	// Tax ID: exact match; if either side is empty we give benefit of the doubt
	if gpt.VendorTaxID != "" && vision.VendorTaxID != "" {
		r.taxIDMatch = gpt.VendorTaxID == vision.VendorTaxID
	} else {
		r.taxIDMatch = true
	}

	// Total + VAT cross-check between GPT and Vision
	if vision.TotalAmount == 0 {
		r.totalsMatch = true
	} else {
		totalOK := math.Abs(gpt.TotalAmount-vision.TotalAmount) <= amountTolerance
		vatCrossOK := vision.VATAmount == 0 || math.Abs(gpt.VATAmount-vision.VATAmount) <= amountTolerance
		r.totalsMatch = totalOK && vatCrossOK
	}

	// Arithmetic check: before_vat + vat_amount should equal total_amount
	if gpt.TotalBeforeVAT > 0 && gpt.TotalAmount > 0 {
		arithmeticOK := math.Abs((gpt.TotalBeforeVAT+gpt.VATAmount)-gpt.TotalAmount) <= amountTolerance
		r.totalsMatch = r.totalsMatch && arithmeticOK
	}

	// Mathematical VAT validation: stated VAT should equal total_before_vat × 7%
	if gpt.TotalBeforeVAT > 0 {
		expectedVAT := gpt.TotalBeforeVAT * vatRate
		r.vatMathOK = math.Abs(gpt.VATAmount-expectedVAT) <= vatMathTolerance
	} else {
		r.vatMathOK = true // can't validate without before-VAT amount
	}

	// vatMathOK is stored as a flag only — not part of matched
	// Thai invoices often round VAT differently (truncate vs round-half-up)
	r.matched = r.taxIDMatch && r.totalsMatch
	return r
}
