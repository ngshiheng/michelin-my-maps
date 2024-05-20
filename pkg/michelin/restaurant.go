package michelin

import "time"

// Restaurant stores information about a restaurant on Michelin Guide.
type Restaurant struct {
	Address               string
	Cuisine               string
	Description           string
	Distinction           string `gorm:"index:idx_distinction"`
	FacilitiesAndServices string // Comma-separated string
	GreenStar             bool
	Latitude              string
	Location              string
	Longitude             string `gorm:"index:idx_location"`
	Name                  string `gorm:"not null;index:idx_name"`
	PhoneNumber           string
	Price                 string
	URL                   string `gorm:"unique"`
	WebsiteURL            string
	UpdatedOn             time.Time
}
