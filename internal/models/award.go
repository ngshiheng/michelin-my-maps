package models

import "gorm.io/gorm"

const (
	ThreeStars          = "3 Stars"
	TwoStars            = "2 Stars"
	OneStar             = "1 Star"
	BibGourmand         = "Bib Gourmand"
	SelectedRestaurants = "Selected Restaurants"
)

// RestaurantAward stores award information for a restaurant in a specific year.
type RestaurantAward struct {
	gorm.Model
	RestaurantID uint   `gorm:"not null;index:idx_restaurant_year;constraint:OnDelete:CASCADE;uniqueIndex:idx_restaurant_year_unique"`
	Year         int    `gorm:"not null;index:idx_restaurant_year;index:idx_year;uniqueIndex:idx_restaurant_year_unique"`
	Distinction  string `gorm:"not null;index:idx_distinction"`
	Price        string
	GreenStar    bool
}

// TableName sets the table name for RestaurantAward
func (RestaurantAward) TableName() string {
	return "restaurant_awards"
}
