package scraper

import (
	"context"
	"fmt"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/queue"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/models"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/parser"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/storage"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/webclient"
	log "github.com/sirupsen/logrus"
)

// Config holds configuration for the scraper process.
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

// DefaultConfig returns a default config for the scraper.
func DefaultConfig() *Config {
	return &Config{
		AllowedDomains: []string{"guide.michelin.com"},
		CachePath:      "cache/scrape",
		DatabasePath:   "data/michelin.db",
		Delay:          2 * time.Second,
		MaxRetry:       3,
		MaxURLs:        30_000,
		RandomDelay:    2 * time.Second,
		ThreadCount:    3,
	}
}

// Scraper orchestrates the web scraping process.
type Scraper struct {
	config       *Config
	client       *webclient.WebClient
	repository   storage.RestaurantRepository
	michelinURLs []models.GuideURL
}

// New returns a new Scraper with default settings.
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
	s.initURLs()
	return s, nil
}

// initURLs initializes the default start URLs for all award distinctions.
func (s *Scraper) initURLs() {
	allAwards := []string{
		models.ThreeStars,
		models.TwoStars,
		models.OneStar,
		models.BibGourmand,
		models.SelectedRestaurants,
	}

	for _, distinction := range allAwards {
		url, ok := models.DistinctionURL[distinction]
		if !ok {
			continue
		}

		michelinURL := models.GuideURL{
			Distinction: distinction,
			URL:         url,
		}
		s.michelinURLs = append(s.michelinURLs, michelinURL)
	}
}

// Run crawls Michelin Guide restaurant information from the configured URLs.
func (s *Scraper) Run(ctx context.Context) error {
	queue := s.client.GetQueue()
	collector := s.client.GetCollector()
	detailCollector := s.client.CreateDetailCollector()

	s.setupMainHandlers(ctx, collector, queue, detailCollector)
	s.setupDetailHandlers(ctx, detailCollector, queue)

	for _, url := range s.michelinURLs {
		s.client.AddURL(url.URL)
	}

	s.client.Run()
	log.Info("scraping completed")
	return nil
}

func (s *Scraper) setupMainHandlers(ctx context.Context, collector *colly.Collector, q *queue.Queue, detailCollector *colly.Collector) {
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
		log.WithFields(log.Fields{"url": r.Request.URL.String()}).Info("listing page parsed")
	})

	collector.OnError(s.createErrorHandler())

	// Extract restaurant URLs from the main page and visit them
	collector.OnXML(restaurantXPath, func(e *colly.XMLElement) {
		url := e.Request.AbsoluteURL(e.ChildAttr(restaurantDetailUrlXPath, "href"))

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

func (s *Scraper) setupDetailHandlers(ctx context.Context, detailCollector *colly.Collector, q *queue.Queue) {
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

	detailCollector.OnError(s.createErrorHandler())

	detailCollector.OnXML(restaurantAwardPublishedYearXPath, func(e *colly.XMLElement) {
		jsonLD := e.Text
		year, err := parser.ParsePublishedYearFromJSONLD(jsonLD)
		if err == nil && year > 0 {
			e.Request.Ctx.Put("jsonLD", jsonLD)
			e.Request.Ctx.Put("publishedYear", year)
		}
	})

	// Extract details of each restaurant and save to database
	detailCollector.OnXML(restaurantDetailXPath, func(e *colly.XMLElement) {
		data := s.extractRestaurantData(e)

		log.WithFields(log.Fields{
			"distinction":   data.Distinction,
			"name":          data.Name,
			"restaurant_id": e.Request.Ctx.Get("restaurant_id"),
			"url":           data.URL,
		}).Debug("restaurant detail extracted")

		if err := s.repository.UpsertRestaurantWithAward(ctx, data); err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"url":   data.URL,
			}).Error("failed to upsert restaurant award")
		} else {
			log.WithFields(log.Fields{
				"distinction": data.Distinction,
				"name":        data.Name,
				"url":         data.URL,
				"year":        data.Year,
			}).Info("upserted restaurant award")
		}
	})
}

// createErrorHandler creates a reusable error handler for collectors.
func (s *Scraper) createErrorHandler() func(*colly.Response, error) {
	return func(r *colly.Response, err error) {
		attempt := 1
		if v := r.Ctx.GetAny("attempt"); v != nil {
			if a, ok := v.(int); ok {
				attempt = a
			}
		}
		shouldRetry := attempt < s.config.MaxRetry

		fields := log.Fields{
			"attempt":     attempt,
			"error":       err,
			"status_code": r.StatusCode,
			"url":         r.Request.URL.String(),
		}

		if shouldRetry {
			if err := s.client.ClearCache(r.Request); err != nil {
				log.WithFields(log.Fields{
					"error": err,
					"url":   r.Request.URL.String(),
				}).Error("failed to clear cache for request")
			}
			backoff := time.Duration(attempt) * s.config.Delay
			log.WithFields(fields).Warnf("request failed on attempt %d, retrying after %v", attempt, backoff)
			time.Sleep(backoff)
			r.Ctx.Put("attempt", attempt+1)
			r.Request.Retry()
		} else {
			log.WithFields(fields).Errorf("request failed on attempt %d, giving up after max retries", attempt)
		}
	}
}
