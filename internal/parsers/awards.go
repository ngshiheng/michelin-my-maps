package parsers

import (
	"regexp"
	"strings"

	"github.com/ngshiheng/michelin-my-maps/v3/internal/models"
)

var (
	// Michelin distinction patterns for parsing award levels
	re3Stars      = regexp.MustCompile(`(?i)\b(three|3)\b.*?\bstars?\b`)
	re2Stars      = regexp.MustCompile(`(?i)\b(two|2)\b.*?\bstars?\b`)
	re1Star       = regexp.MustCompile(`(?i)\b(one|1)\b.*?\bstar\b`)
	reBibGourmand = regexp.MustCompile(`(?i)\bbib\b`)
	reSelected    = regexp.MustCompile(`(?i)\bselected\s*restaurants?\b|\bplate\b`)
)

func parseGreenStar(text string) string {
	if strings.Contains(strings.ToLower(text), "green star") {
		return "true"
	}
	return "false"
}

func parseDistinction(distinction string) string {
	s := strings.ToLower(distinction)
	s = decodeHTMLEntities(s)
	s = strings.Trim(s, " .!?,;:-")
	s = strings.TrimSpace(s)

	switch {
	case re3Stars.MatchString(s):
		return models.ThreeStars
	case re2Stars.MatchString(s):
		return models.TwoStars
	case re1Star.MatchString(s):
		return models.OneStar
	case reBibGourmand.MatchString(s):
		return models.BibGourmand
	case reSelected.MatchString(s):
		return models.SelectedRestaurants
	default:
		return models.SelectedRestaurants
	}
}

func decodeHTMLEntities(s string) string {
	s = strings.ReplaceAll(s, "&bull;", "")
	s = strings.ReplaceAll(s, "•", "")
	return s
}
