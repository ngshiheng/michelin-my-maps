package parsers

import (
	"strings"

	"github.com/gocolly/colly/v2"
)

// ExtractedData contains all possible data that can be extracted from a restaurant page
type ExtractedData struct {
	Address               string
	Cuisine               string
	Description           string
	Distinction           string
	FacilitiesAndServices string
	GreenStar             bool
	Latitude              string
	Location              string
	Longitude             string
	Name                  string
	PhoneNumber           string
	Price                 string
	URL                   string
	WaybackURL            string
	WebsiteURL            string
	Year                  int
}

// Parse is the unified extraction function that works for both scraper and backfill modes
func Parse(e *colly.XMLElement) *ExtractedData {
	currentURL := e.Request.URL.String()
	url, waybackURL := currentURL, ""
	if strings.Contains(currentURL, "web.archive.org") {
		waybackURL = currentURL
		url = extractOriginalURL(currentURL)
	}

	data := &ExtractedData{
		URL:        url,
		WaybackURL: waybackURL,
	}

	address := tryRestaurantSelectors(e, "address", NormalizeAddress)
	data.Description = tryRestaurantSelectors(e, "description", TrimWhiteSpaces)
	data.Name = tryRestaurantSelectors(e, "name", TrimWhiteSpaces)
	data.WebsiteURL = tryRestaurantSelectorsAttr(e, "websiteURL", "href")
	data.Address = address

	data.Distinction, data.GreenStar = ExtractDistinction(e)
	data.Latitude, data.Longitude = ExtractCoordinates(e)
	data.PhoneNumber = ExtractPhoneNumber(e)
	data.Price = ExtractPrice(e)
	data.Year = ExtractPublishedYear(e)

	delimiters := []string{"·", "•", "-", "|", "–", "—"}
	priceAndCuisine := tryRestaurantSelectors(e, "priceAndCuisine", TrimWhiteSpaces)
	price, cuisine := SplitUnpackMultiDelimiter(priceAndCuisine, delimiters)
	if data.Price == "" {
		data.Price = price
	}
	data.Cuisine = cuisine

	facilities := tryRestaurantSelectorsMultiple(e, "facilitiesAndServices")
	data.FacilitiesAndServices = JoinFacilities(facilities)
	data.Location = ParseLocationFromAddress(address)

	return data
}
