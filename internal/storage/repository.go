package storage

import (
	"context"

	"github.com/ngshiheng/michelin-my-maps/v3/internal/models"
)

// RestaurantRepository defines the interface for restaurant data operations.
type RestaurantRepository interface {
	FindRestaurantByURL(ctx context.Context, url string) (*models.Restaurant, error)
	ListRestaurants(ctx context.Context) ([]models.Restaurant, error)
	SaveAward(ctx context.Context, award *models.RestaurantAward) error
	SaveRestaurant(ctx context.Context, restaurant *models.Restaurant) error
}

// RestaurantData holds the scraped restaurant information.
type RestaurantData struct {
	Address               string
	Cuisine               string
	Description           string
	Distinction           string
	FacilitiesAndServices string
	GreenStar             bool
	Latitude              string
	Location              string
	Longitude             string
	Name                  string
	PhoneNumber           string
	Price                 string
	URL                   string
	WaybackURL            string
	WebsiteURL            string
	Year                  int
}

// RestaurantAwardData holds the Michelin award information for a restaurant.
type RestaurantAwardData struct {
	Distinction   string
	GreenStar     bool
	Price         string
	PublishedDate int
}
