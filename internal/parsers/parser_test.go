package parsers

import (
	"testing"

	"github.com/ngshiheng/michelin-my-maps/v4/internal/models"
)

func TestParsePrefersJSONLDAndKeepsFallbacks(t *testing.T) {
	html := `<html><body>
	<script type="application/ld+json"><![CDATA[` + wakuGhinJSONLD + `]]></script>
	<div class="data-sheet__title">Wrong Name</div>
	<div class="data-sheet__block--text">Ignored address</div>
	<div class="data-sheet__block--text">$$$ · Wrong Cuisine</div>
	<ul class="restaurant-details__classification--list"><li>Three MICHELIN Stars: Exceptional cuisine, worth a special journey!</li></ul>
	<div>MICHELIN Green Star</div>
	<a data-event="CTA_website" href="https://example.com"></a>
	<div class="col col-12 col-lg-6"><li>Air conditioning</li><li>Terrace</li></div>
	</body></html>`

	e := mustTestXMLElement(t, html, "https://guide.michelin.com/sg/en/singapore-region/singapore/restaurant/waku-ghin")
	data := Parse(e)

	if data.Name != "Waku Ghin" {
		t.Fatalf("Name = %q; want %q", data.Name, "Waku Ghin")
	}
	if data.Description != "The contemporary room is divided into three sections." {
		t.Fatalf("Description = %q", data.Description)
	}
	if data.Address != "The Shoppes at Marina Bay Sands, Level 2 Dining, L2-03, 10 Bayfront Avenue, Singapore, 018956, SGP" {
		t.Fatalf("Address = %q", data.Address)
	}
	if data.Location != "Singapore, SGP" {
		t.Fatalf("Location = %q", data.Location)
	}
	if data.PhoneNumber != "+6566888507" {
		t.Fatalf("PhoneNumber = %q", data.PhoneNumber)
	}
	if data.Cuisine != "Japanese Contemporary" {
		t.Fatalf("Cuisine = %q", data.Cuisine)
	}
	if data.Distinction != models.OneStar {
		t.Fatalf("Distinction = %q; want %q", data.Distinction, models.OneStar)
	}
	if !data.GreenStar {
		t.Fatal("GreenStar = false; want true")
	}
	if data.Latitude != "1.283175" || data.Longitude != "103.8598" {
		t.Fatalf("Coordinates = (%q, %q)", data.Latitude, data.Longitude)
	}
	if data.Year != 2026 {
		t.Fatalf("Year = %d; want %d", data.Year, 2026)
	}
	if data.Price != "$$$" {
		t.Fatalf("Price = %q; want %q", data.Price, "$$$")
	}
	if data.WebsiteURL != "https://example.com" {
		t.Fatalf("WebsiteURL = %q", data.WebsiteURL)
	}
	if data.FacilitiesAndServices != "Air conditioning,Terrace" {
		t.Fatalf("FacilitiesAndServices = %q", data.FacilitiesAndServices)
	}
	if data.URL != "https://guide.michelin.com/sg/en/singapore-region/singapore/restaurant/waku-ghin" {
		t.Fatalf("URL = %q", data.URL)
	}
	if data.WaybackURL != "" {
		t.Fatalf("WaybackURL = %q; want empty", data.WaybackURL)
	}
}
