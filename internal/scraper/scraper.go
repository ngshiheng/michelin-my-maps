package scraper

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/ngshiheng/michelin-my-maps/v4/internal/client"
	"github.com/ngshiheng/michelin-my-maps/v4/internal/handlers"
	"github.com/ngshiheng/michelin-my-maps/v4/internal/models"
	"github.com/ngshiheng/michelin-my-maps/v4/internal/storage"
	"github.com/ngshiheng/michelin-my-maps/v4/internal/utils"
	log "github.com/sirupsen/logrus"
)

// defaultConfig returns a default config for the scraper.
func defaultConfig() *client.Config {
	return &client.Config{
		AllowedDomains: []string{"guide.michelin.com"},
		CachePath:      "cache/scrape",
		DatabasePath:   "data/michelin.db",
		StoragePath:    client.DefaultStoragePath(),
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
		StoragePath:    cfg.StoragePath,
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

	// TODO: allow user to specify initial URL
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
	log.Info("completed scraping")
	return nil
}

// Run scrapes a single restaurant URL for its details.
func (s *Scraper) Run(ctx context.Context, url string) error {
	detailCollector := s.client.GetDetailCollector()
	s.setupDetailHandlers(ctx, detailCollector)

	log.WithField("url", url).Info("scraping restaurant")
	err := detailCollector.Visit(url)
	if err != nil {
		log.WithError(err).Error("failed to visit restaurant URL")
		return err
	}
	detailCollector.Wait()
	log.Info("completed scraping for one restaurant")
	return nil
}

func (s *Scraper) setupHandlers(ctx context.Context, collector *colly.Collector, detailCollector *colly.Collector) {
	collector.OnError(s.createErrorHandler())

	collector.OnRequest(func(r *colly.Request) {
		attempt := r.Ctx.GetAny("attempt_count")
		if attempt == nil {
			r.Ctx.Put("attempt_count", 1)
			attempt = 1
		}
		cookies := s.client.GetCookies(r.URL.String())
		log.WithFields(log.Fields{
			"attempt_count":   attempt,
			"cookie_count":    len(cookies),
			"request_headers": utils.FlattenHeaders(r.Headers),
			"url":             r.URL,
		}).Debug("requesting restaurant listing page")
	})

	collector.OnResponse(func(r *colly.Response) {
		if r.StatusCode == http.StatusAccepted {
			if err := s.client.ClearCache(r.Request); err != nil {
				log.WithFields(log.Fields{
					"error": err,
					"url":   r.Request.URL,
				}).Warn("failed to clear cached response")
			}
			log.WithFields(log.Fields{
				"url":         r.Request.URL,
				"status_code": r.StatusCode,
			}).Warn("request challenged")
			return
		}

		log.WithFields(log.Fields{
			"url":         r.Request.URL,
			"status_code": r.StatusCode,
		}).Info("processing restaurant listing page")
	})

	collector.OnXML("//div[contains(@class, 'card__menu selection-card')]", func(e *colly.XMLElement) {
		// In 202, this won't run; no need to handle this codepath
		url := e.Request.AbsoluteURL(e.ChildAttr("//a[@class='link']", "href"))
		location := e.ChildText("//div[@class='card__menu-footer--score pl-text']")
		e.Request.Ctx.Put("location", location)
		detailCollector.Request(e.Request.Method, url, nil, e.Request.Ctx, nil)
	})

	collector.OnXML("//li[@class='arrow']/a[@class='btn btn-outline-secondary btn-sm']", func(e *colly.XMLElement) {
		// In 202, this won't run; no need to handle this codepath
		log.WithFields(log.Fields{
			"url": e.Attr("href"),
		}).Debug("queuing next page")
		e.Request.Visit(e.Attr("href"))
	})
}

func (s *Scraper) setupDetailHandlers(ctx context.Context, detailCollector *colly.Collector) {
	detailCollector.OnError(s.createErrorHandler())

	detailCollector.OnRequest(func(r *colly.Request) {
		attempt := r.Ctx.GetAny("attempt_count")
		if attempt == nil {
			r.Ctx.Put("attempt_count", 1)
			attempt = 1
		}
		cookies := s.client.GetCookies(r.URL.String())
		log.WithFields(log.Fields{
			"attempt_count":   attempt,
			"cookie_count":    len(cookies),
			"request_headers": utils.FlattenHeaders(r.Headers),
			"url":             r.URL,
		}).Debug("requesting restaurant detail")
	})

	detailCollector.OnXML("html", func(e *colly.XMLElement) {
		if e.Response.StatusCode == http.StatusAccepted {
			if err := s.client.ClearCache(e.Request); err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Warn("failed to clear cached response")
			}

			log.WithFields(log.Fields{
				"url":         e.Request.URL,
				"status_code": e.Response.StatusCode,
			}).Warn("request challenged")
			return
		}

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
		if v := r.Ctx.GetAny("attempt_count"); v != nil {
			if a, ok := v.(int); ok {
				attempt = a
			}
		}

		cookies := s.client.GetCookies(r.Request.URL.String())
		fields := log.Fields{
			"attempt_count":    attempt,
			"error":            err,
			"status_code":      r.StatusCode,
			"url":              r.Request.URL,
			"request_headers":  utils.FlattenHeaders(r.Request.Headers),
			"cookie_count":     len(cookies),
			"response_headers": utils.FlattenHeaders(r.Headers),
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
				log.WithFields(fields).Error("failed to clear cache for request")
			}

			backoff := time.Duration(attempt) * s.config.Delay
			log.WithFields(fields).Debugf("failed request, retry after %v", backoff)
			time.Sleep(backoff)

			r.Ctx.Put("attempt_count", attempt+1)
			r.Request.Retry()
		} else {
			log.WithFields(fields).Errorf("failed request after %d attempts", attempt)
		}
	}
}
