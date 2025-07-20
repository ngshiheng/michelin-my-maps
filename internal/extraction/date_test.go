package extraction

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
			got := ParseDateFromText(tt.input)
			if got != tt.expected {
				t.Errorf("ParseDateFromText(%q) = %q; want %q", tt.input, got, tt.expected)
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
			got := ParseYearFromAnyFormat(tt.input)
			if got != tt.expected {
				t.Errorf("ParseYearFromAnyFormat(%q) = %d; want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParsePublishedYear(t *testing.T) {
	jsonLD := `{"review":{"datePublished":"2023-01-25"}}`
	year, err := ParsePublishedYear(jsonLD)
	if err != nil {
		t.Fatalf("ParsePublishedYear() unexpected error: %v", err)
	}
	if year != 2023 {
		t.Errorf("ParsePublishedYear() = %d; want 2023", year)
	}

	jsonLD = `{"review":{"datePublished":"2019"}}`
	year, err = ParsePublishedYear(jsonLD)
	if err != nil {
		t.Fatalf("ParsePublishedYear() unexpected error: %v", err)
	}
	if year != 2019 {
		t.Errorf("ParsePublishedYear() = %d; want 2019", year)
	}

	jsonLD = `{"review":{"datePublished":"not-a-date"}}`
	year, err = ParsePublishedYear(jsonLD)
	if err != nil {
		t.Fatalf("ParsePublishedYear() unexpected error: %v", err)
	}
	if year != 0 {
		t.Errorf("ParsePublishedYear() = %d; want 0 for invalid date", year)
	}

	jsonLD = `{}`
	year, err = ParsePublishedYear(jsonLD)
	if err != nil {
		t.Fatalf("ParsePublishedYear() unexpected error: %v", err)
	}
	if year != 0 {
		t.Errorf("ParsePublishedYear() = %d; want 0 for missing review", year)
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
