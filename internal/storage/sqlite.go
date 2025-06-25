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
SaveAward upserts an award for (restaurant_id, year).
If a record exists, it updates distinction, price, greenstar, and updated_at.
*/
func (r *SQLiteRepository) SaveAward(ctx context.Context, award *models.RestaurantAward) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "restaurant_id"}, {Name: "year"}},
		DoUpdates: clause.AssignmentColumns([]string{"distinction", "price", "green_star", "updated_at"}),
	}).Create(award).Error
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

// Keep only one ListAllRestaurantsWithURL implementation
func (r *SQLiteRepository) ListAllRestaurantsWithURL() ([]models.Restaurant, error) {
	var restaurants []models.Restaurant
	err := r.db.Where("url != ''").Find(&restaurants).Error
	return restaurants, err
}

// Close closes the database connection.
func (r *SQLiteRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
