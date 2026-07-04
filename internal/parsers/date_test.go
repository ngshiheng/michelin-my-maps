package parsers

import (
	"testing"
	"time"
)

func TestParseDateFromText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2023 MICHELIN Guide", "2023"},
		{"MICHELIN Guide Singapore 2022", "2022"},
		{"2021-07-15", "2021-07-15"},
		{"No date here", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseDateFromText(tt.input)
			if got != tt.expected {
				t.Errorf("parseDateFromText(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseYearFromAnyFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"2023-01-25", 2023},
		{"2022 MICHELIN Guide", 2022},
		{"MICHELIN Guide Singapore 2021", 2021},
		{"2019", 2019},
		{"", 0},
		{"not a date", 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseYearFromAnyFormat(tt.input)
			if got != tt.expected {
				t.Errorf("parseYearFromAnyFormat(%q) = %d; want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestValidateYear(t *testing.T) {
	currentYear := time.Now().Year()
	tests := []struct {
		input    int
		expected bool
	}{
		{1900, true},
		{currentYear, true},
		{currentYear + 1, true},
		{1899, false},
		{currentYear + 2, false},
		{0, false},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := validateYear(tt.input)
			if got != tt.expected {
				t.Errorf("validateYear(%d) = %v; want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExtractPublishedYearPrefersMetaBeforeReviewDate(t *testing.T) {
	html := `<html><head>
	<meta name="description" content="Christopher Coutanceau – a Three MICHELIN Stars: Exceptional cuisine, worth a special journey! restaurant in the 2022 MICHELIN Guide France." />
	<script type="application/ld+json"><![CDATA[{"@context":"http://schema.org","@type":"Restaurant","review":{"datePublished":"2021-01-18T09:34"}}]]></script>
	</head><body><div class="restaurant-details__heading--label-title">MICHELIN Guide France</div></body></html>`

	e := mustTestXMLElement(t, html, "https://guide.michelin.com/test")
	if year := ExtractPublishedYear(e); year != 2022 {
		t.Fatalf("ExtractPublishedYear() = %d; want %d", year, 2022)
	}
}
