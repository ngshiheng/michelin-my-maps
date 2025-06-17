package scraper

import (
	"strings"
	"testing"

	"github.com/gocolly/colly/v2"
	"github.com/stretchr/testify/assert"
)

// TestXPathSelectors validates that all XPath selectors work against real HTML
func TestXPathSelectors(t *testing.T) {
	tests := []struct {
		name         string
		htmlFile     string
		xpath        string
		expectCount  int
		expectValues []string // Expected values to be found
		expectAttr   string   // Attribute to check instead of text
		description  string
	}{
		// Detail page selectors
		{
			name:         "detail name selector",
			htmlFile:     "restaurant_detail.html",
			xpath:        restaurantNameXPath,
			expectCount:  2,
			expectValues: []string{"Les Amis"},
			description:  "Restaurant name appears in multiple places",
		},
		{
			name:         "detail address selector",
			htmlFile:     "restaurant_detail.html",
			xpath:        restaurantAddressXPath,
			expectCount:  1,
			expectValues: []string{"Shaw Centre, #01-16, 1 Scotts Road, 228208, Singapore"},
			description:  "Restaurant address should be found once",
		},
		{
			name:         "detail facilities selector",
			htmlFile:     "restaurant_detail.html",
			xpath:        restaurantFacilitiesAndServicesXPath,
			expectCount:  6,
			expectValues: []string{"Air conditioning", "Car park", "Interesting wine list"},
			description:  "Six facilities should be listed",
		},
		{
			name:         "detail phone selector",
			htmlFile:     "restaurant_detail.html",
			xpath:        restaurantPhoneNumberXPath,
			expectCount:  1,
			expectAttr:   "href",
			expectValues: []string{"tel:+65 6733 2225"},
			description:  "Phone number link should be found",
		},
		{
			name:         "detail website selector",
			htmlFile:     "restaurant_detail.html",
			xpath:        restaurantWebsiteUrlXPath,
			expectCount:  1,
			expectAttr:   "href",
			expectValues: []string{"https://www.lesamis.com.sg/"},
			description:  "Website link should be found",
		},
		{
			name:         "detail price and cuisine selector",
			htmlFile:     "restaurant_detail.html",
			xpath:        restaurantPriceAndCuisineXPath,
			expectCount:  1,
			expectValues: []string{"$$$$", "French"},
			description:  "Price and cuisine info should be found",
		},
		{
			name:         "detail description selector",
			htmlFile:     "restaurant_detail.html",
			xpath:        restaurantDescriptionXPath,
			expectCount:  1,
			expectValues: []string{"Haute cuisine is increasingly"},
			description:  "Restaurant description should be found",
		},
		{
			name:         "detail distinction selector",
			htmlFile:     "restaurant_detail.html",
			xpath:        restaurantDistinctionXPath,
			expectCount:  1,
			expectValues: []string{"Three Stars: Exceptional cuisine"},
			description:  "Distinction classification should be found",
		},
		{
			name:         "detail google maps selector",
			htmlFile:     "restaurant_detail.html",
			xpath:        restaurantGoogleMapsXPath,
			expectCount:  1,
			expectAttr:   "src",
			expectValues: []string{"google.com/maps"},
			description:  "Google Maps iframe should be found",
		},
		// List page selectors
		{
			name:         "list restaurant cards selector",
			htmlFile:     "restaurant_list.html",
			xpath:        restaurantXPath,
			expectCount:  24,
			expectValues: []string{}, // Too many to list, just check count
			description:  "24 restaurant cards should be found",
		},
		{
			name:         "list detail links selector",
			htmlFile:     "restaurant_list.html",
			xpath:        restaurantDetailUrlXPath,
			expectCount:  24,
			expectAttr:   "href",
			expectValues: []string{"restaurant"}, // Should contain 'restaurant' in URLs
			description:  "24 detail page links should be found",
		},
		{
			name:         "list locations selector",
			htmlFile:     "restaurant_list.html",
			xpath:        restaurantLocationXPath,
			expectCount:  24,
			expectValues: []string{}, // Locations vary, just check count
			description:  "24 restaurant locations should be found",
		},
		{
			name:         "list pagination selector",
			htmlFile:     "restaurant_list.html",
			xpath:        nextPageArrowButtonXPath,
			expectCount:  1, // Pagination element exists but may have no text content
			expectValues: []string{},
			description:  "Pagination link may or may not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			htmlContent := loadTestHTML(t, tt.htmlFile)
			server := createTestServer(htmlContent)
			defer server.Close()

			c := colly.NewCollector()
			var results []string

			c.OnXML(tt.xpath, func(e *colly.XMLElement) {
				if tt.expectAttr != "" {
					// For attributes like href, src, etc.
					attrValue := e.Attr(tt.expectAttr)
					results = append(results, attrValue)
				} else {
					// For text content
					results = append(results, e.Text)
				}
			})

			c.Visit(server.URL)

			// Verify count
			assert.Equal(t, tt.expectCount, len(results), tt.description)

			// Verify values if specified
			if len(tt.expectValues) > 0 && len(results) > 0 {
				for _, expectedValue := range tt.expectValues {
					if expectedValue != "" {
						// Check if any result contains the expected value
						found := false
						for _, result := range results {
							if assert.ObjectsAreEqualValues(result, expectedValue) ||
								(len(result) > 0 && len(expectedValue) > 0 &&
									(result == expectedValue ||
										strings.Contains(result, expectedValue))) {
								found = true
								break
							}
						}
						assert.True(t, found,
							"Expected value '%s' not found in results: %v",
							expectedValue, results)
					}
				}
			}

			// Log results for debugging
			if len(results) > 0 {
				if len(results) <= 3 {
					t.Logf("Extracted values: %v", results)
				} else {
					t.Logf("Extracted %d values, first 3: %v", len(results), results[:3])
				}
			}
		})
	}
}
