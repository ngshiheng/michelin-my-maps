package models

import (
	"errors"
	"fmt"
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

// RestaurantAward stores award information for a restaurant in a specific year.
type RestaurantAward struct {
	ID           uint   `gorm:"primaryKey"`
	WaybackURL   string `gorm:"column:wayback_url"` // "" for live scraping, URL for backfill
	RestaurantID uint   `gorm:"not null;index:idx_restaurant_year;constraint:OnDelete:CASCADE;uniqueIndex:idx_restaurant_year_unique"`
	Distinction  string `gorm:"not null;index:idx_distinction"`
	GreenStar    bool
	Price        string `gorm:"not null"`
	Year         int    `gorm:"not null;index:idx_restaurant_year;index:idx_year;uniqueIndex:idx_restaurant_year_unique"`

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
	if r.RestaurantID == 0 {
		return errors.New("restaurant ID must be positive")
	}
	if strings.TrimSpace(r.Distinction) == "" {
		return errors.New("distinction cannot be empty")
	}
	allowed := map[string]bool{
		ThreeStars:          true,
		TwoStars:            true,
		OneStar:             true,
		BibGourmand:         true,
		SelectedRestaurants: true,
	}
	if !allowed[r.Distinction] {
		return errors.New("distinction must be a valid value")
	}
	if strings.TrimSpace(r.Price) == "" {
		return errors.New("price cannot be empty")
	}
	currentYear := time.Now().Year()
	if r.Year < 1900 || r.Year > currentYear+1 {
		return fmt.Errorf("year must be between 1900 and %d", currentYear)
	}
	return nil
}

// TableName sets the table name for RestaurantAward
func (RestaurantAward) TableName() string {
	return "restaurant_awards"
}
