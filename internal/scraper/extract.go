package scraper

import (
	"github.com/gocolly/colly/v2"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/extraction"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/storage"
	log "github.com/sirupsen/logrus"
)

// extractData parses the provided XMLElement and returns Michelin restaurant data.
func (s *Scraper) extractData(e *colly.XMLElement) storage.RestaurantData {
	url := e.Request.URL.String()
	websiteURL := e.ChildAttr(restaurantWebsiteURLXPath, "href")
	name := e.ChildText(restaurantNameXPath)

	address := e.ChildText(restaurantAddressXPath)
	address = extraction.NormalizeAddress(address)

	description := e.ChildText(restaurantDescriptionXPath)
	distinction := e.ChildText(restaurantDistinctionXPath)
	greenStar := e.ChildText(restaurantGreenStarXPath)

	priceAndCuisine := e.ChildText(restaurantPriceAndCuisineXPath)
	price, cuisine := extraction.SplitUnpack(priceAndCuisine, "·")

	phoneNumber := e.ChildAttr(restaurantPhoneNumberXPath, "href")
	formattedPhoneNumber := extraction.ParsePhoneNumber(phoneNumber)
	if formattedPhoneNumber == "" {
		log.WithFields(log.Fields{
			"phone_number": phoneNumber,
			"url":          url,
		}).Debug("invalid phone number")
	}

	facilitiesAndServices := e.ChildTexts(restaurantFacilitiesAndServicesXPath)

	contextData := extraction.ExtractContextData(e.Request.Ctx)

	return storage.RestaurantData{
		URL:                   url,
		Year:                  contextData.Year,
		Name:                  name,
		Address:               address,
		Location:              contextData.Location,
		Latitude:              contextData.Latitude,
		Longitude:             contextData.Longitude,
		Cuisine:               cuisine,
		PhoneNumber:           formattedPhoneNumber,
		WebsiteURL:            websiteURL,
		Distinction:           extraction.ParseDistinction(distinction),
		Description:           extraction.TrimWhiteSpaces(description),
		Price:                 price,
		FacilitiesAndServices: extraction.JoinFacilities(facilitiesAndServices),
		GreenStar:             extraction.ParseGreenStarValue(greenStar),
	}
}
