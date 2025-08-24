package parsers

import "regexp"

// extractOriginalURL extracts the original URL from a Wayback Machine archive URL
// Wayback URLs have format: https://web.archive.org/web/YYYYMMDDhhmmss/ORIGINAL_URL
func extractOriginalURL(waybackURL string) string {
	waybackPattern := regexp.MustCompile(`https?://web\.archive\.org/web/\d{14}[^/]*/(.+)`)
	matches := waybackPattern.FindStringSubmatch(waybackURL)

	if len(matches) >= 2 {
		return matches[1]
	}

	return waybackURL
}
