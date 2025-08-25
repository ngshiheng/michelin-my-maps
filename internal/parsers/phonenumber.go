package parsers

import "github.com/nyaruka/phonenumbers"

// ParsePhoneNumber extracts and parses phone number from a raw string.
// e.g. "+81 3-3874-1552"
func ParsePhoneNumber(text string) string {
	parsedPhoneNumber, err := phonenumbers.Parse(text, "")
	if err != nil {
		return ""
	}

	return phonenumbers.Format(parsedPhoneNumber, phonenumbers.E164)
}
