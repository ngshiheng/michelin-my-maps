package backfill

import "regexp"

// Price validation regex patterns for extracting price information from HTML text.
// These patterns handle various price formats found on Michelin Guide pages.
var (
	// currencyRegex matches pure currency symbols (e.g., "$$$$", "€€€€")
	currencyRegex = regexp.MustCompile(`^[€$£¥₩₽₹฿₺]+$`)

	// priceCodeRegex matches price with currency code (e.g., "1,800 NOK", "155 EUR", "300 - 2,000 MOP")
	priceCodeRegex = regexp.MustCompile(`^[0-9][0-9,.\-\s]*[0-9]\s*[A-Z]{2,4}$`)

	// priceRangeRegex matches numeric price ranges (e.g., "155 - 380", "300 - 2,000")
	priceRangeRegex = regexp.MustCompile(`^[0-9][0-9,.\-\s]*[0-9]$`)

	// overUnderRegex matches "Over X" or "Under X" patterns (e.g., "Over 75 USD")
	overUnderRegex = regexp.MustCompile(`^(Over|Under)\s+\d+`)

	// betweenRegex matches "Between X and Y [CURRENCY]" patterns (e.g., "Between 350 and 500 HKD")
	betweenRegex = regexp.MustCompile(`^Between\s+\d+.*\d+\s+[A-Z]{2,4}$`)

	// toRangeRegex matches "X to Y [CURRENCY]" patterns (e.g., "500 to 1500 TWD")
	toRangeRegex = regexp.MustCompile(`^\d+\s+to\s+\d+\s+[A-Z]{2,4}$`)

	// lessThanRegex matches "Less than X [CURRENCY]" patterns (e.g., "Less than 200 THB")
	lessThanRegex = regexp.MustCompile(`(?i)^Less than \d+(\.\d+)?\s*[A-Z]{2,4}$`)
)

// Date extraction patterns for parsing published dates from various HTML formats
var (
	// yearMichelinGuideRegex matches patterns like "2023 MICHELIN Guide"
	yearMichelinGuideRegex = regexp.MustCompile(`(\d{4})\s+MICHELIN Guide`)

	// michelinGuideYearRegex matches patterns like "MICHELIN Guide ... 2023"
	michelinGuideYearRegex = regexp.MustCompile(`MICHELIN Guide.*?(\d{4})`)

	// isoDateRegex matches ISO date format (e.g., "2023-01-25")
	isoDateRegex = regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
)

// HTML parsing constants
const (
	// waybackIDMarker is the URL marker used by Wayback Machine to separate snapshot metadata from original URL
	waybackIDMarker = "id_/"

	// jsonLDRestaurantType is the JSON-LD type identifier for restaurant data
	jsonLDRestaurantType = `"@type":"Restaurant"`

	// dLayerMarkers are the required text markers for identifying dLayer script content
	dLayerDataMarker     = "dLayer"
	dLayerDistinctionKey = "distinction"

	// Text separators commonly found in price information
	priceSeparators = "·•"
)

// Error messages for validation and extraction failures
const (
	ErrEmptyURL         = "snapshot URL cannot be empty"
	ErrInvalidURL       = "invalid URL format"
	ErrMissingIDMarker  = "wayback ID marker not found in URL"
	ErrJSONParseFailure = "failed to parse JSON-LD content"
	ErrMissingElement   = "required HTML element not found"
)
