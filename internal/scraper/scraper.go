package scraper

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/ngshiheng/michelin-my-maps/v4/internal/client"
	"github.com/ngshiheng/michelin-my-maps/v4/internal/handlers"
	"github.com/ngshiheng/michelin-my-maps/v4/internal/models"
	"github.com/ngshiheng/michelin-my-maps/v4/internal/storage"
	"github.com/ngshiheng/michelin-my-maps/v4/internal/utils"
	log "github.com/sirupsen/logrus"
	"github.com/velebak/colly-sqlite3-storage/colly/sqlite3"
)

const (
	xPathRestaurantCard         = "//div[contains(@class, 'card__menu selection-card')]"
	xPathRestaurantCardLink     = "//a[@class='link']"
	xPathRestaurantCardLocation = "//div[@class='card__menu-footer--score pl-text']"
	xPathPaginationArrow        = "//li[@class='arrow']/a[@class='btn btn-outline-secondary btn-sm']"
	xPathDetailRoot             = "html"
)

// defaultConfig returns a default config for the scraper
func defaultConfig() *client.Config {
	return &client.Config{
		AllowedDomains: []string{"guide.michelin.com"},
		CachePath:      client.DefaultCacheScrape,
		DatabasePath:   client.DefaultDataPath,
		StoragePath:    client.DefaultStoragePath,
		Delay:          2 * time.Second,
		MaxRetry:       3,
		RandomDelay:    3 * time.Second, // 2–5 s jitter
		// ThreadCount: 1 is intentional – guide.michelin.com uses AWS WAF;
		// parallelising seed requests would make all N listing pages land
		// simultaneously
		ThreadCount: 1,
	}
}

// Scraper orchestrates the scraping process
type Scraper struct {
	client     *client.Colly
	config     *client.Config
	repository storage.RestaurantRepository
	scraped    atomic.Int64
}

