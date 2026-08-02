package storage

import (
	"context"

	"github.com/ngshiheng/michelin-my-maps/v4/internal/models"
)

// RestaurantRepository defines the interface for restaurant data operations.
type RestaurantRepository interface {
	FindRestaurantByURL(ctx context.Context, url string) (*models.Restaurant, error)
	ListRestaurants(ctx context.Context) ([]models.Restaurant, error)
	SaveAward(ctx context.Context, award *models.RestaurantAward) error
	SaveRestaurant(ctx context.Context, restaurant *models.Restaurant) error
}
