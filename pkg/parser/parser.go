package parser

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/ngshiheng/michelin-my-maps/v2/pkg/michelin"
	"github.com/nyaruka/phonenumbers"
	log "github.com/sirupsen/logrus"
)

// SplitUnpack performs SplitN and unpacks a string.
func SplitUnpack(str string, separator string) (string, string) {
	if len(str) == 0 {
		return str, str
	}

	parsedStr := strings.SplitN(str, separator, 2)

	for i, s := range parsedStr {
		parsedStr[i] = strings.TrimSpace(s)
	}

	if len(parsedStr) == 1 {
		return "", parsedStr[0] // Always assume price is missing
	}

	return parsedStr[0], parsedStr[1]
}

// TrimWhiteSpaces trims whitespace character such as line breaks or double spaces.
func TrimWhiteSpaces(str string) string {
	trimWhiteSpace := strings.NewReplacer("\n", "", "  ", "")
	return trimWhiteSpace.Replace(str)
}

// ParseDistinction parses the Michelin distinction based on the input string.
func ParseDistinction(distinction string) string {
	switch strings.ToLower(distinction) {
	case "three stars: exceptional cuisine":
		return michelin.ThreeStars
	case "two stars: excellent cooking":
		return michelin.TwoStars
	case "one star: high quality cooking":
		return michelin.OneStar
	case "bib gourmand: good quality, good value cooking":
		return michelin.BibGourmand
	default:
		return michelin.SelectedRestaurants
	}
}

// ParseGreenStar parses the MICHELIN Green Star based on the input string.
func ParseGreenStar(distinction string) bool {
	return strings.ToLower(distinction) == "michelin green star"
}

/*
ParseCoordinates extracts and parses longitude and latitude from a Google Maps URL.

Example inputUrl: "https://www.google.com/maps/embed/v1/place?key=AIzaSyDvEyVCVpGtn81z5NrMKgdehPsrO9sJiMw&q=45.1712728,10.3565788&language=en-US"
*/
func ParseCoordinates(inputUrl string) (string, string) {
	url, err := url.Parse(inputUrl)
	if err != nil {
		log.WithFields(log.Fields{"inputUrl": inputUrl}).Fatal(err)
	}

	queryParams := url.Query()
	coordinates := queryParams["q"][0] // e.g. "45.4215425,11.8096633"

	if !(IsValidCoordinates(coordinates)) {
		log.WithFields(log.Fields{"coordinates": coordinates}).Warn("invalid coordinates")
		return "", ""
	}

	return SplitUnpack(coordinates, ",")
}

/*
IsValidCoordinates checks if a string contains a valid longitude or latitude.

Reference: https://stackoverflow.com/a/18690202/10067850
*/
func IsValidCoordinates(coordinates string) bool {
	longitudeAndLatitudeRegex := `^[-+]?([1-8]?\d(\.\d+)?|90(\.0+)?),\s*[-+]?(180(\.0+)?|((1[0-7]\d)|([1-9]?\d))(\.\d+)?)$`
	re := regexp.MustCompile(longitudeAndLatitudeRegex)
	return re.MatchString(coordinates)
}

/*
ParsePrice extracts and parses minPrice, maxPrice, and currency from a raw price string.

Example inputPrice: "148-248 USD", "1,000-1,280 CNY"
*/
func ParsePrice(inputPrice string) (string, string, string) {
	if inputPrice == "" {
		return "", "", ""
	}

	currencyRegex := regexp.MustCompile(`(.{3})\s*$`) // Match last 3 characters
	currency := currencyRegex.FindString(inputPrice)

	minPrice, maxPrice := SplitUnpack(inputPrice[:len(inputPrice)-3], "-")

	numberRegex := regexp.MustCompile(`^\d{1,3}(,\d{3})*(\.\d+)?$`) // Match 0,000 format
	minPrice = numberRegex.FindString(minPrice)
	maxPrice = numberRegex.FindString(maxPrice)

	if minPrice == "" {
		minPrice = maxPrice
	}

	return minPrice, maxPrice, currency
}

/*
ParsePhoneNumber extracts and parses phone number from a raw string.

Example inputPhoneNumber: "+81 3-3874-1552"
*/
func ParsePhoneNumber(inputPhoneNumber string) string {
	parsedPhoneNumber, err := phonenumbers.Parse(inputPhoneNumber, "")
	if err != nil {
		return ""
	}

	return phonenumbers.Format(parsedPhoneNumber, phonenumbers.E164)
}
