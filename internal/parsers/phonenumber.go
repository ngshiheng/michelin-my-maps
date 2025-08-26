package parsers

import (
	"github.com/gocolly/colly/v2"
	"github.com/nyaruka/phonenumbers"
)

// ExtractPhoneNumber parses and normalizes a phone number from a raw string.
// e.g. "+81 3-3874-1552"
func ExtractPhoneNumber(e *colly.XMLElement) string {
	phoneNumber := tryRestaurantSelectorsAttr(e, "phoneNumber", "href")
	parsedPhoneNumber, err := phonenumbers.Parse(phoneNumber, "")
	if err != nil {
		return ""
	}
	return phonenumbers.Format(parsedPhoneNumber, phonenumbers.E164)
}
