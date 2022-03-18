package parser

import (
	"net/url"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

// SplitN and unpack a string
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

// Trim whitespace character such as line breaks or double spaces
func TrimWhiteSpaces(str string) string {
	trimWhiteSpace := strings.NewReplacer("\n", "", "  ", "")
	return trimWhiteSpace.Replace(str)
}

/*
Extract longitude and latitude from Google Maps URL

Example input_url "https://www.google.com/maps/embed/v1/place?key=AIzaSyDvEyVCVpGtn81z5NrMKgdehPsrO9sJiMw&q=45.1712728,10.3565788&language=en-US"
*/
func ExtractCoordinates(input_url string) (string, string) {
	url, err := url.Parse(input_url)
	if err != nil {
		log.WithFields(log.Fields{"input_url": input_url}).Fatal(err)
	}

	queryParams := url.Query()
	coordinates := queryParams["q"][0] // e.g. "45.4215425,11.8096633"

	if !(IsValidCoordinates(coordinates)) {
		log.WithFields(log.Fields{"coordinates": coordinates}).Warn("invalid coordinates")
		return "", ""
	}

	return SplitUnpack(coordinates, ",")
}

// Check if a string contains a valid longitude or latitude (reference: https://stackoverflow.com/a/18690202/10067850)
func IsValidCoordinates(coordinates string) bool {
	re := regexp.MustCompile(`^[-+]?([1-8]?\d(\.\d+)?|90(\.0+)?),\s*[-+]?(180(\.0+)?|((1[0-7]\d)|([1-9]?\d))(\.\d+)?)$`)
	return re.MatchString(coordinates)
}

/*
Extract minPrice, maxPrice, and currency from a price string

Example input_price "148-248 USD", "1,000-1,280 CNY"
*/
func ExtractPrice(input_price string) (string, string, string) {
	if input_price == "" {
		return "", "", ""
	}

	currencyRegex := regexp.MustCompile(`(.{3})\s*$`) // Match last 3 characters
	currency := currencyRegex.FindString(input_price)

	minPrice, maxPrice := SplitUnpack(input_price[:len(input_price)-3], "-")

	numberRegex := regexp.MustCompile(`^\d{1,3}(,\d{3})*(\.\d+)?$`) // Match 0,000 format
	minPrice = numberRegex.FindString(minPrice)
	maxPrice = numberRegex.FindString(maxPrice)

	if minPrice == "" {
		minPrice = maxPrice
	}

	return minPrice, maxPrice, currency
}
