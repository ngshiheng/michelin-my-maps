package parsers

import "github.com/nyaruka/phonenumbers"

// ParsePhoneNumber extracts and parses phone number from a raw string.
// Example inputPhoneNumber: "+81 3-3874-1552"
func ParsePhoneNumber(inputPhoneNumber string) string {
	parsedPhoneNumber, err := phonenumbers.Parse(inputPhoneNumber, "")
	if err != nil {
		return ""
	}

	return phonenumbers.Format(parsedPhoneNumber, phonenumbers.E164)
}