// New returns a new Scraper with default settings
func New() (*Scraper, error) {
	cfg := defaultConfig()

	repo, err := storage.NewSQLiteRepository(cfg.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	clientCfg := &client.Config{
		AllowedDomains: cfg.AllowedDomains,
		CachePath:      cfg.CachePath,
		Delay:          cfg.Delay,
		MaxRetry:       cfg.MaxRetry,
		RandomDelay:    cfg.RandomDelay,
		StoragePath:    cfg.StoragePath,
		ThreadCount:    cfg.ThreadCount,
	}

	cl, err := client.New(clientCfg)
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

// InitCookies persists Michelin Guide session cookies to the cookie storage.
// Existing rows are cleared first since the sqlite3 backend uses plain INSERT (not upsert)
// We need to Init -> Clear -> Init because Clear does not do DROP TABLE IF EXISTS
func (s *Scraper) InitCookies(cookies []*http.Cookie) error {
	url := &url.URL{Host: "guide.michelin.com"}

	store := &sqlite3.Storage{Filename: s.config.StoragePath}
	defer store.Close()

	if err := store.Init(); err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	if err := store.Clear(); err != nil {
		return fmt.Errorf("failed to clear storage: %w", err)
	}

	if err := store.Init(); err != nil {
		return fmt.Errorf("failed to re-initialize storage: %w", err)
	}

	lines := make([]string, len(cookies))

	for i, c := range cookies {
		lines[i] = c.String()
	}

	store.SetCookies(url, strings.Join(lines, "\n"))
	return nil
}

// RunAll crawls Michelin Guide restaurant information from the configured URLs.
func (s *Scraper) RunAll(ctx context.Context) error {
	collector := s.client.GetCollector()
	detailCollector := s.client.GetDetailCollector()

	s.setupHandlers(ctx, collector)
	s.setupDetailHandlers(ctx, detailCollector)

	queueSize, err := s.client.QueueSize()
	if err != nil {
		return fmt.Errorf("failed to check queue size: %w", err)
	}

	if queueSize > 0 {
		// Resume an interrupted run: the queue still has unprocessed detail URLs
		// from the previous Phase 1. Skip Phase 1 to avoid appending duplicate
		// rows (AddRequest has no dedup — it's a plain INSERT).
		log.WithField("queue_size", queueSize).Info("non-empty queue detected, resuming detail scrape")
	} else {
		// Fresh start (first run, or resuming after a fully completed prior run).
		// Clear visited so seed listing pages can be re-visited; on a truly first
		// run the table is already empty so this is a no-op.
		if err := s.client.ClearVisited(); err != nil {
			return fmt.Errorf("failed to clear visited table: %w", err)
		}

		// Phase 1: visit all 5 seed listing pages. Each page visit follows pagination
		// via e.Request.Visit (synchronous, collector's WaitGroup tracks it) and
		// enqueues discovered detail page URLs into colly.db via EnqueueURLWithContext.
		// TODO: allow user to specify initial URL
		michelinGuideURLs := map[string]string{
			models.ThreeStars:          "https://guide.michelin.com/en/restaurants/3-stars-michelin",
			models.TwoStars:            "https://guide.michelin.com/en/restaurants/2-stars-michelin",
			models.OneStar:             "https://guide.michelin.com/en/restaurants/1-star-michelin",
			models.BibGourmand:         "https://guide.michelin.com/en/restaurants/bib-gourmand",
			models.SelectedRestaurants: "https://guide.michelin.com/en/restaurants/the-plate-michelin",
		}

		for _, url := range michelinGuideURLs {
			if err := collector.Visit(url); err != nil {
				log.WithField("url", url).WithError(err).Error("failed to visit seed url")
			}
		}
	}

	// Phase 2: drain all ~18k detail page URLs accumulated in colly.db queue
	log.Info("starting detail scrape, draining queue")
	if err := s.client.RunQueue(detailCollector); err != nil {
		return err
	}

	log.WithField("scraped", s.scraped.Load()).Info("completed scraping")
	return nil
}

// Run scrapes a single restaurant URL for its details.
func (s *Scraper) Run(ctx context.Context, url string) error {
	log.WithField("url", url).Debug("running scrape for restaurant")

	detailCollector := s.client.GetDetailCollector()
	s.setupDetailHandlers(ctx, detailCollector)

	err := detailCollector.Visit(url)
	if err != nil {
		log.WithError(err).WithField("url", url).Error("failed to visit restaurant URL")
		return err
	}

	log.WithField("url", url).Debug("completed scraping for one restaurant")
	return nil
}

func (s *Scraper) setupHandlers(ctx context.Context, collector *colly.Collector) {
	collector.OnError(s.createErrorHandler())

	collector.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept-Language", "en-SG,en;q=0.9")

		attempt := r.Ctx.GetAny("attempt")
		if attempt == nil {
			r.Ctx.Put("attempt", 1)
			attempt = 1
		}
		_, cacheHit := s.client.IsCached(r.URL.String())
		r.Ctx.Put("cache_hit", cacheHit)

		log.WithFields(log.Fields{
			"attempt":   attempt,
			"cache_hit": cacheHit,
			"url":       r.URL,
		}).Info("requesting restaurant listing page")
	})

	collector.OnResponse(func(r *colly.Response) {
		if r.StatusCode == http.StatusAccepted {
			s.retryAccepted(r, "restaurant listing page")
			return
		}

		log.WithFields(log.Fields{
			"cache_hit":   r.Ctx.GetAny("cache_hit"),
			"url":         r.Request.URL,
			"status_code": r.StatusCode,
		}).Debug("fetched listing page, enqueuing restaurant details")
	})

	collector.OnXML(xPathRestaurantCard, func(e *colly.XMLElement) {
		// In 202, this won't run; no need to handle this codepath.
		url := e.Request.AbsoluteURL(e.ChildAttr(xPathRestaurantCardLink, "href"))
		location := e.ChildText(xPathRestaurantCardLocation)

		// Enqueue the detail URL into colly.db so phase 2 (RunQueue) can
		// process it with detailCollector. EnqueueURLWithContext is required
		// (instead of queue.AddURL) to carry the location through the queue.
		if err := s.client.EnqueueURLWithContext(url, location); err != nil {
			log.WithError(err).WithField("url", url).Warn("failed to enqueue detail url")
		}
	})

	collector.OnXML(xPathPaginationArrow, func(e *colly.XMLElement) {
		// In 202, this won't run; no need to handle this codepath.
		// xPathPaginationArrow matches both prev and next arrows. Prev-page links
		// are skipped naturally: those pages are already in the visited table, so
		// Visit returns AlreadyVisitedError and the error handler drops it silently.
		nextURL := e.Request.AbsoluteURL(e.Attr("href"))
		log.WithField("url", nextURL).Debug("visiting next page")
		e.Request.Visit(nextURL)
	})
}

