package parsers

import (
	"regexp"
	"strings"

	"github.com/gocolly/colly/v2"
)

var (
	// currencyRegex matches pure currency symbols (e.g., "$$$$", "€€€€")
	currencyRegex = regexp.MustCompile(`^[€$£¥₩₽₹฿₺﷼₫]+$`)

	// priceCodeRegex matches price with currency code (e.g., "1,800 NOK", "155 EUR", "300 - 2,000 MOP")
	priceCodeRegex = regexp.MustCompile(`^[0-9][0-9,.\-\s]*[0-9]\s*[A-Z]{2,4}$`)

	// priceRangeRegex matches numeric price ranges (e.g., "155 - 380", "300 - 2,000")
	priceRangeRegex = regexp.MustCompile(`^[0-9][0-9,.\-\s]*[0-9]$`)

	// overUnderRegex matches "Over X" or "Under X" patterns (e.g., "Over 75 USD")
	overUnderRegex = regexp.MustCompile(`^(Over|Under)\s+\d+`)

	// betweenRegex matches "Between X and Y [CURRENCY]" patterns (e.g., "Between 350 and 500 HKD")
	betweenRegex = regexp.MustCompile(`^Between\s+\d+.*\d+\s+[A-Z]{2,4}$`)

	// toRangeRegex matches "X to Y [CURRENCY]" patterns (e.g., "500 to 1500 TWD")
	toRangeRegex = regexp.MustCompile(`^\d+\s+to\s+\d+\s+[A-Z]{2,4}$`)

	// lessThanRegex matches "Less than X [CURRENCY]" patterns (e.g., "Less than 200 THB")
	lessThanRegex = regexp.MustCompile(`(?i)^Less than \d+(\.\d+)?\s*[A-Z]{2,4}$`)
)

// ExtractPrice centralizes price extraction logic and fallbacks.
func ExtractPrice(e *colly.XMLElement) string {
	price := tryAwardSelectors(e, "price", parsePrice)
	if price != "" {
		return price
	}
	scriptContent := FindDLayerScript(e)
	price = ParseDLayerValue(scriptContent, "price")
	return mapPrice(price)
}

// parsePrice processes and validates a price text candidate against known patterns.
func parsePrice(text string) string {
	const priceSeparators = "·•"
	candidate := normalizePriceText(text, priceSeparators)
	if candidate == "" {
		return ""
	}

	priceValidators := []func(string) string{
		validateCurrencySymbols,
		validatePriceWithCurrencyCode,
		validatePriceRange,
		validateOverUnderPrice,
		validateBetweenPrice,
		validateToRangePrice,
		validateLessThanPrice,
	}

	for _, validator := range priceValidators {
		if result := validator(candidate); result != "" {
			return result
		}
	}
	return ""
}

// mapPrice maps CAT_P01 ... CAT_P04 to $, $$, $$$, $$$$.
func mapPrice(price string) string {
	price = strings.TrimSpace(price)
	if price == "" {
		return ""
	}
	switch price {
	case "CAT_P01":
		return "$"
	case "CAT_P02":
		return "$$"
	case "CAT_P03":
		return "$$$"
	case "CAT_P04":
		return "$$$$"
	default:
		return price
	}
}

// validateCurrencySymbols checks if text contains only currency symbols.
// e.g. "$$$$", "€€€€", "£££", "¥¥¥"
func validateCurrencySymbols(text string) string {
	if currencyRegex.MatchString(text) {
		return text
	}
	return ""
}

// validatePriceWithCurrencyCode checks for price with currency code.
// e.g. "1,800 NOK", "155 EUR", "300 - 2,000 MOP", "75-150 CHF"
func validatePriceWithCurrencyCode(text string) string {
	if match := priceCodeRegex.FindString(text); match != "" {
		return match
	}
	return ""
}

// validatePriceRange checks for numeric price ranges.
// e.g. "155 - 380", "300 - 2,000", "50-75"
func validatePriceRange(text string) string {
	if priceRangeRegex.MatchString(text) {
		return text
	}
	return ""
}

// validateOverUnderPrice checks for "Over X" or "Under X" patterns.
// e.g. "Over 75 USD", "Under 200 SGD"
func validateOverUnderPrice(text string) string {
	if overUnderRegex.MatchString(text) {
		return text
	}
	return ""
}

// validateBetweenPrice checks for "Between X and Y [CURRENCY]" patterns.
// e.g. "Between 350 and 500 HKD", "Between 50 and 100 EUR"
func validateBetweenPrice(text string) string {
	if betweenRegex.MatchString(text) {
		return text
	}
	return ""
}

// validateToRangePrice checks for "X to Y [CURRENCY]" patterns.
// e.g. "500 to 1500 TWD", "25 to 50 GBP"
func validateToRangePrice(text string) string {
	if toRangeRegex.MatchString(text) {
		return text
	}
	return ""
}

// validateLessThanPrice checks for "Less than X [CURRENCY]" patterns.
// e.g. "Less than 200 THB", "Less than 50.5 EUR"
func validateLessThanPrice(text string) string {
	if lessThanRegex.MatchString(text) {
		return text
	}
	return ""
}

// normalizePriceText cleans and normalizes price text for validation by removing separators and extra whitespace.
// e.g. "$$$ · French cuisine", "€€€ • Modern European", "155 - 380"
func normalizePriceText(text string, separators string) string {
	candidate := strings.TrimSpace(text)

	// Normalize whitespace
	candidate = strings.TrimSpace(strings.Join(strings.Fields(candidate), " "))

	// Only consider text before separator characters
	if idx := strings.IndexAny(candidate, separators); idx != -1 {
		candidate = strings.TrimSpace(candidate[:idx])
	}

	return candidate
}
