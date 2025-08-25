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

func ParseGreenStar(text string) string {
	if strings.Contains(strings.ToLower(text), "green star") {
		return "true"
	}
	return "false"
}

func ParseDistinction(text string) string {
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
		return models.SelectedRestaurants
	}
}

func decodeHTMLEntities(text string) string {
	text = strings.ReplaceAll(text, "&bull;", "")
	text = strings.ReplaceAll(text, "•", "")
	return text
}
