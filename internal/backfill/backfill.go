package backfill

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	sqliteRepo, ok := b.repository.(*storage.SQLiteRepository)
	if !ok {
		return fmt.Errorf("repository does not support listing restaurants")
	}

	var (
		restaurants []models.Restaurant
		err         error
	)
	if urlFilter != "" {
		all, err := sqliteRepo.ListAllRestaurantsWithURL()
		if err != nil {
			return fmt.Errorf("failed to list restaurants: %w", err)
		}
		found := false
		for _, r := range all {
			if r.URL == urlFilter {
				restaurants = []models.Restaurant{r}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("no restaurant found with URL: %s", urlFilter)
		}
	} else {
		restaurants, err = sqliteRepo.ListAllRestaurantsWithURL()
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
	if err := mainQueue.Run(collector); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("failed to run main collector queue")
		return fmt.Errorf("failed to run main collector queue: %w", err)
	}

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
			}).Warn("no snapshot rows found")
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
				}).Warn("failed to visit snapshot URL")
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
			}).Warn("no HTML body for snapshot")
			return
		}

		restaurantURL := extractOriginalURL(r.Request.URL.String())
		sqliteRepo, ok := b.repository.(*storage.SQLiteRepository)
		if !ok {
			log.WithFields(log.Fields{
				"url": r.Request.URL.String(),
			}).Error("repository does not support FindRestaurantByURL")
			return
		}

		restaurant, err := sqliteRepo.FindRestaurantByURL(ctx, restaurantURL)
		if err != nil || restaurant == nil {
			log.WithFields(log.Fields{
				"url":            r.Request.URL.String(),
				"restaurant_url": restaurantURL,
				"error":          err,
			}).Warn("no restaurant found for snapshot response")
			return
		}

		distinction, price, greenstar, publishedDate, err := extractAwardDataFromHTML(html)
		if err != nil {
			log.WithFields(log.Fields{
				"url": r.Request.URL.String(),

				"error": err,
			}).Warn("failed to parse award data from HTML")
			return
		}

		distinction = parser.ParseDistinction(distinction)
		price = parser.MapPrice(price)

		year := parser.ParseYear(publishedDate)
		if year == 0 {
			log.WithFields(log.Fields{
				"publishedDate": publishedDate,
				"url":           r.Request.URL.String(),
			}).Warn("skipping award: invalid or missing year")
			return
		}

		award := &models.RestaurantAward{
			RestaurantID: restaurant.ID,
			Distinction:  distinction,
			Price:        price,
			GreenStar:    greenstar,
			Year:         year,
		}
		err = b.repository.SaveAward(ctx, award)
		if err != nil {
			log.WithFields(log.Fields{
				"distinction":   distinction,
				"error":         err,
				"greenstar":     greenstar,
				"price":         price,
				"publishedDate": publishedDate,
				"url":           r.Request.URL.String(),
				"year":          year,
			}).Error("failed to upsert award")
			return
		}

		log.WithFields(log.Fields{
			"distinction": distinction,
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
		shouldRetry := r.StatusCode != http.StatusForbidden && attempt < b.config.MaxRetry

		fields := log.Fields{
			"attempt":     attempt,
			"error":       err,
			"status_code": r.StatusCode,
			"url":         r.Request.URL.String(),
		}

		if shouldRetry {
			if err := b.client.ClearCache(r.Request); err != nil {
				log.WithFields(log.Fields{
					"url":   r.Request.URL.String(),
					"error": err,
				}).Error("failed to clear cache for request")
			}
			backoff := time.Duration(attempt) * b.config.Delay
			log.WithFields(fields).Warnf("request failed on attempt %d, retrying after %v", attempt, backoff)
			time.Sleep(backoff)
			r.Ctx.Put("attempt", attempt+1)
			r.Request.Retry()
		} else {
			log.WithFields(fields).Errorf("request failed on attempt %d, giving up after max retries", attempt)
		}
	}
}
