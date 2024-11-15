package crawler

const (
	// All restaurants page.
	// E.g.: https://guide.michelin.com/en/restaurants/3-stars-michelin/
	nextPageArrowButtonXPath = "//li[@class='arrow']/a[@class='btn btn-outline-secondary btn-sm']"
	restaurantXPath          = "//div[contains(@class, 'js-restaurant__list_item')]"
	restaurantDetailUrlXPath = "//a[@class='link']"
	restaurantLocationXPath  = "//div[@class='card__menu-footer--score pl-text']"

	// Individual restaurant detail page.
	// E.g.: https://guide.michelin.com/en/singapore-region/singapore/restaurant/les-amis
	restaurantDetailXPath = "//*[@class='restaurant-details']"

	restaurantAddressXPath               = "//*[@class='data-sheet__block--text'][1]"
	restaurantDescriptionXPath           = "//*[@class='data-sheet__description']"
	restaurantDistinctionXPath           = "//*[@class='data-sheet__classification-item--content'][2]"
	restaurantFacilitiesAndServicesXPath = "//*[@class='restaurant-details__services']//li"
	restaurantGoogleMapsXPath            = "//*[@class='google-map__static']/iframe"
	restaurantNameXPath                  = "//*[@class='data-sheet__title']"
	restaurantPhoneNumberXPath           = "//a[@data-event='CTA_tel']"
	restaurantPriceAndCuisineXPath       = "//*[@class='data-sheet__block--text'][2]"
	restaurantWebsiteUrlXPath            = "//a[@data-event='CTA_website']"
)
