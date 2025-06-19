package storage

import (
	"context"
	"errors"
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
	log.WithFields(log.Fields{
		"name": restaurant.Name,
		"url":  restaurant.URL,
	}).Debug("upserting restaurant")

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "url"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name", "description", "address", "location",
			"latitude", "longitude", "cuisine",
			"facilities_and_services", "phone_number", "website_url",
		}),
	}).Create(restaurant).Error
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
	log.WithFields(log.Fields{
		"restaurant":  data.Name,
		"distinction": data.Distinction,
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

	return r.processRestaurantAwardChanges(ctx, &restaurant, data)
}

// processRestaurantAwardChanges handles award upsert with temporal assumptions:
// CAVEAT: Assumes current year = current award year (ignores publication cycles)
// RISK: May create incorrect historical records via backdating
func (r *SQLiteRepository) processRestaurantAwardChanges(ctx context.Context, restaurant *models.Restaurant, data RestaurantData) error {
	currentYear := time.Now().Year()

	currentYearAward, err := r.FindAwardByRestaurantAndYear(ctx, restaurant.ID, currentYear)
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		// No award exists for current year - create new one
		//
		// NOTE:
		//		May create incorrect current year awards
		// 		E.g.: Les Amis 3★ (2025) → scraped Jan 1, 2026 → creates 3★ (2026) even though 2026 guide isn't published yet
		// 		Here, I chose to accept temporary incorrect state, let system self-correct
		// 		However, if restaurant is completely not found at source, this will be wrong perpetually
		return r.createNewAward(ctx, restaurant, data, currentYear)
	case err == nil:
		// Award exists - check for changes
		return r.handleExistingAward(ctx, currentYearAward, restaurant, data, currentYear)
	default:
		return fmt.Errorf("failed to find existing award: %w", err)
	}
}

// createNewAward creates award for current year
// WARNING: May be incorrect if scraped before official guide publication
func (r *SQLiteRepository) createNewAward(ctx context.Context, restaurant *models.Restaurant, data RestaurantData, currentYear int) error {
	log.WithFields(log.Fields{
		"id":          restaurant.ID,
		"name":        data.Name,
		"distinction": data.Distinction,
		"url":         data.URL,
	}).Info("✓ new restaurant award")

	newAward := models.RestaurantAward{
		RestaurantID: restaurant.ID,
		Year:         currentYear,
		Distinction:  data.Distinction,
		Price:        data.Price,
		GreenStar:    data.GreenStar,
	}
	return r.SaveAward(ctx, &newAward)
}

// handleExistingAward processes an existing award of the year, checking for changes
func (r *SQLiteRepository) handleExistingAward(ctx context.Context, existingAward *models.RestaurantAward, restaurant *models.Restaurant, data RestaurantData, currentYear int) error {
	hasAwardChange := existingAward.Distinction != data.Distinction ||
		existingAward.Price != data.Price ||
		existingAward.GreenStar != data.GreenStar
	if !hasAwardChange {
		log.WithFields(log.Fields{
			"id":          restaurant.ID,
			"name":        data.Name,
			"distinction": data.Distinction,
			"url":         data.URL,
		}).Debug("restaurant award unchanged, skipping")
		return nil
	}

	return r.handleAwardChange(ctx, existingAward, restaurant.ID, data, currentYear)
}

// handleAwardChange implements backdating logic with data integrity trade-offs:
// - If prev year exists: assume current award was already correct → update in place
// - If prev year missing: assume current award was actually for prev year → backdate + create new
// LIMITATION: Creates phantom historical records that may never have existed
func (r *SQLiteRepository) handleAwardChange(ctx context.Context, existingAward *models.RestaurantAward, restaurantID uint, data RestaurantData, currentYear int) error {
	previousYear := currentYear - 1

	_, err := r.FindAwardByRestaurantAndYear(ctx, restaurantID, previousYear)
	switch {
	case err == nil:
		// Award already exists for previous year - update current award with new data
		existingAward.Distinction = data.Distinction
		existingAward.Price = data.Price
		existingAward.GreenStar = data.GreenStar
		return r.UpdateAward(ctx, existingAward)
	case errors.Is(err, gorm.ErrRecordNotFound):
		// Safe to backdate - update existing award to previous year
		return r.backdateAndCreateAward(ctx, existingAward, restaurantID, data, currentYear, previousYear)
	default:
		return fmt.Errorf("failed to check for previous year conflict: %w", err)
	}
}

// backdateAndCreateAward moves existing award to previous year and creates new current award
// ASSUMPTION: Existing award was incorrectly dated and belongs to previous year
// RISK: Creates false historical record if assumption is wrong
func (r *SQLiteRepository) backdateAndCreateAward(ctx context.Context, existingAward *models.RestaurantAward, restaurantID uint, data RestaurantData, currentYear, previousYear int) error {
	// Safe to backdate - update existing award to previous year
	existingAward.Year = previousYear
	if err := r.UpdateAward(ctx, existingAward); err != nil {
		return fmt.Errorf("failed to backdate existing award: %w", err)
	}

	log.WithFields(log.Fields{
		"id":          restaurantID,
		"name":        data.Name,
		"from_year":   currentYear,
		"to_year":     previousYear,
		"distinction": existingAward.Distinction,
		"url":         data.URL,
	}).Info("↩ backdated award")

	// Create new award for current year
	newAward := models.RestaurantAward{
		RestaurantID: restaurantID,
		Year:         currentYear,
		Distinction:  data.Distinction,
		Price:        data.Price,
		GreenStar:    data.GreenStar,
	}

	log.WithFields(log.Fields{
		"id":       restaurantID,
		"name":     data.Name,
		"year":     currentYear,
		"previous": existingAward.Distinction,
		"new":      data.Distinction,
		"url":      data.URL,
	}).Info("✓ created new award - distinction changed")

	return r.SaveAward(ctx, &newAward)
}

// Close closes the database connection.
func (r *SQLiteRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
