package michelin

import "time"

// Restaurant stores information about a restaurant on Michelin Guide.
type Restaurant struct {
	Address               string
	Cuisine               string
	Description           string
	Distinction           string // Comma-separated string
	FacilitiesAndServices string // Comma-separated string
	Latitude              string
	Location              string
	Longitude             string
	Name                  string `gorm:"not null"`
	PhoneNumber           string
	Price                 string
	URL                   string `gorm:"unique"`
	WebsiteURL            string
	UpdatedOn             time.Time
}
