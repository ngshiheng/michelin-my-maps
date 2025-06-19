package models

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// Restaurant stores information about a restaurant on Michelin Guide.
type Restaurant struct {
	gorm.Model
	URL                   string `gorm:"unique;not null;index"`
	Name                  string `gorm:"index:idx_name"`
	Description           string `gorm:"not null"`
	Address               string `gorm:"not null"`
	Location              string `gorm:"not null;index:idx_location"`
	Latitude              string `gorm:"not null"`
	Longitude             string `gorm:"not null"`
	Cuisine               string `gorm:"not null"`
	PhoneNumber           string
	FacilitiesAndServices string // Comma-separated string
	WebsiteURL            string

	// Relationship
	Awards []RestaurantAward `gorm:"foreignKey:RestaurantID"`
}

// BeforeCreate runs validation before creating a restaurant record
func (r *Restaurant) BeforeCreate(tx *gorm.DB) error {
	return r.validate()
}

// BeforeUpdate runs validation before updating a restaurant record
func (r *Restaurant) BeforeUpdate(tx *gorm.DB) error {
	return r.validate()
}

// validate checks that required fields are not empty
func (r *Restaurant) validate() error {
	if strings.TrimSpace(r.Address) == "" {
		return errors.New("address cannot be empty")
	}
	if strings.TrimSpace(r.Location) == "" {
		return errors.New("location cannot be empty")
	}
	if strings.TrimSpace(r.Cuisine) == "" {
		return errors.New("cuisine cannot be empty")
	}
	if strings.TrimSpace(r.Latitude) == "" {
		return errors.New("latitude cannot be empty")
	}
	if strings.TrimSpace(r.Longitude) == "" {
		return errors.New("longitude cannot be empty")
	}
	if strings.TrimSpace(r.URL) == "" {
		return errors.New("URL cannot be empty")
	}
	if strings.TrimSpace(r.Description) == "" {
		return errors.New("description cannot be empty")
	}
	return nil
}
