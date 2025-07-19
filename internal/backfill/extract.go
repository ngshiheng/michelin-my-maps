package backfill

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/parser"
)

var (
	currencyRegex   = regexp.MustCompile(`^[€$£¥₩₽₹฿₺]+$`)
	priceCodeRegex  = regexp.MustCompile(`^[0-9][0-9,.\-\s]*[0-9]\s*[A-Z]{2,4}$`)
	priceRangeRegex = regexp.MustCompile(`^[0-9][0-9,.\-\s]*[0-9]$`)
	overUnderRegex  = regexp.MustCompile(`^(Over|Under)\s+\d+`)
	betweenRegex    = regexp.MustCompile(`^Between\s+\d+.*\d+\s+[A-Z]{2,4}$`)
	toRangeRegex    = regexp.MustCompile(`^\d+\s+to\s+\d+\s+[A-Z]{2,4}$`)
	lessThanRegex   = regexp.MustCompile(`(?i)^Less than \d+(\.\d+)?\s*[A-Z]{2,4}$`)
)

// AwardData represents extracted Michelin award information for a restaurant.
type AwardData struct {
	Distinction   string
	Price         string
	GreenStar     bool
	PublishedDate string
}

// extractRestaurantAwardData parses the provided XMLElement and returns Michelin award data for a restaurant.
func extractRestaurantAwardData(e *colly.XMLElement) (*AwardData, error) {
	data := &AwardData{}

	data.PublishedDate = extractPublishedDate(e)

	// Try dLayer first (highest priority for newer pages; 2020+)
	if extractFromDLayer(e, data) && data.PublishedDate != "" {
		return data, nil
	}

	// Fallback to individual extractors
	if data.Distinction == "" {
		data.Distinction = extractDistinction(e)
	}

	if data.Price == "" {
		data.Price = extractPrice(e)
	}

	return data, nil
}

/*
extractFromDLayer attempts to populate AwardData fields (Distinction, Price, GreenStar)
from the dLayer script tag in the HTML document. Returns true if extraction was successful.
*/
func extractFromDLayer(e *colly.XMLElement, data *AwardData) bool {
	scriptContent := findScript(e, func(text string) bool {
		return strings.Contains(text, "dLayer") && strings.Contains(text, "distinction")
	})

	if scriptContent == "" {
		return false
	}

	distinction := parser.ParseDLayerValue(scriptContent, "distinction")
	price := extractPrice(e)
	if price == "" {
		price = parser.ParseDLayerValue(scriptContent, "price")
	}
	greenStar := parser.ParseDLayerValue(scriptContent, "greenstar")

	if distinction == "" || price == "" {
		return false
	}

	data.Distinction = distinction
	data.Price = strings.ReplaceAll(price, `\"`, `"`) // Handle unicode escape for commas
	data.GreenStar = strings.EqualFold(greenStar, "True")

	return true
}

/*
extractDistinction returns the restaurant's distinction (e.g., Michelin Star, Bib Gourmand)
from the XML element using known XPath selectors and parsing logic.
*/
func extractDistinction(e *colly.XMLElement) string {
	xpaths := []string{
		awardDistinctionXPath1,
		awardDistinctionXPath2,
	}

	for _, xpath := range xpaths {
		texts := e.ChildTexts(xpath)
		for _, text := range texts {
			text = strings.TrimSpace(text)
			if text != "" {
				return text
			}
		}
	}
	return ""
}

