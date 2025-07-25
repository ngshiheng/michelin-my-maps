package backfill

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/client"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/extraction"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/models"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/storage"
	log "github.com/sirupsen/logrus"
)

// defaultConfig returns a default config for Wayback backfill.
func defaultConfig() *client.Config {
	return &client.Config{
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
	client     *client.Colly
	config     *client.Config
	repository storage.RestaurantRepository
}

// New creates a new Scraper with default config and repository.
func New() (*Scraper, error) {
	cfg := defaultConfig()

	repo, err := storage.NewSQLiteRepository(cfg.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	cl, err := client.New(&client.Config{
		CachePath:      cfg.CachePath,
		AllowedDomains: cfg.AllowedDomains,
		Delay:          cfg.Delay,
		RandomDelay:    cfg.RandomDelay,
		ThreadCount:    cfg.ThreadCount,
		MaxURLs:        cfg.MaxURLs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	s := &Scraper{
		client:     cl,
		config:     cfg,
		repository: repo,
	}
	return s, nil
}

// RunAll runs the backfill workflow for all restaurants.
func (s *Scraper) RunAll(ctx context.Context) error {
	restaurants, err := s.repository.ListAllRestaurantsWithURL(ctx)
	if err != nil {
		return fmt.Errorf("failed to list restaurants: %w", err)
	}

	log.WithFields(log.Fields{
		"count": len(restaurants),
	}).Info("start backfill for restaurants")

	collector := s.client.GetCollector()
	detailCollector := s.client.GetDetailCollector()

	s.setupHandlers(collector, detailCollector)
	s.setupDetailHandlers(ctx, detailCollector)

	for _, r := range restaurants {
		api := "https://web.archive.org/cdx/search/cdx?url=" + r.URL + "&output=json&fl=timestamp,original"
		s.client.EnqueueURL(api)
	}

	s.client.Run()

	// TODO: add summary of results
	log.Info("complete backfill")
	return nil
}

// Run runs the backfill workflow for a single restaurant URL.
func (s *Scraper) Run(ctx context.Context, url string) error {
	restaurant, err := s.repository.FindRestaurantByURL(ctx, url)
	if err != nil {
		return fmt.Errorf("failed to find restaurant by URL: %w", err)
	}

	log.WithFields(log.Fields{
		"url": url,
	}).Info("start backfill for restaurant")

	collector := s.client.GetCollector()
	detailCollector := s.client.GetDetailCollector()

	s.setupHandlers(collector, detailCollector)
	s.setupDetailHandlers(ctx, detailCollector)

	api := "https://web.archive.org/cdx/search/cdx?url=" + restaurant.URL + "&output=json&fl=timestamp,original"
	s.client.EnqueueURL(api)

	s.client.Run()

	log.Info("complete single restaurant backfill")
	return nil
}

func (s *Scraper) setupHandlers(collector *colly.Collector, detailCollector *colly.Collector) {
	collector.OnError(s.createErrorHandler())

	collector.OnResponse(func(r *colly.Response) {
		url := r.Request.URL.Query().Get("url")

		var rows [][]string
		if err := json.Unmarshal(r.Body, &rows); err != nil {
			log.WithFields(log.Fields{
				"url":     url,
				"cdx_api": r.Request.URL.String(),
				"error":   err,
			}).Warn("fail to parse CDX API response")
			return
		}

		if len(rows) <= 1 {
			log.WithFields(log.Fields{
				"url":     url,
				"cdx_api": r.Request.URL.String(),
			}).Debug("no snapshots found")
			return
		}

		minTimestampLen := 14
		snapshotCount := 0
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
					"error":       err,
					"wayback_url": snapshotURL,
					"url":         url,
				}).Debug("fail to visit snapshot URL")
				continue
			}
			snapshotCount++
		}

		log.WithFields(log.Fields{
			"cdx_api":     r.Request.URL.String(),
			"snapshots":   snapshotCount,
			"status_code": r.StatusCode,
		}).Info("process CDX API")
	})
}

func (s *Scraper) setupDetailHandlers(ctx context.Context, detailCollector *colly.Collector) {
	detailCollector.OnError(s.createErrorHandler())

	detailCollector.OnRequest(func(r *colly.Request) {
		attempt := r.Ctx.GetAny("attempt")
		if attempt == nil {
			r.Ctx.Put("attempt", 1)
			attempt = 1
		}
		log.WithFields(log.Fields{
			"attempt": attempt,
			"url":     r.URL.String(),
		}).Debug("fetch Wayback snapshot")
	})

	detailCollector.OnXML("html", func(e *colly.XMLElement) {
		restaurantURL := extractOriginalURL(e.Request.URL.String())

		restaurant, err := s.repository.FindRestaurantByURL(ctx, restaurantURL)
		if err != nil {
			log.WithFields(log.Fields{
				"error":       err,
				"wayback_url": e.Request.URL.String(),
				"url":         restaurantURL,
			}).Error("no restaurant found for URL")
			return
		}

		data := s.extractData(e)

		distinction := extraction.ParseDistinction(data.Distinction)
		price := extraction.MapPrice(data.Price)
		if price == "" {
			log.WithFields(log.Fields{
				"price":       price,
				"wayback_url": e.Request.URL.String(),
			}).Error("skip award: empty price")
			return
		}

		year := data.PublishedDate
		if year == 0 {
			log.WithFields(log.Fields{
				"publishedDate": data.PublishedDate,
				"wayback_url":   e.Request.URL.String(),
			}).Error("skip award: invalid or missing year")
			return
		}

		award := &models.RestaurantAward{
			RestaurantID: restaurant.ID,
			Distinction:  distinction,
			GreenStar:    data.GreenStar,
			Price:        price,
			WaybackURL:   e.Request.URL.String(),
			Year:         year,
		}

		err = s.repository.SaveAward(ctx, award)
		if err != nil {
			log.WithFields(log.Fields{
				"error":       err,
				"wayback_url": e.Request.URL.String(),
			}).Error("fail to save restaurant award")
			return
		}

		log.WithFields(log.Fields{
			"distinction": distinction,
			"name":        restaurant.Name,
			"price":       price,
			"url":         restaurant.URL,
			"wayback_url": e.Request.URL.String(),
			"year":        year,
		}).Debug("save restaurant award")
	})
}

// createErrorHandler creates a reusable error handler for collectors with retry logic.
func (s *Scraper) createErrorHandler() func(*colly.Response, error) {
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
			log.WithFields(fields).Debug("request forbidden, skip retry")
			return
		case http.StatusNotFound:
			log.WithFields(fields).Debug("request not found, skip retry")
			return
		}

		// Do not retry if already visited.
		if strings.Contains(err.Error(), "already visited") {
			log.WithFields(fields).Debug("already visited, skip retry")
			return
		}

		shouldRetry := attempt < s.config.MaxRetry
		if shouldRetry {
			if err := s.client.ClearCache(r.Request); err != nil {
				log.WithFields(fields).Error("fail to clear cache for request")
			}

			backoff := time.Duration(attempt) * s.config.Delay
			time.Sleep(backoff)
			log.WithFields(fields).Warnf("fail request, retry after %v", backoff)

			r.Ctx.Put("attempt", attempt+1)
			r.Request.Retry()
		} else {
			log.WithFields(fields).Errorf("fail request after %d attempts", attempt)
		}
	}
}
