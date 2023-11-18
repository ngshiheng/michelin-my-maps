package crawler

const (
	// All restaurants page.
	// E.g.: https://guide.michelin.com/en/restaurants/3-stars-michelin/
	nextPageArrowButtonXPath = "//li[@class='arrow']/a[@class='btn btn-outline-secondary btn-sm']"
	restaurantXPath          = "//div[@class='card__menu box-placeholder js-restaurant__list_item js-match-height js-map ']"
	restaurantDetailUrlXPath = "//a[@class='link']"
	restaurantLocationXPath  = "//div[@class='card__menu-footer--location flex-fill pl-text']"

	// Individual restaurant detail page.
	// E.g.: https://guide.michelin.com/en/singapore-region/singapore/restaurant/les-amis
	restaurantDetailXPath = "//*[@class='restaurant-details']"

	restaurantAddressXPath               = "//*[@class='restaurant-details__heading--address']"
	restaurantDescriptionXPath           = "//*[@class='restaurant-details__description--text ']"
	restaurantDistinctionXPath           = "//*[@class='restaurant-details__classification--list']//*[contains(@class, 'classification-item')]//*[contains(@class, 'classfication-content') and not(descendant::img) and not(descendant::span)]"
	restaurantFacilitiesAndServicesXPath = "//*[@class='restaurant-details__services']//li"
	restaurantGoogleMapsXPath            = "//*[@class='google-map__static']/iframe"
	restaurantNameXPath                  = "//*[@class='restaurant-details__heading--title']"
	restaurantPhoneNumberXPath           = "//a[@data-event='CTA_tel']"
	restaurantPriceAndCuisineXPath       = "//*[@class='restaurant-details__heading-price']"
	restaurantWebsiteUrlXPath            = "//a[@data-event='CTA_website']"
)
