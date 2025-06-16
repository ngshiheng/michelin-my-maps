package michelin

import "github.com/ngshiheng/michelin-my-maps/v3/internal/models"

// GuideURL represents a Michelin guide distinction and its corresponding URL.
type GuideURL struct {
	Distinction string
	URL         string
}

// DistinctionURL contains the starting URL used by the crawler.
var DistinctionURL = map[string]string{
	models.ThreeStars:  "https://guide.michelin.com/en/restaurants/3-stars-michelin",
	models.TwoStars:    "https://guide.michelin.com/en/restaurants/2-stars-michelin",
	models.OneStar:     "https://guide.michelin.com/en/restaurants/1-star-michelin",
	models.BibGourmand: "https://guide.michelin.com/en/restaurants/bib-gourmand",
	// models.SelectedRestaurants: "https://guide.michelin.com/en/restaurants/the-plate-michelin",
}
