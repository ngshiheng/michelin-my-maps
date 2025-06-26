package backfill

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/parser"
)

// MichelinExtractor handles extraction of Michelin award data from HTML documents,
// supporting multiple layouts and fallbacks for different Wayback Machine snapshots.
type MichelinExtractor struct {
	// Regex patterns for date extraction
	datePatterns []*regexp.Regexp
}

// AwardData represents extracted Michelin award information for a restaurant.
type AwardData struct {
	Distinction   string
	Price         string
	GreenStar     bool
	PublishedDate string
}

// NewMichelinExtractor creates and returns a new MichelinExtractor instance.
func NewMichelinExtractor() *MichelinExtractor {
	return &MichelinExtractor{
		datePatterns: []*regexp.Regexp{
			regexp.MustCompile(`MICHELIN Guide.*?(\d{4})`),
			regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`), // ISO date format
		},
	}
}

// Extract parses the provided HTML and extracts Michelin award data.
// Returns an AwardData struct or an error if parsing fails.
func (e *MichelinExtractor) Extract(html []byte) (*AwardData, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, err
	}

	data := &AwardData{}

	data.PublishedDate = e.extractPublishedDate(doc)

	// Try dLayer first (highest priority for newer pages; 2020+)
	if e.extractFromDLayer(doc, data) && data.PublishedDate != "" {
		return data, nil
	}

	// Fallback to individual extractors
	data.Distinction = e.extractDistinction(doc)
	data.Price = e.extractPrice(doc)

	return data, nil
}

// extractFromDLayer attempts to extract distinction, price, and green star status from the dLayer script tag.
func (e *MichelinExtractor) extractFromDLayer(doc *goquery.Document, data *AwardData) bool {
	scriptContent := e.findScript(doc, func(text string) bool {
		return strings.Contains(text, "dLayer") && strings.Contains(text, "distinction")
	})

	if scriptContent == "" {
		return false
	}

	distinction := parser.ParseDLayerValue(scriptContent, "distinction")
	price := parser.ParseDLayerValue(scriptContent, "price")
	greenStar := parser.ParseDLayerValue(scriptContent, "greenStar")

	if distinction == "" && price == "" {
		return false
	}

	data.Distinction = distinction
	data.Price = strings.ReplaceAll(price, "\\u002c", ",") // Handle unicode escape for commas
	data.GreenStar = strings.EqualFold(greenStar, "True")

	return true
}

// extractDistinction extracts the restaurant distinction from the document using a selector fallback pattern.
func (e *MichelinExtractor) extractDistinction(doc *goquery.Document) string {
	selector := "ul.restaurant-details__classification--list li, div.restaurant__classification p.flex-fill"
	var result string
	doc.Find(selector).EachWithBreak(func(i int, s *goquery.Selection) bool {
		text := strings.TrimSpace(s.Text())
		if parsed := parser.ParseDistinction(text); parsed != "" {
			result = parsed
			return false
		}
		return true
	})
	return result
}

// extractPrice extracts the price information from the document using a selector fallback pattern.
func (e *MichelinExtractor) extractPrice(doc *goquery.Document) string {
	selector := "li.restaurant-details__heading-price, li:has(span.mg-price)"
	var result string
	doc.Find(selector).EachWithBreak(func(i int, s *goquery.Selection) bool {
		clone := s.Clone()
		clone.Find("span.mg-price").Remove()
		text := strings.TrimSpace(clone.Text())
		if idx := strings.Index(text, "•"); idx != -1 {
			text = strings.TrimSpace(text[:idx])
		}
		normalized := strings.Join(strings.Fields(text), " ")
		if normalized != "" {
			result = normalized
			return false
		}
		return true
	})
	return result
}

/*
extractPublishedDate extracts the published date from the document.
Supports both legacy and modern layouts by using a unified selector.
*/
func (e *MichelinExtractor) extractPublishedDate(doc *goquery.Document) string {
	// Try JSON-LD first (2021+ layout)
	if date := e.extractDateFromJSONLD(doc); date != "" {
		return date
	}

	selector := "div.restaurant-details__heading--label-title, div.label-text"
	var result string
	doc.Find(selector).EachWithBreak(func(i int, s *goquery.Selection) bool {
		text := strings.TrimSpace(s.Text())
		for _, pattern := range e.datePatterns {
			if matches := pattern.FindStringSubmatch(text); len(matches) > 1 {
				result = matches[1]
				return false
			}
		}
		return true
	})
	return result
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
func (e *MichelinExtractor) extractDateFromJSONLD(doc *goquery.Document) string {
	jsonLD := e.findScript(doc, func(text string) bool {
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
func (e *MichelinExtractor) findScript(doc *goquery.Document, condition func(string) bool) string {
	var result string
	doc.Find("script").EachWithBreak(func(i int, s *goquery.Selection) bool {
		text := s.Text()
		if condition(text) {
			result = text
			return false
		}
		return true
	})
	return result
}

// extractOriginalURL extracts the original URL from a Wayback Machine snapshot URL.
func extractOriginalURL(snapshotURL string) string {
	const idMarker = "id_/"
	if pos := len(snapshotURL) - len(idMarker); pos >= 0 {
		if i := findLastIndex(snapshotURL, idMarker); i != -1 {
			return snapshotURL[i+len(idMarker):]
		}
	}
	return ""
}

// findLastIndex returns the last index of substr in s, or -1 if not found.
func findLastIndex(s, substr string) int {
	last := -1
	for i := 0; ; {
		j := i + len(substr)
		if j > len(s) {
			break
		}
		if s[i:j] == substr {
			last = i
		}
		i++
	}
	return last
}
