package michelin

import (
	"gorm.io/gorm"
)

// Restaurant stores information about a restaurant on Michelin Guide.
type Restaurant struct {
	gorm.Model
	URL                   string `gorm:"unique;not null;index"`
	Name                  string `gorm:"not null;index:idx_name"`
	Description           string
	Address               string `gorm:"not null"`
	Location              string `gorm:"not null;index:idx_location"`
	Latitude              string
	Longitude             string
	Cuisine               string
	PhoneNumber           string
	FacilitiesAndServices string // Comma-separated string
	WebsiteURL            string

	// Relationship
	Awards []RestaurantAward `gorm:"foreignKey:RestaurantID"`
}

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
