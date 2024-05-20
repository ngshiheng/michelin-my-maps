package michelin

// GuideURL represents a Michelin guide distinction and its corresponding URL.
type GuideURL struct {
	Distinction string
	URL         string
}

// DistinctionURL contains the starting URL used by the crawler.
var DistinctionURL = map[string]string{
	ThreeStars:          "https://guide.michelin.com/en/restaurants/3-stars-michelin",
	TwoStars:            "https://guide.michelin.com/en/restaurants/2-stars-michelin",
	OneStar:             "https://guide.michelin.com/en/restaurants/1-star-michelin",
	BibGourmand:         "https://guide.michelin.com/en/restaurants/bib-gourmand",
	SelectedRestaurants: "https://guide.michelin.com/en/restaurants/the-plate-michelin",
}
