package backfill

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/models"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/parser"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/storage"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/webclient"
	log "github.com/sirupsen/logrus"
)

// Config holds configuration for the backfill process.
type Config struct {
	AllowedDomains []string
	CachePath      string
	DatabasePath   string
	Delay          time.Duration
	MaxRetry       int
	MaxURLs        int
	RandomDelay    time.Duration
	ThreadCount    int
}

// DefaultConfig returns a default config for Wayback backfill.
func DefaultConfig() *Config {
	return &Config{
		AllowedDomains: []string{"web.archive.org"},
		CachePath:      "cache/wayback",
		DatabasePath:   "data/michelin.db",
		Delay:          1 * time.Second,
		MaxRetry:       3,
		MaxURLs:        300_000,
		RandomDelay:    2 * time.Second,
		ThreadCount:    3,
	}
}

// Scraper orchestrates the Wayback backfill process.
type Scraper struct {
	config     *Config
	repository storage.RestaurantRepository
	client     *webclient.WebClient
}

// New creates a new Scraper with default config and repository.
func New() (*Scraper, error) {
	cfg := DefaultConfig()

	repo, err := storage.NewSQLiteRepository(cfg.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	wc, err := webclient.New(&webclient.Config{
		CachePath:      cfg.CachePath,
		AllowedDomains: cfg.AllowedDomains,
		Delay:          cfg.Delay,
		RandomDelay:    cfg.RandomDelay,
		ThreadCount:    cfg.ThreadCount,
		MaxURLs:        cfg.MaxURLs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create web client: %w", err)
	}

	s := &Scraper{
		client:     wc,
		config:     cfg,
		repository: repo,
	}
	return s, nil
}

// Run runs the backfill workflow for all restaurants or a specific URL.
func (b *Scraper) Run(ctx context.Context, urlFilter string) error {
	var (
		restaurants []models.Restaurant
		err         error
	)

	if urlFilter != "" {
		restaurant, err := b.repository.FindRestaurantByURL(ctx, urlFilter)
		if err != nil {
			return fmt.Errorf("failed to find restaurant by URL: %w", err)
		}
		restaurants = append(restaurants, *restaurant)
	} else {
		restaurants, err = b.repository.ListAllRestaurantsWithURL()
		if err != nil {
			return fmt.Errorf("failed to list restaurants: %w", err)
		}
	}

	log.WithFields(log.Fields{"count": len(restaurants)}).Info("restaurants to backfill")

	collector := b.client.GetCollector()
	detailCollector := b.client.CreateDetailCollector()

	mainQueue := b.client.GetQueue()

	b.setupMainHandlers(collector, detailCollector)
	b.setupDetailHandlers(ctx, detailCollector)

	for _, r := range restaurants {
		api := "https://web.archive.org/cdx/search/cdx?url=" + r.URL + "&output=json&fl=timestamp,original"
		mainQueue.AddURL(api)
	}

	log.Info("collecting Wayback snapshots for restaurants")
	mainQueue.Run(collector)
	log.Info("backfill completed")
	return nil
}

func (b *Scraper) setupMainHandlers(collector *colly.Collector, detailCollector *colly.Collector) {
	collector.OnResponse(func(r *colly.Response) {
		url := r.Request.URL.Query().Get("url")

		var rows [][]string
		if err := json.Unmarshal(r.Body, &rows); err != nil {
			log.WithFields(log.Fields{
				"url":     url,
				"cdx_api": r.Request.URL.String(),
				"error":   err,
			}).Warn("failed to parse CDX API response")
			return
		}

		if len(rows) <= 1 {
			log.WithFields(log.Fields{
				"url":     url,
				"cdx_api": r.Request.URL.String(),
			}).Debug("no snapshot rows found")
			return
		}

		const minTimestampLen = 14
		snapCount := 0
		for i, row := range rows {
			if i == 0 || len(row) == 0 {
				continue // skip header or malformed
			}
			ts := row[0]

			// The CDX API may return malformed or incomplete rows.
			// A valid timestamp must be at least 14 characters (yyyyMMddhhmmss), e.g. "20220101123456".
			// Example of a malformed row: [] or [""] or ["2022"].
			if len(ts) < minTimestampLen {
				continue
			}
			snapshotURL := fmt.Sprintf("https://web.archive.org/web/%sid_/%s", ts, url)
			err := detailCollector.Visit(snapshotURL)
			if err != nil {
				log.WithFields(log.Fields{
					"url":          url,
					"snapshot_url": snapshotURL,
					"error":        err,
				}).Debug("failed to visit snapshot URL")
				continue
			}
			snapCount++
		}

		log.WithFields(log.Fields{
			"url":       url,
			"snapshots": snapCount,
		}).Info("processed Wayback snapshot URLs")
	})

	collector.OnError(b.createErrorHandler())
}

func (b *Scraper) setupDetailHandlers(ctx context.Context, detailCollector *colly.Collector) {
	detailCollector.OnRequest(func(r *colly.Request) {
		attempt := r.Ctx.GetAny("attempt")
		if attempt == nil {
			r.Ctx.Put("attempt", 1)
			attempt = 1
		}
		log.WithFields(log.Fields{
			"attempt": attempt,
			"url":     r.URL.String(),
		}).Debug("fetching Wayback snapshot")
	})

	detailCollector.OnError(b.createErrorHandler())

	detailCollector.OnResponse(func(r *colly.Response) {
		html := r.Body
		if html == nil {
			log.WithFields(log.Fields{
				"url": r.Request.URL.String(),
			}).Error("no HTML body for snapshot")
			return
		}

		restaurantURL := extractOriginalURL(r.Request.URL.String())

		restaurant, err := b.repository.FindRestaurantByURL(ctx, restaurantURL)
		if err != nil {
			log.WithFields(log.Fields{
				"error":          err,
				"restaurant_url": restaurantURL,
				"url":            r.Request.URL.String(),
			}).Warn("no restaurant found for URL")
			return
		}

		data, err := extractRestaurantAwardData(html)

		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"url":   r.Request.URL.String(),
			}).Error("failed to parse award data from HTML")
			return
		}

		distinction := parser.ParseDistinction(data.Distinction)
		price := parser.MapPrice(data.Price)
		if price == "" {
			log.WithFields(log.Fields{
				"price": price,
				"url":   r.Request.URL.String(),
			}).Error("skipping award: price is empty")
			return
		}

		year := parser.ParseYear(data.PublishedDate)
		if year == 0 {
			log.WithFields(log.Fields{
				"publishedDate": data.PublishedDate,
				"url":           r.Request.URL.String(),
			}).Error("skipping award: invalid or missing year")
			return
		}

		award := &models.RestaurantAward{
			RestaurantID: restaurant.ID,
			Distinction:  distinction,
			Price:        price,
			GreenStar:    data.GreenStar,
			Year:         year,
		}

		err = b.repository.SaveAward(ctx, award)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"url":   r.Request.URL.String(),
			}).Error("failed to upsert award")
			return
		}

		log.WithFields(log.Fields{
			"distinction": distinction,
			"name":        restaurant.Name,
			"price":       price,
			"url":         r.Request.URL.String(),
			"year":        year,
		}).Info("upserted restaurant award")
	})
}

