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
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		PrepareStmt: true,
		Logger:      logger.Default.LogMode(logger.Silent), // Disable GORM logging
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
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "url"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name", "description", "address", "location",
			"latitude", "longitude", "cuisine",
			"facilities_and_services", "phone_number", "website_url",
		}),
	}).Create(restaurant).Error
}

// FindRestaurantByURL finds a restaurant by its URL.
func (r *SQLiteRepository) FindRestaurantByURL(ctx context.Context, url string) (*models.Restaurant, error) {
	var restaurant models.Restaurant
	err := r.db.WithContext(ctx).Where("url = ?", url).First(&restaurant).Error
	if err != nil {
		return nil, err
	}
	return &restaurant, nil
}

// SaveAward saves an award to the database.
func (r *SQLiteRepository) SaveAward(ctx context.Context, award *models.RestaurantAward) error {
	return r.db.WithContext(ctx).Create(award).Error
}

// FindAwardByRestaurantAndYear finds an award by restaurant ID and year.
func (r *SQLiteRepository) FindAwardByRestaurantAndYear(ctx context.Context, restaurantID uint, year int) (*models.RestaurantAward, error) {
	var award models.RestaurantAward
	err := r.db.WithContext(ctx).Where("restaurant_id = ? AND year = ?", restaurantID, year).First(&award).Error
	if err != nil {
		return nil, err
	}
	return &award, nil
}

// UpdateAward updates an existing award.
func (r *SQLiteRepository) UpdateAward(ctx context.Context, award *models.RestaurantAward) error {
	return r.db.WithContext(ctx).Save(award).Error
}

// UpsertRestaurantWithAward creates or updates a restaurant and its award with simplified change detection.
func (r *SQLiteRepository) UpsertRestaurantWithAward(ctx context.Context, data RestaurantData) error {
	currentYear := time.Now().Year()

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

	return r.handleAwardUpsert(ctx, &restaurant, data, currentYear)
}

// handleAwardUpsert handles the complex award upsert logic with change detection.
func (r *SQLiteRepository) handleAwardUpsert(ctx context.Context, restaurant *models.Restaurant, data RestaurantData, currentYear int) error {
	existingAward, err := r.FindAwardByRestaurantAndYear(ctx, restaurant.ID, currentYear)
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to find existing award: %w", err)
	}

	// No existing award for current year - create new one
	if err == gorm.ErrRecordNotFound {
		newAward := models.RestaurantAward{
			RestaurantID: restaurant.ID,
			Year:         currentYear,
			Distinction:  data.Distinction,
			Price:        data.Price,
			GreenStar:    data.GreenStar,
		}

		log.WithFields(log.Fields{
			"restaurant_id": restaurant.ID,
			"year":          currentYear,
			"distinction":   data.Distinction,
		}).Info("creating new award")

		return r.SaveAward(ctx, &newAward)
	}

	// Existing award found - check for changes
	hasAwardChanged := existingAward.Distinction != data.Distinction ||
		existingAward.Price != data.Price ||
		existingAward.GreenStar != data.GreenStar
	if hasAwardChanged {
		return r.handleAwardChange(ctx, existingAward, restaurant.ID, data, currentYear)
	}

	// No changes detected
	return nil
}

// handleAwardChange handles the logic when an award change is detected.
func (r *SQLiteRepository) handleAwardChange(ctx context.Context, existingAward *models.RestaurantAward, restaurantID uint, data RestaurantData, currentYear int) error {
	timeSinceUpdate := time.Since(existingAward.UpdatedAt)
	const changeThreshold = 24 * time.Hour

	if timeSinceUpdate > changeThreshold {
		// Significant time has passed - likely a real award change
		return r.handleSignificantAwardChange(ctx, existingAward, restaurantID, data, currentYear)
	} else {
		// Recent change - likely a correction
		log.WithFields(log.Fields{
			"restaurant_id":   restaurantID,
			"old_distinction": existingAward.Distinction,
			"new_distinction": data.Distinction,
			"hours_since":     timeSinceUpdate.Hours(),
		}).Info("recent change detected, updating existing award")
	}

	// Update existing award with new data
	existingAward.Distinction = data.Distinction
	existingAward.Price = data.Price
	existingAward.GreenStar = data.GreenStar
	return r.UpdateAward(ctx, existingAward)
}

// handleSignificantAwardChange handles award changes that occurred after significant time.
func (r *SQLiteRepository) handleSignificantAwardChange(ctx context.Context, existingAward *models.RestaurantAward, restaurantID uint, data RestaurantData, currentYear int) error {
	previousYear := currentYear - 1

	// Check if previous year already exists to avoid conflicts
	_, err := r.FindAwardByRestaurantAndYear(ctx, restaurantID, previousYear)
	if err == gorm.ErrRecordNotFound {
		// Safe to backdate - update existing award to previous year
		existingAward.Year = previousYear
		if err := r.UpdateAward(ctx, existingAward); err != nil {
			return fmt.Errorf("failed to backdate existing award: %w", err)
		}

		log.WithFields(log.Fields{
			"restaurant_id":   restaurantID,
			"old_distinction": existingAward.Distinction,
			"new_distinction": data.Distinction,
			"backdated_year":  previousYear,
			"current_year":    currentYear,
		}).Info("backdated existing award and creating new one")

		// Create new award for current year
		newAward := models.RestaurantAward{
			RestaurantID: restaurantID,
			Year:         currentYear,
			Distinction:  data.Distinction,
			Price:        data.Price,
			GreenStar:    data.GreenStar,
		}

		return r.SaveAward(ctx, &newAward)
	} else if err != nil {
		return fmt.Errorf("failed to check for previous year conflict: %w", err)
	} else {
		// Conflict exists - just update the current year award
		log.WithFields(log.Fields{
			"restaurant_id": restaurantID,
			"conflict_year": previousYear,
			"current_year":  currentYear,
		}).Warn("cannot backdate due to year conflict, updating current award")

		// Update existing award with new data
		existingAward.Distinction = data.Distinction
		existingAward.Price = data.Price
		existingAward.GreenStar = data.GreenStar
		return r.UpdateAward(ctx, existingAward)
	}
}

// Close closes the database connection.
func (r *SQLiteRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