// retryAccepted handles a 202 response from Michelin Guide, which (almost)
// always indicates session expiry
func (s *Scraper) retryAccepted(r *colly.Response, requestType string) {
	fields := log.Fields{
		"request_type": requestType,
		"status_code":  r.StatusCode,
		"url":          r.Request.URL,
	}

	if err := s.client.ClearCache(r.Request); err != nil {
		log.WithFields(fields).WithError(err).Warn("failed to clear cache")
	}

	log.WithFields(fields).Error("session expired")
	os.Exit(2)
}

func (s *Scraper) setupDetailHandlers(ctx context.Context, detailCollector *colly.Collector) {
	detailCollector.OnError(s.createErrorHandler())

	detailCollector.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept-Language", "en-SG,en;q=0.9")

		attempt := r.Ctx.GetAny("attempt")
		if attempt == nil {
			r.Ctx.Put("attempt", 1)
			attempt = 1
		}
		_, cacheHit := s.client.IsCached(r.URL.String())

		r.Ctx.Put("cache_hit", cacheHit)

		log.WithFields(log.Fields{
			"attempt":   attempt,
			"cache_hit": cacheHit,
			"url":       r.URL,
		}).Info("requesting restaurant details")
	})

	detailCollector.OnResponse(func(r *colly.Response) {
		if r.StatusCode == http.StatusAccepted {
			s.retryAccepted(r, "restaurant detail")
		}
	})

	detailCollector.OnXML(xPathDetailRoot, func(e *colly.XMLElement) {
		if e.Response.StatusCode == http.StatusAccepted {
			// 202 retries are handled in OnResponse; ignore this response body.
			return
		}

		err := handlers.Handle(ctx, e, s.repository)
		if err != nil {
			log.WithError(err).WithField("url", e.Request.URL).Error("failed to handle restaurant extraction")
			return
		}
		s.scraped.Add(1)
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

		cookies := s.client.GetCookies(r.Request.URL.String())
		fields := log.Fields{
			"attempt":      attempt,
			"cookie_count": len(cookies),
			"status_code":  r.StatusCode,
			"url":          r.Request.URL,
		}

		if strings.Contains(err.Error(), "already visited") {
			log.WithError(err).WithFields(fields).Debug("already visited, skip retry")
			return
		}

		// status 0 means no HTTP response was received (transport-level failure).
		// context.Canceled means the program is shutting down — retrying is pointless
		// and delays shutdown by burning through all MaxRetry attempts.
		if errors.Is(err, context.Canceled) {
			log.WithError(err).WithFields(fields).Debug("context canceled, skip retry")
			return
		}

		switch r.StatusCode {
		case http.StatusTooManyRequests:
			log.WithError(err).WithFields(fields).Warn("request rate limited, skip retry")
			return
		case http.StatusNotFound:
			log.WithError(err).WithFields(fields).Debug("request not found, skip retry")
			return
		}

		shouldRetry := attempt < s.config.MaxRetry
		if shouldRetry {
			if err := s.client.ClearCache(r.Request); err != nil {
				log.WithError(err).WithFields(fields).WithField("request_headers", utils.FlattenHeaders(r.Request.Headers)).Error("failed to clear cache")
			}

			backoff := time.Duration(attempt) * s.config.Delay
			log.WithFields(fields).WithField("backoff", backoff).Debug("failed request, retrying")
			time.Sleep(backoff)

			r.Ctx.Put("attempt", attempt+1)
			r.Request.Retry()
		} else {
			log.WithError(err).WithFields(fields).WithField("request_headers", utils.FlattenHeaders(r.Request.Headers)).Error("failed request, max retries reached")
		}
	}
}
