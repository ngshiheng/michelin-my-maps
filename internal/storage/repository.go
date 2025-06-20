package storage

import (
	"context"

	"github.com/ngshiheng/michelin-my-maps/v3/internal/models"
)

// RestaurantRepository defines the interface for restaurant data operations.
type RestaurantRepository interface {
	SaveRestaurant(ctx context.Context, restaurant *models.Restaurant) error
	SaveAward(ctx context.Context, award *models.RestaurantAward) error
	FindAwardByRestaurantAndYear(ctx context.Context, restaurantID uint, year int) (*models.RestaurantAward, error)
	UpdateAward(ctx context.Context, award *models.RestaurantAward) error
	UpsertRestaurantWithAward(ctx context.Context, restaurantData RestaurantData) error
}

// RestaurantData holds the scraped restaurant information.
type RestaurantData struct {
	URL                   string
	Name                  string
	Address               string
	Location              string
	Latitude              string
	Longitude             string
	Cuisine               string
	PhoneNumber           string
	WebsiteURL            string
	Distinction           string
	Description           string
	Price                 string
	FacilitiesAndServices string
	GreenStar             bool
}
