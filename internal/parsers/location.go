package parsers

import (
	"strings"
)

// ParseLocationFromAddress extracts location information from address string.
// Returns "city, country" for typical addresses, with hardcoded handling for Hong Kong, Singapore, Dubai.
func ParseLocationFromAddress(address string) string {
	addr := strings.ToLower(address)
	switch {
	case strings.Contains(addr, "hong kong"):
		return "Hong Kong, Hong Kong SAR China"
	case strings.Contains(addr, "luxembourg"):
		return "Luxembourg"
	case strings.Contains(addr, "singapore"):
		return "Singapore"
	case strings.Contains(addr, "dubai"):
		return "Dubai"
	case strings.Contains(addr, "macau"):
		return "Macau"
	}

	parts := strings.Split(address, ",")
	if len(parts) >= 4 {
		// e.g. "Lieu-dit la Baquère, Préneron, 32190, France"
		city := strings.TrimSpace(parts[len(parts)-3])
		country := strings.TrimSpace(parts[len(parts)-1])
		return city + ", " + country
	}

	if len(parts) == 3 {
		// e.g. "City, State, Country"
		city := strings.TrimSpace(parts[0])
		country := strings.TrimSpace(parts[2])
		return city + ", " + country
	}

	if len(parts) == 2 {
		// e.g. "Street, Country"
		country := strings.TrimSpace(parts[1])
		return country
	}

	if len(parts) == 1 {
		city := strings.TrimSpace(parts[0])
		return city
	}

	return ""
}
