package models

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	ThreeStars          = "3 Stars"
	TwoStars            = "2 Stars"
	OneStar             = "1 Star"
	BibGourmand         = "Bib Gourmand"
	SelectedRestaurants = "Selected Restaurants"
)

// GuideURL represents a Michelin guide distinction and its corresponding URL.
type GuideURL struct {
	Distinction string
	URL         string
}

// DistinctionURL contains the starting URL used by the scraper.
var DistinctionURL = map[string]string{
	ThreeStars:          "https://guide.michelin.com/en/restaurants/3-stars-michelin",
	TwoStars:            "https://guide.michelin.com/en/restaurants/2-stars-michelin",
	OneStar:             "https://guide.michelin.com/en/restaurants/1-star-michelin",
	BibGourmand:         "https://guide.michelin.com/en/restaurants/bib-gourmand",
	SelectedRestaurants: "https://guide.michelin.com/en/restaurants/the-plate-michelin",
}

// RestaurantAward stores award information for a restaurant in a specific year.
type RestaurantAward struct {
	ID           uint   `gorm:"primaryKey"`
	RestaurantID uint   `gorm:"not null;index:idx_restaurant_year;constraint:OnDelete:CASCADE;uniqueIndex:idx_restaurant_year_unique"`
	Year         int    `gorm:"not null;index:idx_restaurant_year;index:idx_year;uniqueIndex:idx_restaurant_year_unique"`
	Distinction  string `gorm:"not null;index:idx_distinction"`
	Price        string `gorm:"not null"`
	GreenStar    bool

	CreatedAt time.Time `gorm:"type:datetime"`
	UpdatedAt time.Time `gorm:"type:datetime"`
}

// BeforeCreate runs validation before creating a restaurant award record
func (r *RestaurantAward) BeforeCreate(tx *gorm.DB) error {
	return r.validate()
}

// BeforeUpdate runs validation before updating a restaurant award record
func (r *RestaurantAward) BeforeUpdate(tx *gorm.DB) error {
	return r.validate()
}

// validate checks that required fields are not empty
func (r *RestaurantAward) validate() error {
	if strings.TrimSpace(r.Distinction) == "" {
		return errors.New("distinction cannot be empty")
	}
	if strings.TrimSpace(r.Price) == "" {
		return errors.New("price cannot be empty")
	}
	return nil
}

// TableName sets the table name for RestaurantAward
func (RestaurantAward) TableName() string {
	return "restaurant_awards"
}
