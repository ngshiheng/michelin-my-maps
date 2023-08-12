package michelin

// Restaurant stores information about a restaurant on Michelin Guide.
type Restaurant struct {
	Address               string
	Cuisine               string
	Distinction           string
	FacilitiesAndServices string
	Latitude              string
	Location              string
	Longitude             string
	Name                  string `gorm:"not null"`
	PhoneNumber           string
	Price                 string
	URL                   string `gorm:"unique"`
	WebsiteURL            string
}
