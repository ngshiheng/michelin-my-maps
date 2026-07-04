package parsers

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

const xPathJSONLDScript = "//script[@type='application/ld+json']"

type jsonLDRestaurant struct {
	Type                   any           `json:"@type"`
	Address                jsonLDAddress `json:"address"`
	Award                  jsonLDAward   `json:"award"`
	Brand                  string        `json:"brand"`
	CurrenciesAccepted     string        `json:"currenciesAccepted"`
	Description            string        `json:"description"`
	Geo                    jsonLDGeo     `json:"geo"`
	HasDriveThroughService any           `json:"hasDriveThroughService"`
	HasMap                 string        `json:"hasMap"`
	KnowsLanguage          string        `json:"knowsLanguage"`
	Latitude               any           `json:"latitude"`
	Longitude              any           `json:"longitude"`
	Name                   string        `json:"name"`
	PaymentAccepted        string        `json:"paymentAccepted"`
	PriceRange             string        `json:"priceRange"`
	Review                 jsonLDReview  `json:"review"`
	ServesCuisine          string        `json:"servesCuisine"`
	StarRating             string        `json:"starRating"`
	Telephone              string        `json:"telephone"`
	URL                    string        `json:"url"`
	AcceptsReservations    string        `json:"acceptsReservations"`
}

type jsonLDAddress struct {
	Type            string `json:"@type"`
	AddressCountry  any    `json:"addressCountry"`
	AddressLocality string `json:"addressLocality"`
	AddressRegion   string `json:"addressRegion"`
	PostalCode      string `json:"postalCode"`
	StreetAddress   string `json:"streetAddress"`
}

type jsonLDAward struct {
	Type        string `json:"@type"`
	AwardFor    string `json:"awardFor"`
	DateAwarded string `json:"dateAwarded"`
}

func (a *jsonLDAward) UnmarshalJSON(data []byte) error {
	var awardText string
	if err := json.Unmarshal(data, &awardText); err == nil {
		a.AwardFor = strings.TrimSpace(awardText)
		return nil
	}

	type alias jsonLDAward
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*a = jsonLDAward(decoded)
	return nil
}

type jsonLDGeo struct {
	Latitude  any `json:"latitude"`
	Longitude any `json:"longitude"`
}

type jsonLDReview struct {
	Type          string `json:"@type"`
	DatePublished string `json:"datePublished"`
	Description   string `json:"description"`
	Name          string `json:"name"`
}

func findAndParseJSONLD(e *colly.XMLElement) *jsonLDRestaurant {
	for _, script := range e.ChildTexts(xPathJSONLDScript) {
		if restaurant := parseJSONLDRestaurant(script); restaurant != nil {
			return restaurant
		}
	}
	return nil
}

func parseJSONLDRestaurant(raw string) *jsonLDRestaurant {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil
	}

	return findRestaurantJSONLD(payload)
}

func findRestaurantJSONLD(payload any) *jsonLDRestaurant {
	switch value := payload.(type) {
	case []any:
		for _, item := range value {
			if restaurant := findRestaurantJSONLD(item); restaurant != nil {
				return restaurant
			}
		}
	case map[string]any:
		if restaurant := decodeRestaurantJSONLD(value); restaurant != nil {
			return restaurant
		}
		if graph, ok := value["@graph"].([]any); ok {
			for _, item := range graph {
				if restaurant := findRestaurantJSONLD(item); restaurant != nil {
					return restaurant
				}
			}
		}
	}

	return nil
}

func decodeRestaurantJSONLD(node map[string]any) *jsonLDRestaurant {
	if !hasRestaurantShape(node) {
		return nil
	}

	encoded, err := json.Marshal(node)
	if err != nil {
		return nil
	}

	var restaurant jsonLDRestaurant
	if err := json.Unmarshal(encoded, &restaurant); err != nil {
		return nil
	}

	if !hasRestaurantType(restaurant.Type) && strings.TrimSpace(restaurant.Name) == "" {
		return nil
	}

	return &restaurant
}

func hasRestaurantShape(node map[string]any) bool {
	if hasRestaurantType(node["@type"]) {
		return true
	}

	_, hasAddress := node["address"]
	_, hasCuisine := node["servesCuisine"]
	_, hasReview := node["review"]
	_, hasLatitude := node["latitude"]
	_, hasLongitude := node["longitude"]
	name, _ := node["name"].(string)

	return strings.TrimSpace(name) != "" && (hasAddress || hasCuisine || hasReview || hasLatitude || hasLongitude)
}

