package parsers

import (
	"github.com/gocolly/colly/v2"
	"github.com/nyaruka/phonenumbers"
)

// ExtractPhoneNumber parses and normalizes a phone number from a raw string.
// e.g. "+81 3-3874-1552"
func ExtractPhoneNumber(e *colly.XMLElement) string {
	rawPhoneNumber := tryRestaurantSelectorsAttr(e, "phoneNumber", "href")
	return parsePhoneNumber(rawPhoneNumber)

}

func parsePhoneNumber(text string) string {
	parsedPhoneNumber, err := phonenumbers.Parse(text, "")
	if err != nil {
		return ""
	}
	return phonenumbers.Format(parsedPhoneNumber, phonenumbers.E164)
}
