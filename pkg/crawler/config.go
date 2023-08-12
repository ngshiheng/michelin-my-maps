package crawler

import "time"

// Crawler app settings
const (
	allowedDomain = "guide.michelin.com"
	cachePath     = "cache"
	delay         = 2 * time.Second
	randomDelay   = 2 * time.Second
	parallelism   = 5
)

// XPath
const (
	// All restaurants page
	// E.g.: https://guide.michelin.com/en/restaurants/3-stars-michelin/
	nextPageArrowButtonXPath = "//li[@class='arrow']/a[@class='btn btn-outline-secondary btn-sm']"
	restaurantXPath          = "//div[@class='card__menu box-placeholder js-restaurant__list_item js-match-height js-map ']"
	restaurantDetailUrlXPath = "//a[@class='link with-love']"
	restaurantLocationXPath  = "//div[@class='card__menu-footer--location flex-fill pl-text']"

	// Individual restaurant detail page
	// E.g.: https://guide.michelin.com/en/singapore-region/singapore/restaurant/les-amis
	restaurantDetailXPath                = "//div[@class='restaurant-details']"
	restaurantNameXPath                  = "//*[@class='restaurant-details__heading--title']"
	restaurantAddressXPath               = "//*[@class='restaurant-details__heading--address']"
	restaurantPriceAndCuisineXPath       = "//*[@class='restaurant-details__heading-price']"
	restaurantFacilitiesAndServicesXPath = "//div[@class='restaurant-details__services']//li"
	restaurantGoogleMapsXPath            = "//div[@class='google-map__static']/iframe"
	restaurantPhoneNumberXPath           = "//a[@data-event='CTA_tel']"
	restaurantWebsiteUrlXPath            = "//a[@data-event='CTA_website']"
)
