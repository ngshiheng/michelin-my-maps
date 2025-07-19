package scraper

import (
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/parser"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/storage"
	log "github.com/sirupsen/logrus"
)

// extractRestaurantData extracts restaurant data from the XML element.
func (s *Scraper) extractRestaurantData(e *colly.XMLElement) storage.RestaurantData {
	url := e.Request.URL.String()
	websiteURL := e.ChildAttr(restaurantWebsiteURLXPath, "href")
	name := e.ChildText(restaurantNameXPath)

	address := e.ChildText(restaurantAddressXPath)
	address = strings.ReplaceAll(address, "\n", " ")

	description := e.ChildText(restaurantDescriptionXPath)
	distinction := e.ChildText(restaurantDistinctionXPath)
	greenStar := e.ChildText(restaurantGreenStarXPath)

	priceAndCuisine := e.ChildText(restaurantPriceAndCuisineXPath)
	price, cuisine := parser.SplitUnpack(priceAndCuisine, "·")

	phoneNumber := e.ChildAttr(restaurantPhoneNumberXPath, "href")
	formattedPhoneNumber := parser.ParsePhoneNumber(phoneNumber)
	if formattedPhoneNumber == "" {
		log.WithFields(log.Fields{
			"phone_number": phoneNumber,
			"url":          url,
		}).Debug("invalid phone number")
	}

	facilitiesAndServices := e.ChildTexts(restaurantFacilitiesAndServicesXPath)

	var year int
	if v := e.Request.Ctx.GetAny("publishedYear"); v != nil {
		if y, ok := v.(int); ok {
			year = y
		}
	}

	return storage.RestaurantData{
		URL:                   url,
		Year:                  year,
		Name:                  name,
		Address:               address,
		Location:              e.Request.Ctx.Get("location"),
		Latitude:              e.Request.Ctx.Get("latitude"),
		Longitude:             e.Request.Ctx.Get("longitude"),
		Cuisine:               cuisine,
		PhoneNumber:           formattedPhoneNumber,
		WebsiteURL:            websiteURL,
		Distinction:           parser.ParseDistinction(distinction),
		Description:           parser.TrimWhiteSpaces(description),
		Price:                 price,
		FacilitiesAndServices: strings.Join(facilitiesAndServices, ","),
		GreenStar:             parser.ParseGreenStar(greenStar),
	}
}
