package app

import "time"

// Start URLs
var urls = []string{
	"https://guide.michelin.com/en/restaurants/3-stars-michelin/",
	"https://guide.michelin.com/en/restaurants/2-stars-michelin/",
	"https://guide.michelin.com/en/restaurants/1-star-michelin/",
	"https://guide.michelin.com/en/restaurants/bib-gourmand",
}

// Crawler app settings
const (
	allowedDomain  = "guide.michelin.com"
	outputFileName = "michelin_my_maps.csv"
	outputPath     = "generated"
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
	restaurantDetailXPath          = "//div[@class='restaurant-details']"
	restaurantNameXPath            = "//h2[@class='restaurant-details__heading--title']"
	restaurantAddressXPath         = "//ul[@class='restaurant-details__heading--list']/li"
	restaurantpriceAndCuisineXPath = "//li[@class='restaurant-details__heading-price']"
	restarauntGoogleMapsXPath      = "//div[@class='google-map__static']/iframe"
	restarauntPhoneNumberXPath     = "//span[@class='flex-fill']"
	restarauntWebsiteUrlXPath      = "//div[@class='collapse__block-item link-item']/a"
)
