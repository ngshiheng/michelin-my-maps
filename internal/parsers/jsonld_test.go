package parsers

import "testing"

const wakuGhinJSONLD = `{
	"@context":"http://schema.org",
	"address":{
		"@type":"PostalAddress",
		"streetAddress":"The Shoppes at Marina Bay Sands, Level 2 Dining, L2-03, 10 Bayfront Avenue",
		"addressLocality":"Singapore",
		"postalCode":"018956",
		"addressCountry":"SGP",
		"addressRegion":"Singapore"
	},
	"name":"Waku Ghin",
	"image":"https://axwwgrkdco.cloudimg.io/v7/__gmpics3__/55d5395b4f86418daa31c20c6a46adc1.jpg?width=1000",
	"@type":"Restaurant",
	"review":{
		"@type":"Review",
		"datePublished":"2025-07-24T07:04",
		"name":"Waku Ghin",
		"description":"The contemporary room is divided into three sections.",
		"author":{
			"@type":"Person",
			"name":"Michelin Inspector"
		}
	},
	"telephone":"+65 6688 8507",
	"knowsLanguage":"en-SG",
	"acceptsReservations":"Yes",
	"servesCuisine":"Japanese Contemporary",
	"url":"https://guide.michelin.com/sg/en/singapore-region/singapore/restaurant/waku-ghin",
	"starRating":"One Star: High quality cooking",
	"currenciesAccepted":"SGD",
	"paymentAccepted":"American Express credit card, China UnionPay",
	"priceRange":"Spare no expense",
	"award":{
		"@type":"Award",
		"awardFor":"One Star: High quality cooking",
		"awardedBy":{
			"@type":"Organization",
			"name":"Michelin"
		},
		"dateAwarded":"2026"
	},
	"brand":"MICHELIN Guide",
	"hasDriveThroughService":"False",
	"latitude":1.2831750,
	"longitude":103.8598000,
	"hasMap":"https://www.google.com/maps/search/?api=1&query=1.2831750%2C103.8598000"
}`

func TestParseJSONLDRestaurant(t *testing.T) {
	ld := parseJSONLDRestaurant(wakuGhinJSONLD)
	if ld == nil {
		t.Fatal("parseJSONLDRestaurant returned nil")
	}

	if ld.Name != "Waku Ghin" {
		t.Fatalf("Name = %q; want %q", ld.Name, "Waku Ghin")
	}

	if got := ld.descriptionText(); got != "The contemporary room is divided into three sections." {
		t.Fatalf("descriptionText() = %q", got)
	}

	if got := ld.distinctionText(); got != "1 Star" {
		t.Fatalf("distinctionText() = %q; want %q", got, "1 Star")
	}

	if got := ld.publishedYear(); got != 2026 {
		t.Fatalf("publishedYear() = %d; want %d", got, 2026)
	}

	lat, lng := ld.coordinates()
	if lat != "1.283175" || lng != "103.8598" {
		t.Fatalf("coordinates() = (%q, %q)", lat, lng)
	}

	if got := ld.addressText(); got != "The Shoppes at Marina Bay Sands, Level 2 Dining, L2-03, 10 Bayfront Avenue, Singapore, 018956, SGP" {
		t.Fatalf("addressText() = %q", got)
	}

	if got := ld.locationText(); got != "Singapore, SGP" {
		t.Fatalf("locationText() = %q", got)
	}
}

func TestParseJSONLDRestaurantFromGraph(t *testing.T) {
	ld := parseJSONLDRestaurant(`{"@graph":[{"@type":"BreadcrumbList"},` + wakuGhinJSONLD + `]}`)
	if ld == nil {
		t.Fatal("parseJSONLDRestaurant returned nil for @graph payload")
	}

	if ld.Name != "Waku Ghin" {
		t.Fatalf("Name = %q; want %q", ld.Name, "Waku Ghin")
	}
}

func TestParseJSONLDLegacyAwardString(t *testing.T) {
	ld := parseJSONLDRestaurant(`{
		"@context":"http://schema.org",
		"@type":"Restaurant",
		"name":"Christopher Coutanceau",
		"award":"Three MICHELIN Stars: Exceptional cuisine, worth a special journey!",
		"review":{"datePublished":"2021-01-18T09:34"}
	}`)
	if ld == nil {
		t.Fatal("parseJSONLDRestaurant returned nil")
	}

	if got := ld.distinctionText(); got != "3 Stars" {
		t.Fatalf("distinctionText() = %q; want %q", got, "3 Stars")
	}
}
