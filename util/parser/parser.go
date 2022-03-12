package parser

import (
	"log"
	"net/url"
	"strings"
)

// SplitN and unpack a string
func SplitUnpack(str string, separator string) (string, string) {
	parsedStr := strings.SplitN(str, separator, 2)

	for i, s := range parsedStr {
		parsedStr[i] = strings.TrimSpace(s)
	}

	return parsedStr[0], parsedStr[1]
}

// Trim whitespace character such as line breaks or double spaces
func TrimWhiteSpaces(str string) string {
	trimWhiteSpace := strings.NewReplacer("\n", "", "  ", "")
	return trimWhiteSpace.Replace(str)
}

// Extract longitude and latitude from Google Maps URL
func ExtractCoordinates(input_url string) (string, string) {
	url, err := url.Parse(input_url)
	if err != nil {
		log.Fatal(err)
	}

	queryParams := url.Query()
	coordinates := queryParams["q"]

	return SplitUnpack(coordinates[0], ",")
}
