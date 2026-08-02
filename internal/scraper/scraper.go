package scraper

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	"github.com/velebak/colly-sqlite3-storage/colly/sqlite3"
)

const (
	xPathRestaurantCard         = "//div[contains(@class, 'card__menu selection-card')]"
	xPathRestaurantCardLink     = "//a[@class='link']"
	xPathRestaurantCardLocation = "//div[@class='card__menu-footer--score pl-text']"
	xPathPaginationArrow        = "//li[@class='arrow']/a[@class='btn btn-outline-secondary btn-sm']"
	xPathDetailRoot             = "html"
)

func defaultConfig() *client.Config {
	return &client.Config{
		AllowedDomains: []string{"guide.michelin.com"},
		CachePath:      client.DefaultCacheScrape,
		DatabasePath:   client.DefaultDataPath,
		StoragePath:    client.DefaultStoragePath,
		Delay:          2 * time.Second,
		MaxRetry:       3,
		RandomDelay:    3 * time.Second, // 2–5 s jitter
		ThreadCount:    10,
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
func New(ignoreCache bool) (*Scraper, error) {
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
	if ignoreCache {
		slog.Debug("running with no cache")
		clientCfg.CachePath = ""
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
		slog.Info("non-empty queue detected, resuming detail scrape", "queue_size", queueSize)
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
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := collector.Visit(url); err != nil {
				slog.Error("failed to visit seed url", "url", url, "error", err)
			}
		}
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	// Phase 2: drain all ~18k detail page URLs accumulated in colly.db queue
	slog.Info("starting detail scrape, draining queue")
	if err := s.client.RunQueue(detailCollector); err != nil {
		return err
	}

	slog.Info("completed scraping", "scraped", s.scraped.Load())
	return nil
}

// Run scrapes a single restaurant URL for its details.
func (s *Scraper) Run(ctx context.Context, url string) error {
	slog.Debug("running scrape for restaurant", "url", url)

	detailCollector := s.client.GetDetailCollector()
	s.setupDetailHandlers(ctx, detailCollector)

	err := detailCollector.Visit(url)
	if err != nil {
		slog.Error("failed to visit restaurant URL", "error", err, "url", url)
		return err
	}

	slog.Debug("completed scraping for one restaurant", "url", url)
	return nil
}

func (s *Scraper) setupHandlers(ctx context.Context, collector *colly.Collector) {
	collector.OnError(s.createErrorHandler(ctx))

	collector.OnRequest(func(r *colly.Request) {
		if ctx.Err() != nil {
			r.Abort()
			return
		}

		r.Headers.Set("Accept-Language", "en-SG,en;q=0.9")

		attempt := r.Ctx.GetAny("attempt")
		if attempt == nil {
			r.Ctx.Put("attempt", 1)
			attempt = 1
		}
		_, cacheHit := s.client.IsCached(r.URL.String())
		r.Ctx.Put("cache_hit", cacheHit)

		slog.Info("requesting restaurant listing page", "attempt", attempt, "cache_hit", cacheHit, "url", r.URL)
	})

	collector.OnResponse(func(r *colly.Response) {
		if r.StatusCode == http.StatusAccepted {
			s.retryAccepted(r, "restaurant listing page")
			return
		}

		slog.Debug("fetched listing page, enqueuing restaurant details", "cache_hit", r.Ctx.GetAny("cache_hit"), "url", r.Request.URL, "status_code", r.StatusCode)
	})

	collector.OnXML(xPathRestaurantCard, func(e *colly.XMLElement) {
		if ctx.Err() != nil {
			return
		}

		// In 202, this won't run; no need to handle this codepath.
		url := e.Request.AbsoluteURL(e.ChildAttr(xPathRestaurantCardLink, "href"))
		location := e.ChildText(xPathRestaurantCardLocation)

		// Enqueue the detail URL into colly.db so phase 2 (RunQueue) can
		// process it with detailCollector. EnqueueURLWithContext is required
		// (instead of queue.AddURL) to carry the location through the queue.
		if err := s.client.EnqueueURLWithContext(url, location); err != nil {
			slog.Warn("failed to enqueue detail url", "error", err, "url", url)
		}
	})

	collector.OnXML(xPathPaginationArrow, func(e *colly.XMLElement) {
		if ctx.Err() != nil {
			return
		}

		// In 202, this won't run; no need to handle this codepath.
		// xPathPaginationArrow matches both prev and next arrows. Prev-page links
		// are skipped naturally: those pages are already in the visited table, so
		// Visit returns AlreadyVisitedError and the error handler drops it silently.
		nextURL := e.Request.AbsoluteURL(e.Attr("href"))
		slog.Debug("visiting next page", "url", nextURL)
		e.Request.Visit(nextURL)
	})
}

func (s *Scraper) retryAccepted(r *colly.Response, requestType string) {
	fields := map[string]any{
		"request_type": requestType,
		"status_code":  r.StatusCode,
		"url":          r.Request.URL,
	}

	if err := s.client.ClearCache(r.Request); err != nil {
		slog.Warn("failed to clear cache", append(utils.FieldsToArgs(fields), "error", err)...)
	}

	slog.Error("session expired", utils.FieldsToArgs(fields)...)
	os.Exit(2)
}

func (s *Scraper) setupDetailHandlers(ctx context.Context, detailCollector *colly.Collector) {
	detailCollector.OnError(s.createErrorHandler(ctx))

	detailCollector.OnRequest(func(r *colly.Request) {
		if ctx.Err() != nil {
			r.Abort()
			return
		}

		r.Headers.Set("Accept-Language", "en-SG,en;q=0.9")

		attempt := r.Ctx.GetAny("attempt")
		if attempt == nil {
			r.Ctx.Put("attempt", 1)
			attempt = 1
		}
		_, cacheHit := s.client.IsCached(r.URL.String())

		r.Ctx.Put("cache_hit", cacheHit)

		slog.Info("requesting restaurant details", "attempt", attempt, "cache_hit", cacheHit, "url", r.URL)
	})

	detailCollector.OnResponse(func(r *colly.Response) {
		if r.StatusCode == http.StatusAccepted {
			s.retryAccepted(r, "restaurant detail")
		}
	})

	detailCollector.OnXML(xPathDetailRoot, func(e *colly.XMLElement) {
		if ctx.Err() != nil {
			return
		}

		if e.Response.StatusCode == http.StatusAccepted {
			// 202 retries are handled in OnResponse; ignore this response body.
			return
		}

		err := handlers.Handle(ctx, e, s.repository)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				slog.Debug("restaurant extraction canceled", "error", err, "url", e.Request.URL)
				return
			}
			slog.Error("failed to handle restaurant extraction", "error", err, "url", e.Request.URL)
			return
		}
		s.scraped.Add(1)
	})
}