/*
extractPrice returns the restaurant's price information from the XML element.
It checks known XPath selectors and also handles price info in service rows (e.g., "Over 75 USD").
*/
func extractPrice(e *colly.XMLElement) string {
	xpaths := []string{
		awardPriceXPath1,
		awardPriceXPath2,
		awardPriceXPath3,
	}

	for _, xpath := range xpaths {
		texts := e.ChildTexts(xpath)
		for _, text := range texts {
			candidate := strings.TrimSpace(text)

			// Check if there's a span within this element and extract its text
			spanTexts := e.ChildTexts(xpath + awardPriceSpanXPath)
			if len(spanTexts) > 0 && strings.TrimSpace(spanTexts[0]) != "" {
				candidate = strings.TrimSpace(spanTexts[0])
			}

			// Only consider text before "·" or "•"
			if idx := strings.IndexAny(candidate, "·•"); idx != -1 {
				candidate = strings.TrimSpace(candidate[:idx])
			}

			// Normalize whitespace (collapse multiple spaces, trim)
			candidate = strings.TrimSpace(strings.Join(strings.Fields(candidate), " "))

			// Accept if only currency symbols (e.g., "$$$$", "€€€€")
			if currencyRegex.MatchString(candidate) {
				return candidate
			}
			// Accept if price + currency code (e.g., "1,800 NOK", "155 EUR", "300 - 2,000 MOP")
			if m := priceCodeRegex.FindString(candidate); m != "" {
				return m
			}
			// Accept if price range or number (e.g., "155 - 380", "300 - 2,000")
			if priceRangeRegex.MatchString(candidate) {
				return candidate
			}
			// Accept if "Over X" or "Under X" (e.g., "Over 75 USD")
			if overUnderRegex.MatchString(candidate) {
				return candidate
			}
			// Accept if "Between X and Y [CURRENCY]" (e.g., "Between 350 and 500 HKD")
			if betweenRegex.MatchString(candidate) {
				return candidate
			}
			// Accept if "X to Y [CURRENCY]" (e.g., "500 to 1500 TWD")
			if toRangeRegex.MatchString(candidate) {
				return candidate
			}
			// Accept if "Less than X [CURRENCY]" (e.g., "Less than 200 THB")
			if lessThanRegex.MatchString(candidate) {
				return candidate
			}
		}
	}

	return ""
}

// extractPublishedDate returns the published date of the Michelin award from the XML element.
// It first tries to extract the date from JSON-LD, then falls back to known text patterns.
func extractPublishedDate(e *colly.XMLElement) string {
	// Try JSON-LD first (2021+ layout)
	if date := extractDateFromJSONLD(e); date != "" {
		return date
	}

	datePatterns := []*regexp.Regexp{
		regexp.MustCompile(`(\d{4})\s+MICHELIN Guide`), // e.g. "2023 MICHELIN Guide"
		regexp.MustCompile(`MICHELIN Guide.*?(\d{4})`), // e.g. "MICHELIN Guide ... 2023"
		regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`),      // ISO date format
	}

	xpaths := []string{
		awardDateXPath1,
		awardDateXPath2,
	}

	// Check date XPaths
	for _, xpath := range xpaths {
		texts := e.ChildTexts(xpath)
		for _, text := range texts {
			text = strings.TrimSpace(text)
			for _, pattern := range datePatterns {
				if matches := pattern.FindStringSubmatch(text); len(matches) > 1 {
					return matches[1]
				}
			}
		}
	}

	// Check meta description
	metaContent := e.ChildAttr(awardDateMetaXPath, "content")
	if metaContent != "" {
		for _, pattern := range datePatterns {
			if matches := pattern.FindStringSubmatch(metaContent); len(matches) > 1 {
				return matches[1]
			}
		}
	}

	return ""
}

/*
extractDateFromJSONLD extracts the published date from JSON-LD script tags.

Example:

<script type="application/ld+json">

	{
	  "@type": "Restaurant",
	  "review": {
	    "datePublished": "2021-01-25T05:32"
	  }
	}

</script>

The function will extract and return "2021-01-25T05:32".
*/
func extractDateFromJSONLD(e *colly.XMLElement) string {
	jsonLD := findScript(e, func(text string) bool {
		return strings.Contains(text, `"@type":"Restaurant"`)
	})

	if jsonLD == "" {
		return ""
	}

	var ld map[string]any
	if err := json.Unmarshal([]byte(jsonLD), &ld); err != nil {
		return ""
	}

	if review, ok := ld["review"].(map[string]any); ok {
		if date, ok := review["datePublished"].(string); ok {
			return date
		}
	}
	return ""
}

// findScript searches for a <script> tag whose content matches the given condition.
func findScript(e *colly.XMLElement, condition func(string) bool) string {
	scripts := e.ChildTexts(awardScriptXPath)
	for _, script := range scripts {
		if condition(script) {
			return script
		}
	}
	return ""
}

// extractOriginalURL extracts the original URL from a Wayback Machine snapshot URL
// using the last occurrence of "id_/" as the marker and returns the cleaned URL.
func extractOriginalURL(snapshotURL string) string {
	const idMarker = "id_/"
	i := strings.LastIndex(snapshotURL, idMarker)
	if i == -1 {
		return ""
	}

	raw := snapshotURL[i+len(idMarker):]
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.RawQuery = ""

	if u.Path != "/" {
		u.Path = strings.TrimSuffix(u.Path, "/")
	}
	return u.String()
}
