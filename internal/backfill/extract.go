package backfill

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/extraction"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/storage"
)

// extractRestaurantAwardData parses the provided XMLElement and returns Michelin award data for a restaurant.
func (s *Scraper) extractRestaurantAwardData(e *colly.XMLElement) storage.RestaurantAwardData {
	data := storage.RestaurantAwardData{}

	data.PublishedDate = extractPublishedDate(e)

	// Try dLayer first (highest priority for newer pages; 2020+)
	if extractFromDLayer(e, &data) && data.PublishedDate != "" {
		return data
	}

	// Fallback to individual extractors
	if data.Distinction == "" {
		data.Distinction = extractDistinction(e)
	}

	if data.Price == "" {
		data.Price = extractPrice(e)
	}

	// Log debug info for empty or problematic extractions (similar to scraper pattern)
	if data.Distinction == "" || data.Price == "" {
		// Note: logrus import will be auto-removed if not used, so keeping this as a comment for now
		// Future enhancement: add structured logging similar to scraper
	}

	return data
}

// extractFromDLayer attempts to populate storage.RestaurantAwardData fields (Distinction, Price, GreenStar)
// from the dLayer script tag in the HTML document. Returns true if extraction was successful.
func extractFromDLayer(e *colly.XMLElement, data *storage.RestaurantAwardData) bool {
	scriptContent := findDLayerScript(e)
	if scriptContent == "" {
		return false
	}

	return populateDataFromDLayer(scriptContent, e, data)
}

// findDLayerScript searches for a dLayer script containing restaurant distinction data.
func findDLayerScript(e *colly.XMLElement) string {
	return findScript(e, func(text string) bool {
		return strings.Contains(text, dLayerDataMarker) && strings.Contains(text, dLayerDistinctionKey)
	})
}

// populateDataFromDLayer extracts and populates restaurant data from dLayer script content.
func populateDataFromDLayer(scriptContent string, e *colly.XMLElement, data *storage.RestaurantAwardData) bool {
	distinction := extraction.ParseDLayerValue(scriptContent, dLayerDistinctionKey)

	// Try extracting price from HTML first, fallback to dLayer
	price := extractPrice(e)
	if price == "" {
		price = extraction.ParseDLayerValue(scriptContent, "price")
	}

	greenStar := extraction.ParseDLayerValue(scriptContent, "greenstar")

	// Require both distinction and price for successful extraction
	if distinction == "" || price == "" {
		return false
	}

	data.Distinction = distinction
	data.Price = extraction.CleanPriceValue(price)
	data.GreenStar = extraction.ParseGreenStarValue(greenStar)

	return true
}

// extractDistinction returns the restaurant's distinction (e.g., Michelin Star, Bib Gourmand)
// from the XML element using known XPath selectors and parsing logic.
func extractDistinction(e *colly.XMLElement) string {
	texts := e.ChildTexts(awardDistinctionXPath)
	return extraction.ExtractTextFromElements(texts)
}

// extractPrice returns the restaurant's price information from the XML element.
// It checks known XPath selectors and validates price formats using regex patterns.
func extractPrice(e *colly.XMLElement) string {
	texts := e.ChildTexts(awardPriceXPath)
	for _, text := range texts {
		if price := extraction.ValidatePriceText(text); price != "" {
			return price
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

	// Try extracting from XPath text content
	if date := extractDateFromTexts(e); date != "" {
		return date
	}

	// Check meta description as fallback
	return extractDateFromMetaContent(e)
}

// extractDateFromTexts extracts date from XPath text content using predefined patterns.
func extractDateFromTexts(e *colly.XMLElement) string {
	texts := e.ChildTexts(awardDateXPath)
	for _, text := range texts {
		if date := extraction.ParseDateFromText(strings.TrimSpace(text)); date != "" {
			return date
		}
	}
	return ""
}

// extractDateFromMetaContent extracts date from meta description content.
func extractDateFromMetaContent(e *colly.XMLElement) string {
	metaContent := e.ChildAttr(awardDateMetaXPath, "content")
	if metaContent != "" {
		return extraction.ParseDateFromText(metaContent)
	}
	return ""
}

// extractDateFromJSONLD extracts the published date from JSON-LD script tags.
// Example JSON-LD format: {"@type": "Restaurant", "review": {"datePublished": "2021-01-25T05:32"}}
// Returns the datePublished value, e.g., "2021-01-25T05:32".
func extractDateFromJSONLD(e *colly.XMLElement) string {
	jsonLD := findJSONLDScript(e)
	if jsonLD == "" {
		return ""
	}

	return parseJSONLDDate(jsonLD)
}

// findJSONLDScript searches for a JSON-LD script containing restaurant data.
func findJSONLDScript(e *colly.XMLElement) string {
	return findScript(e, func(text string) bool {
		return strings.Contains(text, jsonLDRestaurantType)
	})
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

// parseJSONLDDate parses the datePublished field from JSON-LD content.
func parseJSONLDDate(jsonLD string) string {
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

// extractOriginalURL extracts the original URL from a Wayback Machine snapshot URL
// using the last occurrence of waybackIDMarker as the separator and returns the cleaned URL.
// Returns empty string if the URL is invalid or doesn't contain the required marker.
func extractOriginalURL(snapshotURL string) string {
	if snapshotURL == "" {
		return ""
	}

	i := strings.LastIndex(snapshotURL, waybackIDMarker)
	if i == -1 {
		return ""
	}

	rawURL := snapshotURL[i+len(waybackIDMarker):]
	return normalizeURL(rawURL)
}

// normalizeURL parses and normalizes a URL by converting scheme and host to lowercase,
// removing query parameters, and cleaning up the path.
func normalizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL // Return original if parsing fails
	}

	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.RawQuery = ""

	// Clean up path: remove trailing slash except for root path
	if u.Path != "/" {
		u.Path = strings.TrimSuffix(u.Path, "/")
	}

	return u.String()
}
