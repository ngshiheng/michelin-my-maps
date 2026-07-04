package parsers

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/antchfx/xmlquery"
	"github.com/gocolly/colly/v2"
	"github.com/ngshiheng/michelin-my-maps/v4/internal/models"
)

func TestDecodeHTMLEntities(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"&bull; 3 star", " 3 star"},
		{"• 2 stars", " 2 stars"},
		{"No entity", "No entity"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := decodeHTMLEntities(tt.input)
			if got != tt.expected {
				t.Errorf("decodeHTMLEntities(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseGreenStar(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"green star", "true"},
		{"MICHELIN Green Star", "true"},
		{"has a Green Star award", "true"},
		{"1 Star", "false"},
		{"Bib Gourmand", "false"},
		{"", "false"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseGreenStar(tt.input)
			if got != tt.expected {
				t.Errorf("parseGreenStar(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseDistinctionStrict(t *testing.T) {
	if got := parseDistinctionStrict(""); got != "" {
		t.Fatalf("parseDistinctionStrict(empty) = %q; want empty", got)
	}

	if got := parseDistinctionStrict("Extremely comfortable restaurant"); got != "" {
		t.Fatalf("parseDistinctionStrict(unrelated text) = %q; want empty", got)
	}

	if got := parseDistinctionStrict("Three MICHELIN Stars: Exceptional cuisine, worth a special journey!"); got != models.ThreeStars {
		t.Fatalf("parseDistinctionStrict(stars) = %q; want %q", got, models.ThreeStars)
	}
}

func TestExtractDistinctionFallsBackToDLayer(t *testing.T) {
	html := `<html><body><script>dLayer['distinction'] = '3 star';</script></body></html>`
	e := mustTestXMLElement(t, html, "https://guide.michelin.com/test")

	distinction, greenStar := ExtractDistinction(e)
	if distinction != models.ThreeStars {
		t.Fatalf("ExtractDistinction() = %q; want %q", distinction, models.ThreeStars)
	}
	if greenStar {
		t.Fatal("ExtractDistinction() unexpectedly set green star")
	}
}

func mustHTMLDoc(t *testing.T, body string) *xmlquery.Node {
	t.Helper()

	doc, err := xmlquery.Parse(strings.NewReader(body))
	if err != nil {
		t.Fatalf("xmlquery.Parse() error = %v", err)
	}
	return doc
}

func mustTestXMLElement(t *testing.T, body, rawURL string) *colly.XMLElement {
	t.Helper()

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	response := &colly.Response{Request: &colly.Request{URL: parsedURL, Headers: &http.Header{}}}
	return colly.NewXMLElementFromXMLNode(response, mustHTMLDoc(t, body))
}
