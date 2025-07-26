package extraction

import (
	"testing"

	"github.com/gocolly/colly/v2"
)

func TestExtractContextData_AllFieldsPresent(t *testing.T) {
	ctx := colly.NewContext()
	ctx.Put("location", "Singapore")
	ctx.Put("latitude", "1.3521")
	ctx.Put("longitude", "103.8198")
	ctx.Put("publishedYear", 2023)

	got := ExtractContextData(ctx)
	want := RequestContext{
		Location:  "Singapore",
		Latitude:  "1.3521",
		Longitude: "103.8198",
		Year:      2023,
	}
	if got != want {
		t.Errorf("ExtractContextData() = %+v; want %+v", got, want)
	}
}

func TestExtractContextData_MissingFields(t *testing.T) {
	ctx := colly.NewContext()
	// No fields set

	got := ExtractContextData(ctx)
	want := RequestContext{
		Location:  "",
		Latitude:  "",
		Longitude: "",
		Year:      0,
	}
	if got != want {
		t.Errorf("ExtractContextData() = %+v; want %+v", got, want)
	}
}

func TestExtractContextData_InvalidYearType(t *testing.T) {
	ctx := colly.NewContext()
	ctx.Put("publishedYear", "not-an-int")

	got := ExtractContextData(ctx)
	if got.Year != 0 {
		t.Errorf("ExtractContextData() Year = %d; want 0 for invalid type", got.Year)
	}
}
