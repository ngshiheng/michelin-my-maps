package models

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