func (s *Scraper) createErrorHandler(ctx context.Context) func(*colly.Response, error) {
	return func(r *colly.Response, err error) {
		attempt := 1
		if v := r.Ctx.GetAny("attempt"); v != nil {
			if a, ok := v.(int); ok {
				attempt = a
			}
		}

		cookies := s.client.GetCookies(r.Request.URL.String())
		fields := map[string]any{
			"attempt":      attempt,
			"cookie_count": len(cookies),
			"status_code":  r.StatusCode,
			"url":          r.Request.URL,
		}

		if strings.Contains(err.Error(), "already visited") {
			slog.Debug("already visited, skip retry", append(utils.FieldsToArgs(fields), "error", err)...)
			return
		}

		// status 0 means no HTTP response was received (transport-level failure).
		// context.Canceled means the program is shutting down — retrying is pointless
		// and delays shutdown by burning through all MaxRetry attempts.
		if errors.Is(err, context.Canceled) {
			slog.Debug("context canceled, skip retry", append(utils.FieldsToArgs(fields), "error", err)...)
			return
		}

		switch r.StatusCode {
		case http.StatusTooManyRequests:
			slog.Warn("request rate limited, skip retry", append(utils.FieldsToArgs(fields), "error", err)...)
			return
		case http.StatusNotFound:
			slog.Debug("request not found, skip retry", append(utils.FieldsToArgs(fields), "error", err)...)
			return
		}

		shouldRetry := attempt < s.config.MaxRetry
		if shouldRetry {
			if ctx.Err() != nil {
				slog.Debug("context canceled, skip retry", utils.FieldsToArgs(fields)...)
				return
			}

			if err := s.client.ClearCache(r.Request); err != nil {
				slog.Error("failed to clear cache", append(utils.FieldsToArgs(fields), "error", err, "request_headers", utils.FlattenHeaders(r.Request.Headers))...)
			}

			backoff := time.Duration(attempt) * s.config.Delay
			slog.Debug("failed request, retrying", append(utils.FieldsToArgs(fields), "backoff", backoff)...)
			select {
			case <-ctx.Done():
				slog.Debug("context canceled during backoff, skip retry", utils.FieldsToArgs(fields)...)
				return
			case <-time.After(backoff):
			}

			if ctx.Err() != nil {
				slog.Debug("context canceled before retry, skip retry", utils.FieldsToArgs(fields)...)
				return
			}

			r.Ctx.Put("attempt", attempt+1)
			r.Request.Retry()
		} else {
			slog.Error("failed request, max retries reached", append(utils.FieldsToArgs(fields), "error", err, "request_headers", utils.FlattenHeaders(r.Request.Headers))...)
		}
	}
}
