package michelin

// Restaurant stores information about a restaurant on Michelin Guide.
type Restaurant struct {
	Name                  string `gorm:"not null"`
	Address               string
	Location              string
	Price                 string
	Cuisine               string
	Longitude             string
	Latitude              string
	PhoneNumber           string
	URL                   string `gorm:"unique"`
	WebsiteURL            string
	Award                 string
	FacilitiesAndServices string
}
