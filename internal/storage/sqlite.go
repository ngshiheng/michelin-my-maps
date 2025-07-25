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

// SaveRestaurant saves or updates a restaurant in the database.
// If the restaurant already exists (identified by URL), it updates the existing record.
func (r *SQLiteRepository) SaveRestaurant(ctx context.Context, restaurant *models.Restaurant) error {
	log.WithFields(log.Fields{
		"url": restaurant.URL,
	}).Debug("upsert restaurant")

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

// SaveAward saves or updates a restaurant award in the database.
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

	// Scenario 1: Both live scrape
	if existing.WaybackURL == "" && award.WaybackURL == "" {
		if awardsEqual(&existing, award) {
			return nil
		}

		updates := map[string]any{
			"distinction": award.Distinction,
			"price":       award.Price,
			"green_star":  award.GreenStar,
			"wayback_url": award.WaybackURL,
			"year":        award.Year,
		}
		return r.db.WithContext(ctx).
			Model(&existing).
			Updates(updates).Error
	}

	// Scenario 2: Incoming Wayback (authoritative),
	if award.WaybackURL != "" {
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

			if _, ok := diff["distinction"]; ok {
				log.WithFields(log.Fields{
					"restaurant_id": existing.RestaurantID,
					"year":          existing.Year,
					"diff":          diff,
				}).Warn("update award from Wayback")
			}
		}

		updates := map[string]any{
			"distinction": award.Distinction,
			"price":       award.Price,
			"green_star":  award.GreenStar,
			"wayback_url": award.WaybackURL,
			"year":        award.Year,
		}

		return r.db.WithContext(ctx).
			Model(&existing).
			Updates(updates).Error
	}

	// Scenario 3: Existing Wayback, incoming live scrape
	if existing.WaybackURL != "" && award.WaybackURL == "" {
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

			shouldOverride := existing.Distinction == models.SelectedRestaurants && award.Distinction != models.SelectedRestaurants
			if shouldOverride {
				if _, ok := diff["distinction"]; ok {
					diff["distinction"] = fmt.Sprintf("%v → %v", existing.Distinction, award.Distinction)
					log.WithFields(log.Fields{
						"restaurant_id": existing.RestaurantID,
						"year":          existing.Year,
						"diff":          diff,
					}).Warn("update award distinction from Wayback to Live")
				}

				updates := map[string]any{
					"distinction": award.Distinction,
					"price":       award.Price,
					"green_star":  award.GreenStar,
					"wayback_url": award.WaybackURL,
					"year":        award.Year,
				}

				return r.db.WithContext(ctx).
					Model(&existing).
					Updates(updates).Error
			} else {
				log.WithFields(log.Fields{
					"restaurant_id": existing.RestaurantID,
					"year":          existing.Year,
					"diff":          diff,
				}).Debug("award conflict: Wayback vs Live (not overridden)")
			}
		}
		return nil
	}
	return nil
}

func (r *SQLiteRepository) FindRestaurantByURL(ctx context.Context, url string) (*models.Restaurant, error) {
	var restaurant models.Restaurant
	err := r.db.WithContext(ctx).Where("url = ?", url).First(&restaurant).Error
	if err != nil {
		return nil, err
	}
	return &restaurant, nil
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
	}).Debug("retrieve all restaurants with URL")
	return restaurants, nil
}
