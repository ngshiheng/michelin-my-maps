package handlers

import (
	"context"

	"github.com/gocolly/colly/v2"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/models"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/parsers"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/storage"
	log "github.com/sirupsen/logrus"
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
			log.WithFields(log.Fields{
				"error":       err,
				"wayback_url": data.WaybackURL,
				"url":         data.URL,
			}).Debug("restaurant not found, will create from Wayback data")
		}
	}

	// Location data from listing page is preferred for better accuracy
	// The `parseLocationFromAddress` function is insufficient for extracting detailed location from a restaurant address
	// It splits by commas and returns only the last segment, often just the country (e.g., "Taiwan"),
	// missing useful locality info
	if e.Request.Ctx.Get("location") != "" {
		data.Location = e.Request.Ctx.Get("location")
	}

	if restaurant == nil {
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
			log.WithFields(log.Fields{
				"error": err,
				"url":   data.URL,
			}).Error("failed to save restaurant")
			return err
		}
	}

	award := &models.RestaurantAward{
		RestaurantID: restaurant.ID,
		Year:         data.Year,
		Distinction:  data.Distinction,
		Price:        data.Price,
		GreenStar:    data.GreenStar,
		WaybackURL:   data.WaybackURL, // "" for live scraping, URL for backfill
	}

	if err := repo.SaveAward(ctx, award); err != nil {
		log.WithFields(log.Fields{
			"error":       err,
			"wayback_url": data.WaybackURL,
			"url":         data.URL,
		}).Error("failed to save restaurant award")
		return err
	}

	log.WithFields(log.Fields{
		"distinction": data.Distinction,
		"name":        restaurant.Name,
		"url":         data.URL,
		"year":        data.Year,
		"wayback":     data.WaybackURL != "",
	}).Debug("saved restaurant and award")

	return nil
}
