package backfill

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/models"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/parser"
)

/*
extractAwardDataFromHTML extracts distinction and price from dLayer,
and publishedDate from JSON-LD. Includes fallback logic.

Example HTML:

	<script>
	    dLayer['distinction'] = '3 Stars'
	    dLayer['price'] = '$$$'
	    dLayer['greenstar'] = 'True'
	</script>
	<script type="application/ld+json">
	    {"@type":"Restaurant","review":{"datePublished":"2023-07-01"}}
	</script>

Extraction result:

	distinction: "3 Stars"
	price: "$$$"
	greenstar: true
	publishedDate: "2023-07-01"
*/
func extractAwardDataFromHTML(html []byte) (distinction, price string, greenstar bool, publishedDate string, err error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return "", "", false, "", err
	}

	// Try dLayer extraction first
	if d, p, g := extractFromDLayer(doc); d != "" || p != "" {
		return d, strings.ReplaceAll(p, "\\u002c", ","), g, extractPublishedDate(doc), nil
	}

	// Try distinction helpers
	distinction = extractDistinction(doc)
	if distinction == "" || distinction == models.SelectedRestaurants {
		distinction = extractDistinctionFromMeta(doc)
	}

	price = extractPrice(doc)
	greenstar = extractGreenStar(doc)
	publishedDate = extractPublishedDate(doc)

	return distinction, price, greenstar, publishedDate, nil
}

// extractFromDLayer tries to extract distinction, price, and greenstar from dLayer script
func extractFromDLayer(doc *goquery.Document) (distinction, price string, greenstar bool) {
	var scriptContent string
	doc.Find("script").EachWithBreak(func(i int, s *goquery.Selection) bool {
		txt := s.Text()
		if strings.Contains(txt, "dLayer") && strings.Contains(txt, "distinction") {
			scriptContent = txt
			return false
		}
		return true
	})
	if scriptContent != "" {
		distinction = parser.ParseDLayerValue(scriptContent, "distinction")
		price = parser.ParseDLayerValue(scriptContent, "price")
		greenstarStr := parser.ParseDLayerValue(scriptContent, "greenstar")
		greenstar = strings.EqualFold(greenstarStr, "True")
	}
	return
}

// extractDistinction tries to extract distinction from the classification list
func extractDistinction(doc *goquery.Document) string {
	var starDistinction, bibDistinction, selectedDistinction string
	doc.Find("ul.restaurant-details__classification--list li").Each(func(i int, s *goquery.Selection) {
		liText := s.Text()
		parsed := parser.ParseDistinction(liText)
		switch parsed {
		case models.ThreeStars, models.TwoStars, models.OneStar:
			if starDistinction == "" {
				starDistinction = parsed
			}
		case models.BibGourmand:
			if bibDistinction == "" {
				bibDistinction = parsed
			}
		case models.SelectedRestaurants:
			if selectedDistinction == "" {
				selectedDistinction = parsed
			}
		}
	})
	if starDistinction != "" {
		return starDistinction
	}
	if bibDistinction != "" {
		return bibDistinction
	}
	if selectedDistinction != "" {
		return selectedDistinction
	}
	return ""
}

// extractDistinctionFromMeta tries to extract distinction from meta description tags
func extractDistinctionFromMeta(doc *goquery.Document) string {
	var distinction string
	doc.Find("meta[name='description'],meta[property='og:description']").EachWithBreak(func(i int, s *goquery.Selection) bool {
		content, exists := s.Attr("content")
		if !exists {
			return true
		}
		parsed := parser.ParseDistinction(content)
		if parsed != "" && parsed != models.SelectedRestaurants {
			distinction = parsed
			return true
		}
		return false
	})
	return distinction
}

// extractPrice tries to extract price from the price element
func extractPrice(doc *goquery.Document) string {
	var price string
	doc.Find("li.restaurant-details__heading-price").EachWithBreak(func(i int, s *goquery.Selection) bool {
		priceText := strings.TrimSpace(s.Text())
		if idx := strings.Index(priceText, "•"); idx != -1 {
			priceText = strings.TrimSpace(priceText[:idx])
		}
		priceText = strings.Join(strings.Fields(priceText), " ")
		price = priceText
		return false
	})
	return price
}

// extractGreenStar tries to extract greenstar from dLayer or other logic if needed
func extractGreenStar(doc *goquery.Document) bool {
	// This can be extended if greenstar is found elsewhere
	_, _, greenstar := extractFromDLayer(doc)
	return greenstar
}

// extractPublishedDate tries to extract publishedDate from JSON-LD or fallback
func extractPublishedDate(doc *goquery.Document) string {
	var publishedDate string
	var jsonLD string
	doc.Find("script[type='application/ld+json']").EachWithBreak(func(i int, s *goquery.Selection) bool {
		txt := s.Text()
		if strings.Contains(txt, "\"@type\":\"Restaurant\"") {
			jsonLD = txt
			return false
		}
		return true
	})
	if jsonLD != "" {
		var ld map[string]any
		if err := json.Unmarshal([]byte(jsonLD), &ld); err == nil {
			if review, ok := ld["review"].(map[string]any); ok {
				if pd, ok := review["datePublished"].(string); ok {
					publishedDate = pd
				}
			}
		}
	}
	if publishedDate != "" {
		return publishedDate
	}

	// Fallback to extracting from the restaurant details heading
	doc.Find("div.restaurant-details__heading--label-title").EachWithBreak(func(i int, s *goquery.Selection) bool {
		text := strings.TrimSpace(s.Text())
		re := regexp.MustCompile(`MICHELIN Guide.*?(\d{4})`)
		m := re.FindStringSubmatch(text)
		if len(m) == 2 {
			publishedDate = m[1]
			return false
		}
		return true
	})
	return publishedDate
}

func extractOriginalURL(snapshotURL string) string {
	const idMarker = "id_/"
	if pos := len(snapshotURL) - len(idMarker); pos >= 0 {
		if i := findLastIndex(snapshotURL, idMarker); i != -1 {
			return snapshotURL[i+len(idMarker):]
		}
	}
	return ""
}

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