// createErrorHandler creates a reusable error handler for collectors with retry logic.
func (b *Scraper) createErrorHandler() func(*colly.Response, error) {
	return func(r *colly.Response, err error) {
		attempt := 1
		if v := r.Ctx.GetAny("attempt"); v != nil {
			if a, ok := v.(int); ok {
				attempt = a
			}
		}

		fields := log.Fields{
			"attempt":     attempt,
			"error":       err,
			"status_code": r.StatusCode,
			"url":         r.Request.URL.String(),
		}

		// We don't retry 403 Forbidden errors, as they indicate restricted access and retries won't help.
		// In the Wayback Machine, a 403 typically means the site owner has blocked archiving.
		switch r.StatusCode {
		case http.StatusForbidden:
			log.WithFields(fields).Debug("request forbidden, skipping retry")
			return
		case http.StatusNotFound:
			log.WithFields(fields).Debug("request not found, skipping retry")
			return
		}

		// Do not retry if already visited.
		if strings.Contains(err.Error(), "already visited") {
			log.WithFields(fields).Debug("request already visited, skipping retry")
			return
		}

		shouldRetry := attempt < b.config.MaxRetry
		if shouldRetry {
			if err := b.client.ClearCache(r.Request); err != nil {
				log.WithFields(fields).Error("failed to clear cache for request")
			}

			backoff := time.Duration(attempt) * b.config.Delay
			time.Sleep(backoff)
			log.WithFields(fields).Warnf("request failed, retrying after %v", backoff)

			r.Ctx.Put("attempt", attempt+1)
			r.Request.Retry()
		} else {
			log.WithFields(fields).Errorf("request failed after %d attempts, giving up", attempt)
		}
	}
}
