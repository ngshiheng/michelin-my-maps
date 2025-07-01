package backfill

import (
	"bytes"
	"encoding/json"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/parser"
)

// AwardData represents extracted Michelin award information for a restaurant.
type AwardData struct {
	Distinction   string
	Price         string
	GreenStar     bool
	PublishedDate string
}

// extractRestaurantAwardData parses the provided HTML and returns Michelin award data for a restaurant.
func extractRestaurantAwardData(html []byte) (*AwardData, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, err
	}

	data := &AwardData{}

	data.PublishedDate = extractPublishedDate(doc)

	// Try dLayer first (highest priority for newer pages; 2020+)
	if extractFromDLayer(doc, data) && data.PublishedDate != "" {
		return data, nil
	}

	// Fallback to individual extractors
	if data.Distinction == "" {
		data.Distinction = extractDistinction(doc)
	}

	if data.Price == "" {
		data.Price = extractPrice(doc)
	}

	return data, nil
}

/*
extractFromDLayer attempts to populate AwardData fields (Distinction, Price, GreenStar)
from the dLayer script tag in the HTML document. Returns true if extraction was successful.
*/
func extractFromDLayer(doc *goquery.Document, data *AwardData) bool {
	scriptContent := findScript(doc, func(text string) bool {
		return strings.Contains(text, "dLayer") && strings.Contains(text, "distinction")
	})

	if scriptContent == "" {
		return false
	}

	distinction := parser.ParseDLayerValue(scriptContent, "distinction")
	price := parser.ParseDLayerValue(scriptContent, "price")
	greenStar := parser.ParseDLayerValue(scriptContent, "greenstar")

	if distinction == "" && price == "" {
		return false
	}

	data.Distinction = distinction
	data.Price = strings.ReplaceAll(price, "\\u002c", ",") // Handle unicode escape for commas
	data.GreenStar = strings.EqualFold(greenStar, "True")

	return true
}

/*
extractDistinction returns the restaurant's distinction (e.g., Michelin Star, Bib Gourmand)
from the HTML document using known selectors and parsing logic.
*/
func extractDistinction(doc *goquery.Document) string {
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

/*
extractPrice returns the restaurant's price information from the HTML document
using known selectors and normalization logic.
*/
func extractPrice(doc *goquery.Document) string {
	selector := "li.restaurant-details__heading-price, li:has(span.mg-price), li:has(span.mg-euro-circle)"
	var result string
	doc.Find(selector).EachWithBreak(func(i int, s *goquery.Selection) bool {
		clone := s.Clone()
		clone.Find("span").Remove()
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

// extractPublishedDate returns the published date of the Michelin award from the HTML document.
// It first tries to extract the date from JSON-LD, then falls back to known text patterns.
func extractPublishedDate(doc *goquery.Document) string {
	// Try JSON-LD first (2021+ layout)
	if date := extractDateFromJSONLD(doc); date != "" {
		return date
	}

	datePatterns := []*regexp.Regexp{
		regexp.MustCompile(`(\d{4})\s+MICHELIN Guide`), // e.g. "2023 MICHELIN Guide"
		regexp.MustCompile(`MICHELIN Guide.*?(\d{4})`), // e.g. "MICHELIN Guide ... 2023"
		regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`),      // ISO date format
	}

	selector := "div.restaurant-details__heading--label-title, div.label-text, meta[name=\"description\"]"
	var result string
	doc.Find(selector).EachWithBreak(func(i int, s *goquery.Selection) bool {
		var text string
		if goquery.NodeName(s) == "meta" {
			content, exists := s.Attr("content")
			if !exists {
				return true
			}
			text = strings.TrimSpace(content)
		} else {
			text = strings.TrimSpace(s.Text())
		}

		for _, pattern := range datePatterns {
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
func extractDateFromJSONLD(doc *goquery.Document) string {
	jsonLD := findScript(doc, func(text string) bool {
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
func findScript(doc *goquery.Document, condition func(string) bool) string {
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
