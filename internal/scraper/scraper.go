package scraper

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
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

const (
	xPathRestaurantCard         = "//div[contains(@class, 'card__menu selection-card')]"
	xPathRestaurantCardLink     = "//a[@class='link']"
	xPathRestaurantCardLocation = "//div[@class='card__menu-footer--score pl-text']"
	xPathPaginationArrow        = "//li[@class='arrow']/a[@class='btn btn-outline-secondary btn-sm']"
	xPathDetailRoot             = "html"
)

// defaultConfig returns a default config for the scraper.
func defaultConfig() *client.Config {
	return &client.Config{
		AllowedDomains: []string{"guide.michelin.com"},
		CachePath:      client.DefaultCacheScrape,
		DatabasePath:   client.DefaultDataPath,
		StoragePath:    client.DefaultStoragePath,
		Delay:          2 * time.Second,
		MaxRetry:       3,
		// Queue only holds the 5 listing seed URLs; keep small to avoid
		// misleading over-allocation.
		MaxURLs:     100,
		RandomDelay: 3 * time.Second, // 2–5 s jitter; wider spread reduces WAF fingerprinting
		// ThreadCount: 1 is intentional – guide.michelin.com uses AWS WAF;
		// parallelising seed requests would make all 5 listing pages land
		// simultaneously
		ThreadCount: 1,
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

	clientCfg := &client.Config{
		CachePath:      cfg.CachePath,
		AllowedDomains: cfg.AllowedDomains,
		StoragePath:    cfg.StoragePath,
		Delay:          cfg.Delay,
		RandomDelay:    cfg.RandomDelay,
		ThreadCount:    cfg.ThreadCount,
		MaxURLs:        cfg.MaxURLs,
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
		if err := s.client.EnqueueURL(url); err != nil {
			return err
		}
	}
	if err := s.client.Run(); err != nil {
		return err
	}

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
		cacheEnabled, cacheHit := s.client.IsCached(r.URL.String())
		r.Ctx.Put("cache_enabled", cacheEnabled)
		r.Ctx.Put("cache_hit", cacheHit)

		cookies := s.client.GetCookies(r.URL.String())

		log.WithFields(log.Fields{
			"attempt_count":   attempt,
			"cache_enabled":   cacheEnabled,
			"cache_hit":       cacheHit,
			"cookie_count":    len(cookies),
			"request_headers": utils.FlattenHeaders(r.Headers),
			"url":             r.URL,
		}).Debug("requesting restaurant listing page")
	})

	collector.OnResponse(func(r *colly.Response) {
		if r.StatusCode == http.StatusAccepted {
			s.retryAccepted(r, "restaurant listing page")
			return
		}

		log.WithFields(log.Fields{
			"cache_enabled": r.Ctx.GetAny("cache_enabled"),
			"cache_hit":     r.Ctx.GetAny("cache_hit"),
			"url":           r.Request.URL,
			"status_code":   r.StatusCode,
		}).Info("processing restaurant listing page")
	})

	collector.OnXML(xPathRestaurantCard, func(e *colly.XMLElement) {
		// In 202, this won't run; no need to handle this codepath.
		// Use a fresh context per detail request so that attempt_count
		// from one restaurant's retries does not leak into the next.
		url := e.Request.AbsoluteURL(e.ChildAttr(xPathRestaurantCardLink, "href"))
		ctx := colly.NewContext()
		ctx.Put("location", e.ChildText(xPathRestaurantCardLocation))
		detailCollector.Request(e.Request.Method, url, nil, ctx, nil)
	})

	collector.OnXML(xPathPaginationArrow, func(e *colly.XMLElement) {
		// In 202, this won't run; no need to handle this codepath.
		// xPathPaginationArrow matches both prev and next arrows. AllowURLRevisit
		// is enabled on the collector (required for 202 challenge retries), so
		// Colly won't deduplicate URLs. Without the guard below, following the
		// prev-arrow from page N back to page N-1 would cause an infinite loop.
		// We prevent this by only visiting a link whose page number is strictly
		// greater than the current page (i.e. forward-only navigation)
		nextURL := e.Request.AbsoluteURL(e.Attr("href"))
		if getListingPageNumber(nextURL) <= getListingPageNumber(e.Request.URL.String()) {
			return
		}
		log.WithFields(log.Fields{
			"url": nextURL,
		}).Debug("queuing next page")
		e.Request.Visit(nextURL)
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
		cacheEnabled, cacheHit := s.client.IsCached(r.URL.String())
		r.Ctx.Put("cache_enabled", cacheEnabled)
		r.Ctx.Put("cache_hit", cacheHit)
		cookies := s.client.GetCookies(r.URL.String())

		log.WithFields(log.Fields{
			"attempt_count":   attempt,
			"cache_enabled":   cacheEnabled,
			"cache_hit":       cacheHit,
			"cookie_count":    len(cookies),
			"request_headers": utils.FlattenHeaders(r.Headers),
			"url":             r.URL,
		}).Debug("requesting restaurant detail")
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
			log.WithError(err).Error("failed to handle restaurant extraction")
		}
	})
}

func (s *Scraper) retryAccepted(r *colly.Response, requestType string) {
	attempt := 1
	if v := r.Ctx.GetAny("attempt_count"); v != nil {
		if a, ok := v.(int); ok {
			attempt = a
		}
	}

	fields := log.Fields{
		"attempt_count": attempt,
		"max_retry":     s.config.MaxRetry,
		"request_type":  requestType,
		"status_code":   r.StatusCode,
		"url":           r.Request.URL,
	}

	if err := s.client.ClearCache(r.Request); err != nil {
		log.WithFields(fields).WithError(err).Warn("failed to clear cache")
	}

	if attempt >= s.config.MaxRetry {
		log.WithFields(fields).Error("request challenged and max retries reached")
		return
	}

	backoff := time.Duration(attempt) * s.config.Delay
	log.WithFields(fields).Warnf("request challenged, retry after %v", backoff)
	time.Sleep(backoff)

	r.Ctx.Put("attempt_count", attempt+1)
	if err := r.Request.Retry(); err != nil {
		log.WithFields(fields).WithError(err).Error("failed to retry challenged request")
	}
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
			"attempt_count":   attempt,
			"error":           err,
			"status_code":     r.StatusCode,
			"url":             r.Request.URL,
			"request_headers": utils.FlattenHeaders(r.Request.Headers),
			"cookie_count":    len(cookies),
		}

		if strings.Contains(err.Error(), "already visited") {
			log.WithFields(fields).Warn("already visited, skip retry")
			return
		}

		// status 0 means no HTTP response was received (transport-level failure).
		// context.Canceled means the program is shutting down — retrying is pointless
		// and delays shutdown by burning through all MaxRetry attempts.
		if errors.Is(err, context.Canceled) {
			log.WithFields(fields).Debug("context canceled, skip retry")
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
				log.WithFields(fields).Error("failed to clear cache")
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

// getListingPageNumber extracts the page number from a Michelin listing URL.
// URLs without an explicit page component (e.g. "/en/restaurants/2-stars-michelin")
// are treated as page 1. This is used by the pagination handler to distinguish
// forward (next) from backward (prev) arrow links without any stateful tracking
func getListingPageNumber(rawURL string) int {
	u, err := url.Parse(rawURL)
	if err != nil {
		return 1
	}
	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i, seg := range segments {
		if seg == "page" && i+1 < len(segments) {
			if n, err := strconv.Atoi(segments[i+1]); err == nil {
				return n
			}
		}
	}
	return 1
}
