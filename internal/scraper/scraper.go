package scraper

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/client"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/extraction"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/models"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/storage"
	log "github.com/sirupsen/logrus"
)

// defaultConfig returns a default config for the scraper.
func defaultConfig() *client.Config {
	return &client.Config{
		AllowedDomains: []string{"guide.michelin.com"},
		CachePath:      "cache/scrape",
		DatabasePath:   "data/michelin.db",
		Delay:          2 * time.Second,
		MaxRetry:       3,
		MaxURLs:        30_000,
		RandomDelay:    2 * time.Second,
		ThreadCount:    1,
	}
}

// Scraper orchestrates the scraping process.
type Scraper struct {
	client     *client.Colly
	config     *client.Config
	repository storage.RestaurantRepository
}

// New returns a new Scraper with default settings.
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

// Run crawls Michelin Guide restaurant information from the configured URLs.
func (s *Scraper) Run(ctx context.Context) error {
	collector := s.client.GetCollector()
	detailCollector := s.client.GetDetailCollector()

	s.setupHandlers(collector, detailCollector)
	s.setupDetailHandlers(ctx, detailCollector)

	// TODO: add support for command line arguments to specify URLs
	michelinGuideURLs := map[string]string{
		models.ThreeStars:          "https://guide.michelin.com/en/restaurants/3-stars-michelin",
		models.TwoStars:            "https://guide.michelin.com/en/restaurants/2-stars-michelin",
		models.OneStar:             "https://guide.michelin.com/en/restaurants/1-star-michelin",
		models.BibGourmand:         "https://guide.michelin.com/en/restaurants/bib-gourmand",
		models.SelectedRestaurants: "https://guide.michelin.com/en/restaurants/the-plate-michelin",
	}

	for _, url := range michelinGuideURLs {
		s.client.EnqueueURL(url)
	}

	s.client.Run()

	// TODO: add summary of results
	log.Info("scraping completed")
	return nil
}

func (s *Scraper) setupHandlers(collector *colly.Collector, detailCollector *colly.Collector) {
	collector.OnError(s.createErrorHandler())

	collector.OnRequest(func(r *colly.Request) {
		attempt := r.Ctx.GetAny("attempt")
		if attempt == nil {
			r.Ctx.Put("attempt", 1)
			attempt = 1
		}
		log.WithFields(log.Fields{
			"url":     r.URL.String(),
			"attempt": attempt,
		}).Debug("fetching listing page")
	})

	collector.OnResponse(func(r *colly.Response) {
		log.WithFields(log.Fields{
			"url":         r.Request.URL.String(),
			"status_code": r.StatusCode,
		}).Debug("processing listing page")
	})

	collector.OnScraped(func(r *colly.Response) {
		log.WithFields(log.Fields{"url": r.Request.URL.String()}).Debug("listing page parsed")
	})

	// Extract restaurant URLs from the listing page and visit them
	collector.OnXML(restaurantXPath, func(e *colly.XMLElement) {
		url := e.Request.AbsoluteURL(e.ChildAttr(restaurantDetailURLXPath, "href"))

		location := e.ChildText(restaurantLocationXPath)
		longitude := e.Attr("data-lng")
		latitude := e.Attr("data-lat")

		e.Request.Ctx.Put("location", location)
		e.Request.Ctx.Put("longitude", longitude)
		e.Request.Ctx.Put("latitude", latitude)

		log.WithFields(log.Fields{
			"url":       url,
			"location":  location,
			"longitude": longitude,
			"latitude":  latitude,
		}).Debug("queueing restaurant detail page")

		detailCollector.Request(e.Request.Method, url, nil, e.Request.Ctx, nil)
	})

	// Extract and visit next page links
	collector.OnXML(nextPageArrowButtonXPath, func(e *colly.XMLElement) {
		log.WithFields(log.Fields{
			"url": e.Attr("href"),
		}).Debug("queueing next page")
		e.Request.Visit(e.Attr("href"))
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
			"attempt":       attempt,
			"url":           r.URL.String(),
			"restaurant_id": r.Ctx.Get("restaurant_id"),
		}).Debug("fetching restaurant detail")
	})

	detailCollector.OnXML(restaurantAwardPublishedYearXPath, func(e *colly.XMLElement) {
		jsonLD := e.Text
		year, err := extraction.ParsePublishedYear(jsonLD)
		if err == nil && year > 0 {
			e.Request.Ctx.Put("publishedYear", year)
		}
	})

	// Extract details of each restaurant and save to database
	detailCollector.OnXML(restaurantDetailXPath, func(e *colly.XMLElement) {
		data := s.extractRestaurantData(e)

		if err := s.repository.UpsertRestaurantWithAward(ctx, data); err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"url":   data.URL,
			}).Error("failed to upsert restaurant award")
			return
		}

		log.WithFields(log.Fields{
			"distinction": data.Distinction,
			"name":        data.Name,
			"url":         data.URL,
			"year":        data.Year,
		}).Info("upserted restaurant award")
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

		// Do not retry if already visited.
		if strings.Contains(err.Error(), "already visited") {
			log.WithFields(fields).Debug("request already visited, skipping retry")
			return
		}

		shouldRetry := attempt < s.config.MaxRetry
		if shouldRetry {
			if err := s.client.ClearCache(r.Request); err != nil {
				log.WithFields(fields).Error("failed to clear cache for request")
			}

			backoff := time.Duration(attempt) * s.config.Delay
			log.WithFields(fields).Warnf("request failed, retrying after %v", backoff)
			time.Sleep(backoff)

			r.Ctx.Put("attempt", attempt+1)
			r.Request.Retry()
		} else {
			log.WithFields(fields).Errorf("request failed after %d attempts, giving up", attempt)
		}
	}
}
