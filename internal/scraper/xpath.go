package scraper

const (
	// All restaurants page.
	// E.g.: https://guide.michelin.com/en/restaurants/3-stars-michelin/
	nextPageArrowButtonXPath = "//li[@class='arrow']/a[@class='btn btn-outline-secondary btn-sm']"
	restaurantXPath          = "//div[contains(@class, 'card__menu selection-card')]"
	restaurantDetailUrlXPath = "//a[@class='link']"
	restaurantLocationXPath  = "//div[@class='card__menu-footer--score pl-text']"

	// Individual restaurant detail page.
	// E.g.: https://guide.michelin.com/en/singapore-region/singapore/restaurant/les-amis
	restaurantDetailXPath = "//main[@class]"

	restaurantAddressXPath               = "//*[contains(@class, 'data-sheet__block--text')][1]"
	restaurantAwardPublishedYearXPath    = "//script[@type='application/ld+json']"
	restaurantDescriptionXPath           = "//*[contains(@class, 'data-sheet__description')]"
	restaurantDistinctionXPath           = "//*[@class='data-sheet__classification-item--content'][2]"
	restaurantFacilitiesAndServicesXPath = "//*[contains(@class, 'col col-12 col-lg-6')]//li"
	restaurantGoogleMapsXPath            = "//*[@class='google-map__static']/iframe"
	restaurantGreenStarXPath             = "//*[contains(text(),'MICHELIN Green Star')]"
	restaurantNameXPath                  = "//*[@class='data-sheet__title']"
	restaurantPhoneNumberXPath           = "//a[@data-event='CTA_tel']"
	restaurantPriceAndCuisineXPath       = "//*[contains(@class, 'data-sheet__block--text')][2]"
	restaurantWebsiteUrlXPath            = "//a[@data-event='CTA_website']"
)
