package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ngshiheng/michelin-my-maps/v3/internal/models"
)

func newTestRepo(t *testing.T) (*SQLiteRepository, func()) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create test repo: %v", err)
	}

	cleanup := func() {
		// nothing to do; t.TempDir() will cleanup
		_ = repo
	}
	return repo, cleanup
}

func validRestaurant() *models.Restaurant {
	return &models.Restaurant{
		URL:                   "https://guide.michelin.com/test/1",
		Address:               "1 Test St",
		Cuisine:               "Test Cuisine",
		Description:           "A test restaurant",
		FacilitiesAndServices: "",
		Latitude:              "12.34",
		Longitude:             "56.78",
		Location:              "Test City",
		Name:                  "Test Resto",
		PhoneNumber:           "",
		WebsiteURL:            "",
	}
}

func TestSQLiteRepository(t *testing.T) {
	ctx := context.Background()

	t.Run("NewSQLiteRepository creates DB and runs migrations", func(t *testing.T) {
		repo, cleanup := newTestRepo(t)
		defer cleanup()

		if repo == nil {
			t.Fatalf("expected repo, got nil")
		}

		if ok := repo.db.Migrator().HasTable(&models.Restaurant{}); !ok {
			t.Fatalf("restaurants table not present")
		}
		if ok := repo.db.Migrator().HasTable(&models.RestaurantAward{}); !ok {
			t.Fatalf("restaurant_awards table not present")
		}
	})

	t.Run("SaveAward validation errors for invalid inputs", func(t *testing.T) {
		repo, _ := newTestRepo(t)

		r := validRestaurant()
		if err := repo.SaveRestaurant(ctx, r); err != nil {
			t.Fatalf("SaveRestaurant setup failed: %v", err)
		}
		created, _ := repo.FindRestaurantByURL(ctx, r.URL)

		cases := []struct {
			name  string
			award *models.RestaurantAward
		}{
			{"missing restaurant id", &models.RestaurantAward{RestaurantID: 0, Distinction: models.OneStar, GreenStar: false, Price: "$$", Year: time.Now().Year(), WaybackURL: ""}},
			{"invalid distinction", &models.RestaurantAward{RestaurantID: created.ID, Distinction: "NotAThing", GreenStar: false, Price: "$$", Year: time.Now().Year(), WaybackURL: ""}},
			{"empty price", &models.RestaurantAward{RestaurantID: created.ID, Distinction: models.OneStar, GreenStar: false, Price: "", Year: time.Now().Year(), WaybackURL: ""}},
			{"invalid year", &models.RestaurantAward{RestaurantID: created.ID, Distinction: models.OneStar, GreenStar: false, Price: "$$", Year: 1800, WaybackURL: ""}},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				err := repo.SaveAward(ctx, tc.award)
				if err == nil {
					t.Fatalf("expected validation error for case %q, got nil", tc.name)
				}
			})
		}
	})

	t.Run("SaveAward incoming live should overwrite price and distinction", func(t *testing.T) {
		repo, _ := newTestRepo(t)

		r := validRestaurant()
		if err := repo.SaveRestaurant(ctx, r); err != nil {
			t.Fatalf("SaveRestaurant setup failed: %v", err)
		}
		created, _ := repo.FindRestaurantByURL(ctx, r.URL)

		ex := &models.RestaurantAward{
			RestaurantID: created.ID,
			Distinction:  models.OneStar,
			GreenStar:    false,
			Price:        "$$",
			Year:         time.Now().Year(),
			WaybackURL:   "",
		}
		if err := repo.SaveAward(ctx, ex); err != nil {
			t.Fatalf("SaveAward insert existing failed: %v", err)
		}

		inc := &models.RestaurantAward{
			RestaurantID: created.ID,
			Distinction:  models.TwoStars,
			GreenStar:    false,
			Price:        "$$$",
			Year:         ex.Year,
			WaybackURL:   "",
		}
		if err := repo.SaveAward(ctx, inc); err != nil {
			t.Fatalf("SaveAward update failed: %v", err)
		}

		var got models.RestaurantAward
		if err := repo.db.WithContext(ctx).Where("restaurant_id = ? AND year = ?", created.ID, ex.Year).First(&got).Error; err != nil {
			t.Fatalf("query award failed: %v", err)
		}
		if got.Price != "$$$" || got.Distinction != models.TwoStars {
			t.Fatalf("expected updated price and distinction, got %+v", got)
		}
	})

	t.Run("SaveAward should preserve existing price when incoming price empty", func(t *testing.T) {
		repo, _ := newTestRepo(t)

		r := validRestaurant()
		if err := repo.SaveRestaurant(ctx, r); err != nil {
			t.Fatalf("SaveRestaurant setup failed: %v", err)
		}
		created, _ := repo.FindRestaurantByURL(ctx, r.URL)

		ex := &models.RestaurantAward{
			RestaurantID: created.ID,
			Distinction:  models.BibGourmand,
			GreenStar:    false,
			Price:        "$$",
			Year:         time.Now().Year(),
			WaybackURL:   "",
		}
		if err := repo.SaveAward(ctx, ex); err != nil {
			t.Fatalf("SaveAward insert failed: %v", err)
		}

		inc := &models.RestaurantAward{
			RestaurantID: created.ID,
			Distinction:  models.BibGourmand,
			GreenStar:    false,
			Price:        "",
			Year:         ex.Year,
			WaybackURL:   "",
		}
		if err := repo.SaveAward(ctx, inc); err != nil {
			t.Fatalf("SaveAward update failed: %v", err)
		}

		var got models.RestaurantAward
		if err := repo.db.WithContext(ctx).Where("restaurant_id = ? AND year = ?", created.ID, ex.Year).First(&got).Error; err != nil {
			t.Fatalf("query award failed: %v", err)
		}
		if got.Price != "$$" {
			t.Fatalf("expected price to be preserved, got %q", got.Price)
		}
	})

	t.Run("SaveAward incoming wayback sets wayback_url", func(t *testing.T) {
		repo, _ := newTestRepo(t)

		r := validRestaurant()
		if err := repo.SaveRestaurant(ctx, r); err != nil {
			t.Fatalf("SaveRestaurant setup failed: %v", err)
		}
		created, _ := repo.FindRestaurantByURL(ctx, r.URL)

		ex := &models.RestaurantAward{
			RestaurantID: created.ID,
			Distinction:  models.OneStar,
			GreenStar:    false,
			Price:        "$$",
			Year:         time.Now().Year(),
			WaybackURL:   "",
		}
		if err := repo.SaveAward(ctx, ex); err != nil {
			t.Fatalf("SaveAward insert failed: %v", err)
		}

		inc := &models.RestaurantAward{
			RestaurantID: created.ID,
			Distinction:  ex.Distinction,
			GreenStar:    ex.GreenStar,
			Price:        ex.Price,
			Year:         ex.Year,
			WaybackURL:   "https://web.archive.org/same",
		}
		if err := repo.SaveAward(ctx, inc); err != nil {
			t.Fatalf("SaveAward wayback insert failed: %v", err)
		}

		var got models.RestaurantAward
		if err := repo.db.WithContext(ctx).Where("restaurant_id = ? AND year = ?", created.ID, ex.Year).First(&got).Error; err != nil {
			t.Fatalf("query award failed: %v", err)
		}
		if got.WaybackURL == "" {
			t.Fatalf("expected wayback_url to be set, got empty")
		}
	})

	t.Run("SaveAward wayback should cause incoming live to be skipped when shouldOverride is false", func(t *testing.T) {
		repo, _ := newTestRepo(t)

		r := validRestaurant()
		if err := repo.SaveRestaurant(ctx, r); err != nil {
			t.Fatalf("SaveRestaurant setup failed: %v", err)
		}
		created, _ := repo.FindRestaurantByURL(ctx, r.URL)

		seed := &models.RestaurantAward{
			RestaurantID: created.ID,
			Distinction:  models.OneStar,
			GreenStar:    false,
			Price:        "$$",
			Year:         time.Now().Year(),
			WaybackURL:   "https://web.archive.org/orig",
		}
		if err := repo.db.WithContext(ctx).Create(seed).Error; err != nil {
			t.Fatalf("failed to seed award: %v", err)
		}

		incoming := &models.RestaurantAward{
			RestaurantID: created.ID,
			Distinction:  models.TwoStars,
			GreenStar:    true,
			Price:        "$$$",
			Year:         seed.Year,
			WaybackURL:   "",
		}
		if err := repo.SaveAward(ctx, incoming); err != nil {
			t.Fatalf("SaveAward should skip update but returned error: %v", err)
		}

		var got models.RestaurantAward
		if err := repo.db.WithContext(ctx).Where("restaurant_id = ? AND year = ?", created.ID, seed.Year).First(&got).Error; err != nil {
			t.Fatalf("query award failed: %v", err)
		}
		if got.Distinction != seed.Distinction || got.Price != seed.Price {
			t.Fatalf("expected award to be unchanged, got %+v", got)
		}
	})

	t.Run("SaveAward overrides existing wayback when shouldOverride is true", func(t *testing.T) {
		repo, _ := newTestRepo(t)

		r := validRestaurant()
		if err := repo.SaveRestaurant(ctx, r); err != nil {
			t.Fatalf("SaveRestaurant setup failed: %v", err)
		}
		created, _ := repo.FindRestaurantByURL(ctx, r.URL)

		seed := &models.RestaurantAward{
			RestaurantID: created.ID,
			Distinction:  models.SelectedRestaurants,
			GreenStar:    false,
			Price:        "$$",
			Year:         time.Now().Year(),
			WaybackURL:   "https://web.archive.org/orig",
		}
		if err := repo.db.WithContext(ctx).Create(seed).Error; err != nil {
			t.Fatalf("failed to seed award: %v", err)
		}

		incoming := &models.RestaurantAward{
			RestaurantID: created.ID,
			Distinction:  models.TwoStars,
			GreenStar:    true,
			Price:        "$$$",
			Year:         seed.Year,
			WaybackURL:   "",
			// assume ShouldOverride is expressed via Distinction/WaybackURL logic inside SaveAward
		}

		if err := repo.SaveAward(ctx, incoming); err != nil {
			t.Fatalf("SaveAward update failed: %v", err)
		}

		var got models.RestaurantAward
		if err := repo.db.WithContext(ctx).Where("restaurant_id = ? AND year = ?", created.ID, seed.Year).First(&got).Error; err != nil {
			t.Fatalf("query award failed: %v", err)
		}
		if got.Distinction != incoming.Distinction || got.Price != incoming.Price {
			t.Fatalf("expected award to be updated, got %+v", got)
		}
	})

	t.Run("SaveAward should return error for non-existent restaurant id", func(t *testing.T) {
		repo, _ := newTestRepo(t)

		// choose a high id that should not exist
		bad := &models.RestaurantAward{
			RestaurantID: 999999,
			Distinction:  models.OneStar,
			GreenStar:    false,
			Price:        "$$",
			Year:         time.Now().Year(),
			WaybackURL:   "",
		}
		if err := repo.SaveAward(ctx, bad); err == nil {
			t.Fatalf("expected error when saving award for non-existent restaurant id, got nil")
		}
	})

	t.Run("SaveAward should reject future year", func(t *testing.T) {
		repo, _ := newTestRepo(t)

		r := validRestaurant()
		if err := repo.SaveRestaurant(ctx, r); err != nil {
			t.Fatalf("SaveRestaurant setup failed: %v", err)
		}
		created, _ := repo.FindRestaurantByURL(ctx, r.URL)

		future := &models.RestaurantAward{
			RestaurantID: created.ID,
			Distinction:  models.OneStar,
			GreenStar:    false,
			Price:        "$$",
			Year:         time.Now().Year() + 5,
			WaybackURL:   "",
		}
		if err := repo.SaveAward(ctx, future); err == nil {
			t.Fatalf("expected validation error for future year, got nil")
		}
	})

	t.Run("SaveRestaurant insert and upsert; FindRestaurantByURL not-found", func(t *testing.T) {
		repo, _ := newTestRepo(t)

		r := validRestaurant()
		if err := repo.SaveRestaurant(ctx, r); err != nil {
			t.Fatalf("SaveRestaurant insert failed: %v", err)
		}

		got, err := repo.FindRestaurantByURL(ctx, r.URL)
		if err != nil {
			t.Fatalf("FindRestaurantByURL failed: %v", err)
		}
		if got.Name != r.Name {
			t.Fatalf("expected name %q, got %q", r.Name, got.Name)
		}

		// upsert: change name
		prevUpdated := got.UpdatedAt
		r.Name = "Updated Resto"
		if err := repo.SaveRestaurant(ctx, r); err != nil {
			t.Fatalf("SaveRestaurant upsert failed: %v", err)
		}

		got2, err := repo.FindRestaurantByURL(ctx, r.URL)
		if err != nil {
			t.Fatalf("FindRestaurantByURL after upsert failed: %v", err)
		}
		if got2.Name != "Updated Resto" {
			t.Fatalf("expected updated name, got %q", got2.Name)
		}
		if !got2.UpdatedAt.After(prevUpdated) && !got2.UpdatedAt.Equal(prevUpdated) {
			t.Fatalf("expected UpdatedAt to be updated (or equal), prev=%v now=%v", prevUpdated, got2.UpdatedAt)
		}

		if _, err := repo.FindRestaurantByURL(ctx, "https://no-such-url.example/"); err == nil {
			t.Fatalf("expected error for nonexistent restaurant URL, got nil")
		}
	})
}
