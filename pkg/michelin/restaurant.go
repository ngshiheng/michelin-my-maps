package michelin

import "time"

// Restaurant stores information about a restaurant on Michelin Guide.
type Restaurant struct {
	Address               string `gorm:"not null"`
	Cuisine               string
	Description           string
	Distinction           string `gorm:"not null;index:idx_distinction"`
	FacilitiesAndServices string // Comma-separated string
	GreenStar             bool
	Location              string `gorm:"not null;index:idx_location"`
	Latitude              string
	Longitude             string
	Name                  string `gorm:"not null;index:idx_name"`
	PhoneNumber           string
	Price                 string
	URL                   string `gorm:"unique"`
	WebsiteURL            string
	UpdatedOn             time.Time
}
