package models

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Restaurant stores information about a restaurant on Michelin Guide.
type Restaurant struct {
	ID                    uint              `gorm:"primaryKey"`
	URL                   string            `gorm:"unique;not null;index"`
	Address               string            `gorm:"not null"`
	Awards                []RestaurantAward `gorm:"foreignKey:RestaurantID"`
	Cuisine               string            `gorm:"not null"`
	Description           string            `gorm:"not null"`
	FacilitiesAndServices string            // Comma-separated string
	Latitude              string            `gorm:"not null"`
	Location              string            `gorm:"not null;index:idx_location"`
	Longitude             string            `gorm:"not null"`
	Name                  string            `gorm:"index:idx_name"`
	PhoneNumber           string
	WebsiteURL            string

	CreatedAt time.Time `gorm:"type:datetime"`
	UpdatedAt time.Time `gorm:"type:datetime"`
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
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name cannot be empty")
	}
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
