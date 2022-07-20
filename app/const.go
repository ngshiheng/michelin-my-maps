package app

import "time"

type startUrl struct {
	Award string
	Url   string
}

// Start URLs
var urls = []startUrl{
	{"3 MICHELIN Stars", "https://guide.michelin.com/en/restaurants/3-stars-michelin/"},
	{"2 MICHELIN Stars", "https://guide.michelin.com/en/restaurants/2-stars-michelin/"},
	{"1 MICHELIN Star", "https://guide.michelin.com/en/restaurants/1-star-michelin/"},
	{"Bib Gourmand", "https://guide.michelin.com/en/restaurants/bib-gourmand"},
}

// Crawler app settings
const (
	allowedDomain  = "guide.michelin.com"
	outputFileName = "michelin_my_maps.csv"
	outputPath     = "data"
	cachePath      = "cache"
	delay          = 2 * time.Second
	randomDelay    = 2 * time.Second
	parallelism    = 5
)

// XPath
const (
	// All restaurants page
	// E.g.: https://guide.michelin.com/en/restaurants/3-stars-michelin/
	nextPageArrowButtonXPath = "//a[@class='btn btn-outline-secondary btn-sm']"
	restaurantXPath          = "//div[@class='card__menu box-placeholder js-restaurant__list_item js-match-height js-map ']"
	restaurantDetailUrlXPath = "//a[@class='link']"
	restaurantLocationXPath  = "//div[@class='card__menu-footer--location flex-fill pl-text']/i/following-sibling::text()"

	// Individual restaurant detail page
	// E.g.: https://guide.michelin.com/en/singapore-region/singapore/restaurant/les-amis
	restaurantDetailXPath                = "//div[@class='restaurant-details']"
	restaurantNameXPath                  = "//h2[@class='restaurant-details__heading--title']"
	restaurantAddressXPath               = "//ul[@class='restaurant-details__heading--list']/li"
	restaurantPriceAndCuisineXPath       = "//li[@class='restaurant-details__heading-price']"
	restaurantFacilitiesAndServicesXPath = "//div[@class='restaurant-details__services--content']/i/following-sibling::text()"
	restaurantGoogleMapsXPath            = "//div[@class='google-map__static']/iframe"
	restaurantPhoneNumberXPath           = "//a[@data-event='CTA_tel']"
	restaurantWebsiteUrlXPath            = "//a[@data-event='CTA_website']"
)
