package model

// Restaurant stores information about a restaurant on Michelin Guide
type Restaurant struct {
	Name           string
	Address        string
	Price          string
	Type           string
	Latitude       float64
	Longitude      float64
	PhoneNumber    string
	MichelinUrl    string
	WebsiteUrl     string
	Classification string
}
