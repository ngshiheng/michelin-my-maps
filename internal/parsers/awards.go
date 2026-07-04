package parsers

import (
	"regexp"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/ngshiheng/michelin-my-maps/v4/internal/models"
)

var (
	re3Stars      = regexp.MustCompile(`(?i)\b(three|3)\b.*?\bstars?\b`)
	re2Stars      = regexp.MustCompile(`(?i)\b(two|2)\b.*?\bstars?\b`)
	re1Star       = regexp.MustCompile(`(?i)\b(one|1)\b.*?\bstar\b`)
	reBibGourmand = regexp.MustCompile(`(?i)\bbib\b`)
	reSelected    = regexp.MustCompile(`(?i)\bselected\s*restaurants?\b|\bplate\b`)
)

// ExtractDistinction extracts the restaurant's distinction and green star status from an XML element.
// NOTE: Green Star award has ended (2026 - 2026)
func ExtractDistinction(e *colly.XMLElement) (string, bool) {
	greenStar := extractGreenStar(e)

	if distinction := tryAwardSelectors(e, "distinction", parseDistinctionStrict); distinction != "" {
		return distinction, greenStar
	}

	if distinction := extractDistinctionFromJSONLD(findAndParseJSONLD(e)); distinction != "" {
		return distinction, greenStar
	}

	if distinction := parseDistinctionStrict(parseDLayerValue(findDLayerScript(e), "distinction")); distinction != "" {
		return distinction, greenStar
	}

	return models.SelectedRestaurants, greenStar
}

func extractGreenStar(e *colly.XMLElement) bool {
	greenStar := tryAwardSelectors(e, "greenStar", parseGreenStar) == "true"
	if !greenStar {
		greenStar = parseDLayerValue(findDLayerScript(e), "greenstar") == "True"
	}
	return greenStar
}

func parseGreenStar(text string) string {
	if strings.Contains(strings.ToLower(text), "green star") {
		return "true"
	}
	return "false"
}

func parseDistinctionStrict(text string) string {
	distinction := strings.ToLower(text)
	distinction = decodeHTMLEntities(distinction)
	distinction = strings.Trim(distinction, " .!?,;:-")
	distinction = strings.TrimSpace(distinction)

	switch {
	case re3Stars.MatchString(distinction):
		return models.ThreeStars
	case re2Stars.MatchString(distinction):
		return models.TwoStars
	case re1Star.MatchString(distinction):
		return models.OneStar
	case reBibGourmand.MatchString(distinction):
		return models.BibGourmand
	case reSelected.MatchString(distinction):
		return models.SelectedRestaurants
	default:
		return ""
	}
}

func decodeHTMLEntities(text string) string {
	replacer := strings.NewReplacer(
		"&bull;", "",
		"•", "",
	)
	return replacer.Replace(text)
}

func extractDistinctionFromJSONLD(ld *jsonLDRestaurant) string {
	if ld == nil {
		return ""
	}
	return ld.distinctionText()
}
