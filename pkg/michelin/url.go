package michelin

// GuideURL represents a Michelin guide distinction and its corresponding URL.
type GuideURL struct {
	Distinction string
	URL         string
}

var DistinctionURL = map[string]string{
	"3 MICHELIN Stars": "https://guide.michelin.com/en/restaurants/3-stars-michelin/",
	"2 MICHELIN Stars": "https://guide.michelin.com/en/restaurants/2-stars-michelin/",
	"1 MICHELIN Star":  "https://guide.michelin.com/en/restaurants/1-star-michelin/",
	"Bib Gourmand":     "https://guide.michelin.com/en/restaurants/bib-gourmand",
}
