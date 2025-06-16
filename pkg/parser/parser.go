package parser

import (
	"strings"

	"github.com/ngshiheng/michelin-my-maps/v2/pkg/michelin"
	"github.com/nyaruka/phonenumbers"
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
