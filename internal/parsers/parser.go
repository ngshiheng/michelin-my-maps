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
	url, waybackURL := parseRequestURL(e.Request.URL.String())
	data := seedExtractedData(findAndParseJSONLD(e))
	data.URL = url
	data.WaybackURL = waybackURL

	address := firstNonEmpty(data.Address, tryRestaurantSelectors(e, "address", NormalizeAddress))
	distinction, greenStar := ExtractDistinction(e)
	price, cuisine := splitPriceAndCuisine(e)

	data.Address = address
	data.Description = firstNonEmpty(tryRestaurantSelectors(e, "description", TrimWhiteSpaces), data.Description)
	data.Name = firstNonEmpty(data.Name, tryRestaurantSelectors(e, "name", TrimWhiteSpaces))
	data.WebsiteURL = tryRestaurantSelectorsAttr(e, "websiteURL", "href")
	data.Distinction = firstNonEmpty(data.Distinction, distinction)
	data.GreenStar = greenStar
	data.PhoneNumber = firstNonEmpty(data.PhoneNumber, ExtractPhoneNumber(e))
	data.Price = firstNonEmpty(ExtractPrice(e), price)
	data.Cuisine = firstNonEmpty(data.Cuisine, cuisine)
	data.Year = firstNonZero(data.Year, ExtractPublishedYear(e))
	data.FacilitiesAndServices = JoinFacilities(tryRestaurantSelectorsMultiple(e, "facilitiesAndServices"))
	data.Location = firstNonEmpty(data.Location, ParseLocationFromAddress(address))

	if data.Latitude == "" || data.Longitude == "" {
		data.Latitude, data.Longitude = ExtractCoordinates(e)
	}

	return data
}

func parseRequestURL(currentURL string) (url, waybackURL string) {
	url = currentURL
	if strings.Contains(currentURL, "web.archive.org") {
		waybackURL = currentURL
		url = extractOriginalURL(currentURL)
	}
	return url, waybackURL
}

func seedExtractedData(ld *jsonLDRestaurant) *ExtractedData {
	data := &ExtractedData{}
	if ld == nil {
		return data
	}

	data.Address = NormalizeAddress(ld.addressText())
	data.Description = ld.descriptionText()
	data.Name = TrimWhiteSpaces(ld.Name)
	data.PhoneNumber = parsePhoneNumber(ld.Telephone)
	data.Cuisine = TrimWhiteSpaces(ld.ServesCuisine)
	data.Location = TrimWhiteSpaces(ld.locationText())
	data.Latitude, data.Longitude = ld.coordinates()
	data.Distinction = ld.distinctionText()
	data.Year = ld.publishedYear()
	return data
}

func splitPriceAndCuisine(e *colly.XMLElement) (price, cuisine string) {
	delimiters := []string{"·", "•", "-", "|", "–", "—"}
	priceAndCuisine := tryRestaurantSelectors(e, "priceAndCuisine", TrimWhiteSpaces)
	return SplitUnpackMultiDelimiter(priceAndCuisine, delimiters)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}
