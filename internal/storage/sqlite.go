package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ngshiheng/michelin-my-maps/v3/internal/models"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

// SQLiteRepository implements RestaurantRepository using SQLite database.
type SQLiteRepository struct {
	db *gorm.DB
}

// NewSQLiteRepository creates a new SQLite repository instance.
func NewSQLiteRepository(dbPath string) (*SQLiteRepository, error) {
	dsn := fmt.Sprintf("%s?_loc=UTC", dbPath)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		PrepareStmt: true,
		Logger:      logger.Default.LogMode(logger.Silent), // Disable GORM logging
		NowFunc: func() time.Time {
			return time.Now().UTC() // Force UTC timestamps
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get the generic database object sql.DB to use its functions
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database object: %w", err)
	}

	// Set PRAGMA statements for better performance
	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA cache_size = 10000;",
		"PRAGMA temp_store = MEMORY;",
	}

	for _, pragma := range pragmas {
		if _, err := sqlDB.Exec(pragma); err != nil {
			return nil, fmt.Errorf("failed to execute %s: %w", pragma, err)
		}
	}

	// Auto-migrate the Restaurant and RestaurantAward models
	if err := db.AutoMigrate(&models.Restaurant{}, &models.RestaurantAward{}); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate models: %w", err)
	}

	return &SQLiteRepository{db: db}, nil
}

// SaveRestaurant saves a restaurant to the database.
func (r *SQLiteRepository) SaveRestaurant(ctx context.Context, restaurant *models.Restaurant) error {
	log.WithFields(log.Fields{
		"id":  restaurant.ID,
		"url": restaurant.URL,
	}).Debug("upserting restaurant")

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "url"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name", "description", "address", "location",
			"latitude", "longitude", "cuisine",
			"facilities_and_services", "phone_number", "website_url",
			"updated_at",
		}),
	}).Create(restaurant).Error
}

/*
SaveAward saves a restaurant award to the database with strict provenance protection:
  - If no award exists for (restaurant_id, year), insert as new.
  - If an award exists and wayback_url is empty, update fields as needed.
  - If an award exists and wayback_url is non-empty, do not update any fields.
    If the incoming data differs, log a warning.
*/
func (r *SQLiteRepository) SaveAward(ctx context.Context, award *models.RestaurantAward) error {
	awardsEqual := func(a, b *models.RestaurantAward) bool {
		return a.Distinction == b.Distinction &&
			a.Price == b.Price &&
			a.GreenStar == b.GreenStar
	}

	var existing models.RestaurantAward
	err := r.db.WithContext(ctx).
		Where("restaurant_id = ? AND year = ?", award.RestaurantID, award.Year).
		First(&existing).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return r.db.WithContext(ctx).Create(award).Error
		}
		return err
	}

	if existing.WaybackURL == "" {
		if !awardsEqual(&existing, award) {
			diff := map[string]string{}
			if existing.Distinction != award.Distinction {
				diff["distinction"] = fmt.Sprintf("%v → %v", existing.Distinction, award.Distinction)
			}
			if existing.Price != award.Price {
				diff["price"] = fmt.Sprintf("%v → %v", existing.Price, award.Price)
			}
			if existing.GreenStar != award.GreenStar {
				diff["green_star"] = fmt.Sprintf("%v → %v", existing.GreenStar, award.GreenStar)
			}
			if existing.WaybackURL != award.WaybackURL {
				diff["wayback_url"] = fmt.Sprintf("%v → %v", existing.WaybackURL, award.WaybackURL)
			}
			log.WithFields(log.Fields{
				"restaurant_id": existing.RestaurantID,
				"year":          existing.Year,
				"diff":          diff,
			}).Warn("attempted overwrite of award; update skipped due to provenance protection")
			return nil
		}
	}

	existing.WaybackURL = award.WaybackURL
	return r.db.WithContext(ctx).Save(&existing).Error
}

func (r *SQLiteRepository) FindRestaurantByURL(ctx context.Context, url string) (*models.Restaurant, error) {
	var restaurant models.Restaurant
	err := r.db.WithContext(ctx).Where("url = ?", url).First(&restaurant).Error
	if err != nil {
		return nil, err
	}
	return &restaurant, nil
}

/*
UpsertRestaurantWithAward creates or updates a restaurant and its award for the explicit year provided in data.Year.
If data.Year is zero or invalid, the award upsert is skipped and a warning is logged.
*/
func (r *SQLiteRepository) UpsertRestaurantWithAward(ctx context.Context, data RestaurantData) error {
	log.WithFields(log.Fields{
		"url":         data.URL,
		"distinction": data.Distinction,
		"year":        data.Year,
	}).Debug("processing restaurant data")

	restaurant := models.Restaurant{
		URL:                   data.URL,
		Name:                  data.Name,
		Description:           data.Description,
		Address:               data.Address,
		Location:              data.Location,
		Latitude:              data.Latitude,
		Longitude:             data.Longitude,
		Cuisine:               data.Cuisine,
		FacilitiesAndServices: data.FacilitiesAndServices,
		PhoneNumber:           data.PhoneNumber,
		WebsiteURL:            data.WebsiteURL,
	}

	if err := r.SaveRestaurant(ctx, &restaurant); err != nil {
		return fmt.Errorf("failed to save restaurant: %w", err)
	}

	if data.Year <= 0 {
		log.WithFields(log.Fields{
			"url":  data.URL,
			"note": "award year missing or invalid, skipping award upsert",
		}).Warn("Skipping award upsert due to invalid year")
		return nil
	}

	award := &models.RestaurantAward{
		RestaurantID: restaurant.ID,
		Year:         data.Year,
		Distinction:  data.Distinction,
		Price:        data.Price,
		GreenStar:    data.GreenStar,
	}
	return r.SaveAward(ctx, award)
}

// ListAllRestaurantsWithURL retrieves all restaurants that have a non-empty URL.
func (r *SQLiteRepository) ListAllRestaurantsWithURL(ctx context.Context) ([]models.Restaurant, error) {
	var restaurants []models.Restaurant
	err := r.db.WithContext(ctx).Where("url != ''").Find(&restaurants).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list restaurants with URL: %w", err)
	}
	log.WithFields(log.Fields{
		"count": len(restaurants),
	}).Debug("retrieved all restaurants with URL")
	return restaurants, nil
}
