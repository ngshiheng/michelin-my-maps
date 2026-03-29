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

func validAward(restaurantID uint, year int) *models.RestaurantAward {
	return &models.RestaurantAward{
		WaybackURL:   "",
		RestaurantID: restaurantID,
		Distinction:  models.SelectedRestaurants,
		GreenStar:    false,
		Price:        "$$",
		Year:         year,
	}
}

func TestSQLiteRepository(t *testing.T) {
	ctx := context.Background()

	t.Run("NewSQLiteRepository creates DB and tables", func(t *testing.T) {
		repo, cleanup := newTestRepo(t)
		defer cleanup()

		if repo == nil {
			t.Fatalf("expected repo, got nil")
		}

		// ensure tables exist
		if ok := repo.db.Migrator().HasTable(&models.Restaurant{}); !ok {
			t.Fatalf("restaurants table not present")
		}
		if ok := repo.db.Migrator().HasTable(&models.RestaurantAward{}); !ok {
			t.Fatalf("restaurant_awards table not present")
		}
	})
	t.Run("NewSQLiteRepository should create DB and run migrations", func(t *testing.T) {
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

		// create restaurant for valid references
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

	t.Run("SaveAward both-live differing updates (price handling)", func(t *testing.T) {
		repo, _ := newTestRepo(t)

		// create restaurant
		r := validRestaurant()
		if err := repo.SaveRestaurant(ctx, r); err != nil {
			t.Fatalf("SaveRestaurant setup failed: %v", err)
		}
		created, _ := repo.FindRestaurantByURL(ctx, r.URL)

		// existing award (live)
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

		// incoming live with different price should overwrite
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
	t.Run("SaveAward should update price and distinction when both live and different", func(t *testing.T) {
		repo, _ := newTestRepo(t)

		r := validRestaurant()
		if err := repo.SaveRestaurant(ctx, r); err != nil {
			t.Fatalf("SaveRestaurant setup failed: %v", err)
		}
		created, _ := repo.FindRestaurantByURL(ctx, r.URL)

		existing := &models.RestaurantAward{
			RestaurantID: created.ID,
			Distinction:  models.OneStar,
			GreenStar:    false,
			Price:        "$$",
			Year:         time.Now().Year(),
			WaybackURL:   "",
		}
		if err := repo.SaveAward(ctx, existing); err != nil {
			t.Fatalf("SaveAward insert existing failed: %v", err)
		}

		incoming := &models.RestaurantAward{
			RestaurantID: created.ID,
			Distinction:  models.TwoStars,
			GreenStar:    false,
			Price:        "$$$",
			Year:         existing.Year,
			WaybackURL:   "",
		}
		if err := repo.SaveAward(ctx, incoming); err != nil {
			t.Fatalf("SaveAward update failed: %v", err)
		}

		var got models.RestaurantAward
		if err := repo.db.WithContext(ctx).
			Where("restaurant_id = ? AND year = ?", created.ID, existing.Year).
			First(&got).Error; err != nil {
			t.Fatalf("query award failed: %v", err)
		}
		if got.Price != "$$$" || got.Distinction != models.TwoStars {
			t.Fatalf("expected updated price and distinction, got %+v", got)
		}
	})

	t.Run("SaveAward preserves price when incoming price empty", func(t *testing.T) {
		repo, _ := newTestRepo(t)

		// create restaurant and existing award with price
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

		// incoming live with empty price should not overwrite price
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
	t.Run("SaveAward should preserve existing price when incoming price is empty", func(t *testing.T) {
		repo, _ := newTestRepo(t)

		r := validRestaurant()
		if err := repo.SaveRestaurant(ctx, r); err != nil {
			t.Fatalf("SaveRestaurant setup failed: %v", err)
		}
		created, _ := repo.FindRestaurantByURL(ctx, r.URL)

		existing := &models.RestaurantAward{
			RestaurantID: created.ID,
			Distinction:  models.BibGourmand,
			GreenStar:    false,
			Price:        "$$",
			Year:         time.Now().Year(),
			WaybackURL:   "",
		}
		if err := repo.SaveAward(ctx, existing); err != nil {
			t.Fatalf("SaveAward insert failed: %v", err)
		}

		incoming := &models.RestaurantAward{
			RestaurantID: created.ID,
			Distinction:  models.BibGourmand,
			GreenStar:    false,
			Price:        "",
			Year:         existing.Year,
			WaybackURL:   "",
		}
		if err := repo.SaveAward(ctx, incoming); err != nil {
			t.Fatalf("SaveAward update failed: %v", err)
		}

		var got models.RestaurantAward
		if err := repo.db.WithContext(ctx).
			Where("restaurant_id = ? AND year = ?", created.ID, existing.Year).
			First(&got).Error; err != nil {
			t.Fatalf("query award failed: %v", err)
		}
		if got.Price != "$$" {
			t.Fatalf("expected price to be preserved, got %q", got.Price)
		}
	})

	t.Run("SaveAward incoming wayback overwrites even if identical", func(t *testing.T) {
		repo, _ := newTestRepo(t)

		// create restaurant and existing live award
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
	t.Run("SaveAward should set wayback_url when incoming wayback is authoritative", func(t *testing.T) {
		repo, _ := newTestRepo(t)

		r := validRestaurant()
		if err := repo.SaveRestaurant(ctx, r); err != nil {
			t.Fatalf("SaveRestaurant setup failed: %v", err)
		}
		created, _ := repo.FindRestaurantByURL(ctx, r.URL)

		existing := &models.RestaurantAward{
			RestaurantID: created.ID,
			Distinction:  models.OneStar,
			GreenStar:    false,
			Price:        "$$",
			Year:         time.Now().Year(),
			WaybackURL:   "",
		}
		if err := repo.SaveAward(ctx, existing); err != nil {
			t.Fatalf("SaveAward insert failed: %v", err)
		}

		incoming := &models.RestaurantAward{
			RestaurantID: created.ID,
			Distinction:  existing.Distinction,
			GreenStar:    existing.GreenStar,
			Price:        existing.Price,
			Year:         existing.Year,
			WaybackURL:   "https://web.archive.org/same",
		}
		if err := repo.SaveAward(ctx, incoming); err != nil {
			t.Fatalf("SaveAward wayback insert failed: %v", err)
		}

		var got models.RestaurantAward
		if err := repo.db.WithContext(ctx).
			Where("restaurant_id = ? AND year = ?", created.ID, existing.Year).
			First(&got).Error; err != nil {
			t.Fatalf("query award failed: %v", err)
		}
		if got.WaybackURL == "" {
			t.Fatalf("expected wayback_url to be set, got empty")
		}
	})

	t.Run("SaveAward existing wayback + incoming live skips when shouldOverride is false", func(t *testing.T) {
		repo, _ := newTestRepo(t)

		// create restaurant and existing award with wayback and non-SelectedRestaurants distinction
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
	t.Run("SaveAward should skip update when existing wayback present and shouldOverride is false", func(t *testing.T) {
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
		if err := repo.db.WithContext(ctx).
			Where("restaurant_id = ? AND year = ?", created.ID, seed.Year).
			First(&got).Error; err != nil {
			t.Fatalf("query award failed: %v", err)
		}
		if got.Distinction != seed.Distinction || got.Price != seed.Price {
			t.Fatalf("expected award to be unchanged, got %+v", got)
		}
	})

	t.Run("SaveRestaurant insert and upsert", func(t *testing.T) {
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
	})

	t.Run("SaveAward create and branches", func(t *testing.T) {
		repo, _ := newTestRepo(t)

		// create restaurant
		r := validRestaurant()
		if err := repo.SaveRestaurant(ctx, r); err != nil {
			t.Fatalf("SaveRestaurant for award setup failed: %v", err)
		}
		created, err := repo.FindRestaurantByURL(ctx, r.URL)
		if err != nil {
			t.Fatalf("FindRestaurantByURL failed: %v", err)
		}

		// 1) insert award (no existing)
		a := validAward(created.ID, time.Now().Year())
		if err := repo.SaveAward(ctx, a); err != nil {
			t.Fatalf("SaveAward insert failed: %v", err)
		}

		// 2) both live and identical -> no-op
		same := &models.RestaurantAward{
			RestaurantID: created.ID,
			Distinction:  a.Distinction,
			GreenStar:    a.GreenStar,
			Price:        a.Price,
			Year:         a.Year,
			WaybackURL:   "",
		}
		if err := repo.SaveAward(ctx, same); err != nil {
			t.Fatalf("SaveAward no-op branch returned error: %v", err)
		}

		// 3) incoming wayback authoritative overwrites
		wayback := &models.RestaurantAward{
			RestaurantID: created.ID,
			Distinction:  models.OneStar,
			GreenStar:    true,
			Price:        "$",
			Year:         a.Year,
			WaybackURL:   "https://web.archive.org/snapshot",
		}
		if err := repo.SaveAward(ctx, wayback); err != nil {
			t.Fatalf("SaveAward wayback overwrite failed: %v", err)
		}

		// fetch and assert updated
		var fetched models.RestaurantAward
		if err := repo.db.WithContext(ctx).Where("restaurant_id = ? AND year = ?", created.ID, a.Year).First(&fetched).Error; err != nil {
			t.Fatalf("failed to query award: %v", err)
		}
		if fetched.WaybackURL == "" || fetched.Distinction != models.OneStar {
			t.Fatalf("expected wayback overwrite; got %+v", fetched)
		}

		// 4) existing wayback + incoming live: shouldOverride when existing Distinction == SelectedRestaurants
		// seed existing with wayback and SelectedRestaurants
		seed := &models.RestaurantAward{
			RestaurantID: created.ID,
			Distinction:  models.SelectedRestaurants,
			GreenStar:    false,
			Price:        "$$",
			Year:         a.Year + 1,
			WaybackURL:   "https://web.archive.org/old",
		}
		if err := repo.db.WithContext(ctx).Create(seed).Error; err != nil {
			t.Fatalf("failed to seed award: %v", err)
		}

		incoming := &models.RestaurantAward{
			RestaurantID: created.ID,
			Distinction:  models.OneStar,
			GreenStar:    false,
			Price:        "$$$",
			Year:         seed.Year,
			WaybackURL:   "",
		}
		if err := repo.SaveAward(ctx, incoming); err != nil {
			t.Fatalf("SaveAward shouldOverride branch failed: %v", err)
		}

		var fetched2 models.RestaurantAward
		if err := repo.db.WithContext(ctx).Where("restaurant_id = ? AND year = ?", created.ID, seed.Year).First(&fetched2).Error; err != nil {
			t.Fatalf("failed to query seeded award: %v", err)
		}
		if fetched2.Distinction != models.OneStar {
			t.Fatalf("expected distinction override to OneStar, got %v", fetched2.Distinction)
		}
	})

	t.Run("FindRestaurantByURL not found returns error", func(t *testing.T) {
		repo, _ := newTestRepo(t)
		if _, err := repo.FindRestaurantByURL(ctx, "does-not-exist"); err == nil {
			t.Fatalf("expected error for missing restaurant")
		}
	})

	t.Run("ListRestaurants filters empty URLs", func(t *testing.T) {
		repo, _ := newTestRepo(t)

		// create one with empty URL via direct DB to bypass validation
		r := validRestaurant()
		r.URL = ""
		// direct create to bypass validation
		if err := repo.db.WithContext(ctx).Create(r).Error; err == nil {
			t.Fatalf("expected validation error when creating restaurant with empty URL")
		}

		// create a valid one
		v := validRestaurant()
		if err := repo.SaveRestaurant(ctx, v); err != nil {
			t.Fatalf("SaveRestaurant failed: %v", err)
		}

		list, err := repo.ListRestaurants(ctx)
		if err != nil {
			t.Fatalf("ListRestaurants failed: %v", err)
		}
		if len(list) != 1 {
			t.Fatalf("expected 1 restaurant, got %d", len(list))
		}
	})
}
