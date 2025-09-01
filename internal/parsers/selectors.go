package parsers

import (
	"github.com/gocolly/colly/v2"
)

// tryAwardSelectors tries each selector in the award selectors list until one returns a valid result
func tryAwardSelectors(e *colly.XMLElement, field string, parser func(string) string) string {
	selectors, exists := AwardSelectors[field]
	if !exists {
		return ""
	}

	for _, selector := range selectors {
		if result := e.ChildText(selector); result != "" {
			if parsed := parser(result); parsed != "" {
				return parsed
			}
		}
	}
	return ""
}

// tryRestaurantSelectors tries each selector in the restaurant selectors list until one returns a valid result
func tryRestaurantSelectors(e *colly.XMLElement, field string, parser func(string) string) string {
	selectors, exists := RestaurantSelectors[field]
	if !exists {
		return ""
	}

	for _, selector := range selectors {
		if result := e.ChildText(selector); result != "" {
			if parsed := parser(result); parsed != "" {
				return parsed
			}
		}
	}
	return ""
}

// tryRestaurantSelectorsAttr tries each selector to get an attribute value
func tryRestaurantSelectorsAttr(e *colly.XMLElement, field string, attr string) string {
	selectors, exists := RestaurantSelectors[field]
	if !exists {
		return ""
	}

	for _, selector := range selectors {
		if result := e.ChildAttr(selector, attr); result != "" {
			return result
		}
	}
	return ""
}

// tryRestaurantSelectorsMultiple tries each selector to get multiple text results
func tryRestaurantSelectorsMultiple(e *colly.XMLElement, field string) []string {
	selectors, exists := RestaurantSelectors[field]
	if !exists {
		return nil
	}

	for _, selector := range selectors {
		if results := e.ChildTexts(selector); len(results) > 0 {
			return results
		}
	}
	return nil
}

var RestaurantSelectors = map[string][]string{
	"googleMapDiv": {
		"//div[@id='map']",
	},
	"name": {
		"//*[@class='data-sheet__title']",
		"//*[@class='restaurant-details__heading--title']",
	},
	"description": {
		"//div[contains(@class,'data-sheet__description')]",
		"//*[contains(@class,'js-show-description-text')]",
		"//div[contains(@class,'restaurant-details__description--text ')]",
		"//div[@id='opinion']//div[contains(@class,'tab__content-paragraph')]/p", // 20190818190359
	},
	"address": {
		"//*[contains(@class,'data-sheet__block--text')][1]",
		"//*[contains(@class,'restaurant-details__heading--address')]",
		"//div[contains(@class,'collapse__block-title')]//span[contains(@class,'fa-map-marker-alt')]/following-sibling::span[contains(@class,'flex-fill')]", // 20190818190359
		"//li[*[contains(@class,'fa-map-marker-alt')]]/text()[normalize-space()]",                                                                           // 20220125203424, 20211127004727
	},
	"priceAndCuisine": {
		"//div[contains(@class,'data-sheet__block--text')][2]",
		"//div[contains(@class,'restaurant-details__heading--price')]",
		"//*[contains(@class,'restaurant-details__heading-price')]",
		"//li[span[contains(@class, 'jumbotron__card-detail--icon')]][last()]", // 20190818190359
	},
	"phoneNumber": {
		"//a[@data-event='CTA_tel']",
		"//a[contains(@href,'tel:')]",
	},
	"websiteURL": {
		"//a[@data-event='CTA_website']",
		"//a[contains(@class,'website')]",
	},
	"facilitiesAndServices": {
		"//div[contains(@class,'col col-12 col-lg-6')]//li",
		"//div[@class='restaurant-details__services']//div[@class='restaurant-details__services--content']/text()[normalize-space()]",
		"//div[@class='restaurant-details__services']//li",
	},
	"googleMaps": {
		"//div[@class='google-map__static']/iframe",
		"//iframe[contains(@src,'google.com/maps')]",
		"//iframe[contains(@src,'maps.google')]",
	},
}

var AwardSelectors = map[string][]string{
	"distinction": {
		"//div[@class='data-sheet__classification-item--content'][2]",
		"//ul[contains(@class,'restaurant-details__classification--list')]//li",                 // Legacy 2020-2023
		"//div[contains(@class,'restaurant__classification')]//p[contains(@class,'flex-fill')]", // Older fallback
		"//div[contains(@class,'classification')]",                                              // Generic fallback
	},
	"price": {
		"//div[contains(@class,'data-sheet__block--text')][2]",                     // Modern
		"//div[@class='col-lg-12']/p",                                              // ???
		"//*[contains(@class,'restaurant-details__heading-price')]",                // Legacy
		"//span[contains(@class,'mg-price') or contains(@class,'mg-euro-circle')]", // Price spans
		"//div[contains(@class,'data-sheet__block--text')]",                        // Generic block text
	},
	"greenStar": {
		"//div[contains(text(),'MICHELIN Green Star')]",
		"//span[contains(text(),'Green Star')]",
		"//div[contains(@class,'green-star')]",
	},
	"publishedDate": {
		"//script[@type='application/ld+json']",                              // JSON-LD (highest priority)
		"//div[contains(@class,'restaurant-details__heading--label-title')]", // Legacy date
		"//div[contains(@class,'label-text')]",                               // Older date
		"//meta[@name='description']",                                        // Meta fallback
	},
}
