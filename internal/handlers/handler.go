package handlers

import (
	"context"
	"errors"
	"log/slog"

	"github.com/gocolly/colly/v2"
	"github.com/ngshiheng/michelin-my-maps/v4/internal/models"
	"github.com/ngshiheng/michelin-my-maps/v4/internal/parsers"
	"github.com/ngshiheng/michelin-my-maps/v4/internal/storage"
)

// Handle handles the extraction and saving of restaurant data for both scraper and backfill
func Handle(ctx context.Context, e *colly.XMLElement, repo storage.RestaurantRepository) error {
	data := parsers.Parse(e)

	// For backfill, try to find existing restaurant first
	var (
		restaurant *models.Restaurant
		err        error
	)
	if data.WaybackURL != "" {
		restaurant, err = repo.FindRestaurantByURL(ctx, data.URL)
		if err != nil {
			slog.Debug("restaurant not found, recreating from wayback data", "error", err, "wayback_url", data.WaybackURL, "url", data.URL)
		}
	}

	if data.Price == "" {
		slog.Warn("skipping award, price is empty", "wayback_url", e.Request.URL)
		return nil
	}

	// Location data from listing page is preferred for better accuracy
	// The `ParseLocationFromAddress` function is insufficient for extracting detailed location from a restaurant address
	// It splits by commas and returns only the last segment, often just the country (e.g., "Taiwan"),
	// missing useful locality info
	if e.Request.Ctx.Get("location") != "" {
		data.Location = e.Request.Ctx.Get("location")
	}

	restaurant = &models.Restaurant{
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

	if err := repo.SaveRestaurant(ctx, restaurant); err != nil {
		if errors.Is(err, context.Canceled) {
			slog.Debug("save restaurant canceled", "error", err, "id", restaurant.ID, "url", data.URL)
			return err
		}
		slog.Error("failed to save restaurant", "error", err, "id", restaurant.ID, "url", data.URL)
		return err
	}

	award := &models.RestaurantAward{
		RestaurantID: restaurant.ID,
		Year:         data.Year,
		Distinction:  data.Distinction,
		Price:        data.Price,
		GreenStar:    data.GreenStar,
		WaybackURL:   data.WaybackURL,
	}

	if err := repo.SaveAward(ctx, award); err != nil {
		if errors.Is(err, context.Canceled) {
			slog.Debug("save restaurant award canceled", "error", err, "id", restaurant.ID, "wayback_url", data.WaybackURL)
			return err
		}
		slog.Error("failed to save restaurant award", "error", err, "id", restaurant.ID, "wayback_url", data.WaybackURL)
		return err
	}

	slog.Debug("saved restaurant and award", "distinction", data.Distinction, "name", restaurant.Name, "year", data.Year, "has_wayback", data.WaybackURL != "")

	return nil
}
