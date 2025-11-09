package trip

import (
	"context"
	"strings"

	"github.com/ngshiheng/michelin-my-maps/v3/internal/models"
	"gorm.io/gorm"
)

// RestaurantWithAward combines restaurant and award information
type RestaurantWithAward struct {
	models.Restaurant
	Award models.RestaurantAward
}

// QueryRestaurants queries restaurants based on trip filters
func (m *Manager) QueryRestaurants(ctx context.Context, db *gorm.DB, trip *Trip) ([]RestaurantWithAward, error) {
	query := db.WithContext(ctx).
		Table("restaurants").
		Select("restaurants.*, restaurant_awards.*").
		Joins("JOIN restaurant_awards ON restaurants.id = restaurant_awards.restaurant_id").
		Where("restaurant_awards.year = (SELECT MAX(year) FROM restaurant_awards WHERE restaurant_id = restaurants.id)")

	// Filter by cities
	if len(trip.Cities) > 0 {
		cityConditions := make([]string, len(trip.Cities))
		cityArgs := make([]interface{}, len(trip.Cities))
		for i, city := range trip.Cities {
			cityConditions[i] = "restaurants.location LIKE ?"
			cityArgs[i] = "%" + city + "%"
		}
		query = query.Where(strings.Join(cityConditions, " OR "), cityArgs...)
	}

	// Filter by distinctions
	if len(trip.Distinctions) > 0 {
		query = query.Where("restaurant_awards.distinction IN ?", trip.Distinctions)
	}

	// Filter by minimum stars
	if trip.MinStars > 0 {
		starFilters := []string{}
		if trip.MinStars <= 1 {
			starFilters = append(starFilters, models.OneStar, models.TwoStars, models.ThreeStars)
		} else if trip.MinStars == 2 {
			starFilters = append(starFilters, models.TwoStars, models.ThreeStars)
		} else if trip.MinStars == 3 {
			starFilters = append(starFilters, models.ThreeStars)
		}
		query = query.Where("restaurant_awards.distinction IN ?", starFilters)
	}

	// Filter by cuisines
	if len(trip.Cuisines) > 0 {
		cuisineConditions := make([]string, len(trip.Cuisines))
		cuisineArgs := make([]interface{}, len(trip.Cuisines))
		for i, cuisine := range trip.Cuisines {
			cuisineConditions[i] = "restaurants.cuisine LIKE ?"
			cuisineArgs[i] = "%" + cuisine + "%"
		}
		query = query.Where(strings.Join(cuisineConditions, " OR "), cuisineArgs...)
	}

	// Filter by green star
	if trip.GreenStarOnly {
		query = query.Where("restaurant_awards.green_star = ?", true)
	}

	// Execute query
	var results []struct {
		models.Restaurant
		AwardID          uint   `gorm:"column:id"`
		AwardRestaurantID uint  `gorm:"column:restaurant_id"`
		AwardDistinction string `gorm:"column:distinction"`
		AwardGreenStar   bool   `gorm:"column:green_star"`
		AwardPrice       string `gorm:"column:price"`
		AwardYear        int    `gorm:"column:year"`
		AwardWaybackURL  string `gorm:"column:wayback_url"`
	}

	if err := query.Find(&results).Error; err != nil {
		return nil, err
	}

	// Convert results to RestaurantWithAward
	restaurants := make([]RestaurantWithAward, len(results))
	for i, r := range results {
		restaurants[i] = RestaurantWithAward{
			Restaurant: r.Restaurant,
			Award: models.RestaurantAward{
				ID:           r.AwardID,
				RestaurantID: r.AwardRestaurantID,
				Distinction:  r.AwardDistinction,
				GreenStar:    r.AwardGreenStar,
				Price:        r.AwardPrice,
				Year:         r.AwardYear,
				WaybackURL:   r.AwardWaybackURL,
			},
		}
	}

	return restaurants, nil
}
