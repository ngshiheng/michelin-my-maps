package scraper

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/client"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/handlers"
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

// RunAll crawls Michelin Guide restaurant information from the configured URLs.
func (s *Scraper) RunAll(ctx context.Context) error {
	collector := s.client.GetCollector()
	detailCollector := s.client.GetDetailCollector()

	s.setupHandlers(ctx, collector, detailCollector)
	s.setupDetailHandlers(ctx, detailCollector)

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
	log.Info("complete scraping")
	return nil
}

// Run scrapes a single restaurant URL for its details.
func (s *Scraper) Run(ctx context.Context, url string) error {
	detailCollector := s.client.GetDetailCollector()
	s.setupDetailHandlers(ctx, detailCollector)

	log.WithField("url", url).Info("scrape restaurant")
	err := detailCollector.Visit(url)
	if err != nil {
		log.WithError(err).Error("failed to visit restaurant URL")
		return err
	}
	detailCollector.Wait()
	log.Info("complete single restaurant scraping")
	return nil
}

func (s *Scraper) setupHandlers(ctx context.Context, collector *colly.Collector, detailCollector *colly.Collector) {
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
		}).Debug("fetch listing page")
	})

	collector.OnResponse(func(r *colly.Response) {
		log.WithFields(log.Fields{
			"url":         r.Request.URL.String(),
			"status_code": r.StatusCode,
		}).Info("process listing page")
	})

	collector.OnXML("//div[contains(@class, 'card__menu selection-card')]", func(e *colly.XMLElement) {
		url := e.Request.AbsoluteURL(e.ChildAttr("//a[@class='link']", "href"))
		location := e.ChildText("//div[@class='card__menu-footer--score pl-text']")
		e.Request.Ctx.Put("location", location)
		detailCollector.Request(e.Request.Method, url, nil, e.Request.Ctx, nil)
	})

	collector.OnXML("//li[@class='arrow']/a[@class='btn btn-outline-secondary btn-sm']", func(e *colly.XMLElement) {
		log.WithFields(log.Fields{
			"url": e.Attr("href"),
		}).Debug("queue next page")
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
			"attempt": attempt,
			"url":     r.URL.String(),
		}).Debug("fetch restaurant detail")
	})

	detailCollector.OnXML("html", func(e *colly.XMLElement) {
		err := handlers.Handle(ctx, e, s.repository)
		if err != nil {
			log.WithError(err).Error("failed to handle restaurant extraction")
		}
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

		if strings.Contains(err.Error(), "already visited") {
			log.WithFields(fields).Warn("already visited, skip retry")
			return
		}

		switch r.StatusCode {
		case http.StatusTooManyRequests:
			log.WithFields(fields).Debug("request rate limited, skip retry")
			return
		case http.StatusNotFound:
			log.WithFields(fields).Debug("request not found, skip retry")
			return
		}

		shouldRetry := attempt < s.config.MaxRetry
		if shouldRetry {
			if err := s.client.ClearCache(r.Request); err != nil {
				log.WithFields(fields).Error("fail to clear cache for request")
			}

			backoff := time.Duration(attempt) * s.config.Delay
			log.WithFields(fields).Debugf("fail request, retry after %v", backoff)
			time.Sleep(backoff)

			r.Ctx.Put("attempt", attempt+1)
			r.Request.Retry()
		} else {
			log.WithFields(fields).Errorf("fail request after %d attempts", attempt)
		}
	}
}