func hasRestaurantType(value any) bool {
	switch typed := value.(type) {
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "Restaurant")
	case []any:
		for _, item := range typed {
			if hasRestaurantType(item) {
				return true
			}
		}
	}

	return false
}

func parseCoordinate(value any) string {
	switch typed := value.(type) {
	case string:
		coord := strings.TrimSpace(typed)
		if c, err := strconv.ParseFloat(coord, 64); err == nil && c >= -180.0 && c <= 180.0 {
			return coord
		}
	case float64:
		coord := strconv.FormatFloat(typed, 'f', -1, 64)
		if typed >= -180.0 && typed <= 180.0 {
			return coord
		}
	case int:
		coord := strconv.Itoa(typed)
		if typed >= -180 && typed <= 180 {
			return coord
		}
	}

	return ""
}

func (ld *jsonLDRestaurant) coordinates() (string, string) {
	if ld == nil {
		return "", ""
	}

	latitude := parseCoordinate(ld.Latitude)
	longitude := parseCoordinate(ld.Longitude)
	if latitude == "" {
		latitude = parseCoordinate(ld.Geo.Latitude)
	}
	if longitude == "" {
		longitude = parseCoordinate(ld.Geo.Longitude)
	}
	return latitude, longitude
}

func (ld *jsonLDRestaurant) descriptionText() string {
	if ld == nil {
		return ""
	}
	if description := TrimWhiteSpaces(ld.Review.Description); description != "" {
		return description
	}
	return TrimWhiteSpaces(ld.Description)
}

func (ld *jsonLDRestaurant) distinctionText() string {
	if ld == nil {
		return ""
	}
	if distinction := parseDistinctionStrict(ld.Award.AwardFor); distinction != "" {
		return distinction
	}
	if distinction := parseDistinctionStrict(ld.StarRating); distinction != "" {
		return distinction
	}
	return ""
}

func (ld *jsonLDRestaurant) publishedYear() int {
	if ld == nil {
		return 0
	}
	return parseJSONLDYear(ld.Award.DateAwarded)
}

func (ld *jsonLDRestaurant) reviewPublishedYear() int {
	if ld == nil {
		return 0
	}
	return parseJSONLDYear(ld.Review.DatePublished)
}

func parseJSONLDYear(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}

	if len(value) == 4 {
		if year, err := strconv.Atoi(value); err == nil && validateYear(year) {
			return year
		}
	}

	for _, layout := range commonDateLayouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			year := parsed.Year()
			if validateYear(year) {
				return year
			}
		}
	}

	return 0
}

func (ld *jsonLDRestaurant) addressText() string {
	if ld == nil {
		return ""
	}

	parts := make([]string, 0, 5)
	appendUnique := func(part string) {
		part = strings.TrimSpace(part)
		if part == "" {
			return
		}
		for _, existing := range parts {
			if strings.EqualFold(existing, part) {
				return
			}
		}
		parts = append(parts, part)
	}

	appendUnique(ld.Address.StreetAddress)
	appendUnique(ld.Address.AddressLocality)
	appendUnique(ld.Address.AddressRegion)
	appendUnique(ld.Address.PostalCode)
	appendUnique(stringifyJSONLDValue(ld.Address.AddressCountry))

	return strings.Join(parts, ", ")
}

func (ld *jsonLDRestaurant) locationText() string {
	if ld == nil {
		return ""
	}

	locality := strings.TrimSpace(ld.Address.AddressLocality)
	region := strings.TrimSpace(ld.Address.AddressRegion)
	country := strings.TrimSpace(stringifyJSONLDValue(ld.Address.AddressCountry))

	switch {
	case locality != "" && country != "" && !strings.EqualFold(locality, country):
		return locality + ", " + country
	case locality != "":
		return locality
	case region != "" && country != "" && !strings.EqualFold(region, country):
		return region + ", " + country
	case region != "":
		return region
	default:
		return country
	}
}

func stringifyJSONLDValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		if name, ok := typed["name"].(string); ok {
			return strings.TrimSpace(name)
		}
	}

	return ""
}
