package parsers

import "regexp"

var (
	// waybackRegex matches a Wayback Machine archive URL
	waybackRegex = regexp.MustCompile(`https?://web\.archive\.org/web/\d{14}[^/]*/(.+)`)
)

// extractOriginalURL extracts the original URL from a Wayback Machine archive URL.
// e.g. https://web.archive.org/web/YYYYMMDDhhmmss/ORIGINAL_URL
func extractOriginalURL(waybackURL string) string {
	matches := waybackRegex.FindStringSubmatch(waybackURL)
	if len(matches) >= 2 {
		return matches[1]
	}
	return waybackURL
}
