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
	var (
		url        = ""
		waybackURL = ""
	)

	currentURL := e.Request.URL.String()

	if strings.Contains(currentURL, "web.archive.org") {
		waybackURL = currentURL
		url = extractOriginalURL(currentURL)
	} else {
		url = currentURL
	}

	data := &ExtractedData{
		URL:        url,
		WaybackURL: waybackURL,
	}

	data.Distinction, data.GreenStar = ExtractDistinction(e)

	data.Price = ExtractPrice(e)
	data.Year = ExtractPublishedYear(e)

	data.Name = tryRestaurantSelectors(e, "name", TrimWhiteSpaces)
	data.Description = tryRestaurantSelectors(e, "description", TrimWhiteSpaces)

	address := tryRestaurantSelectors(e, "address", NormalizeAddress)
	data.Address = address

	delimiters := []string{"·", "•", "-", "|", "–", "—"}
	priceAndCuisine := tryRestaurantSelectors(e, "priceAndCuisine", TrimWhiteSpaces)
	price, cuisine := SplitUnpackMultiDelimiter(priceAndCuisine, delimiters)
	if data.Price == "" {
		data.Price = price
	}
	data.Cuisine = cuisine

	phoneNumber := tryRestaurantSelectorsAttr(e, "phoneNumber", "href")
	data.PhoneNumber = ExtractPhoneNumber(phoneNumber)

	data.WebsiteURL = tryRestaurantSelectorsAttr(e, "websiteURL", "href")

	facilities := tryRestaurantSelectorsMultiple(e, "facilitiesAndServices")
	data.FacilitiesAndServices = JoinFacilities(facilities)

	data.Latitude, data.Longitude = ExtractCoordinates(e)
	data.Location = ParseLocationFromAddress(address)

	return data
}
