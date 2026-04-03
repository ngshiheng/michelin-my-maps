package backfill

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/ngshiheng/michelin-my-maps/v4/internal/client"
	"github.com/ngshiheng/michelin-my-maps/v4/internal/handlers"
	"github.com/ngshiheng/michelin-my-maps/v4/internal/storage"
	"github.com/ngshiheng/michelin-my-maps/v4/internal/utils"
	log "github.com/sirupsen/logrus"
)

const xPathDetailRoot = "html"

// defaultConfig returns a default config for Wayback backfill
func defaultConfig() *client.Config {
	return &client.Config{
		AllowedDomains: []string{"web.archive.org"},
		CachePath:      client.DefaultCacheWayback,
		DatabasePath:   client.DefaultDataPath,
		StoragePath:    client.DefaultStoragePath,
		// Wayback CDX guidance is < 60 requests/minute. 1.0-1.5s pacing
		// yields ~40-60 req/minute with jitter while remaining conservative.
		Delay:       1 * time.Second,
		MaxRetry:    3,
		RandomDelay: 500 * time.Millisecond,
		// 2 threads provides useful queue parallelism (two restaurants looked
		// up simultaneously) while keeping the aggregate request rate
		// below Wayback's limits.
		ThreadCount:    2,
		RequestTimeout: 20 * time.Second,
	}
}

// Scraper orchestrates the Wayback backfill process
type Scraper struct {
	client     *client.Colly
	config     *client.Config
	repository storage.RestaurantRepository
	scraped    atomic.Int64
}

// New creates a new Scraper with default config and repository
func New(ignoreCache bool) (*Scraper, error) {
	cfg := defaultConfig()

	repo, err := storage.NewSQLiteRepository(cfg.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	clientCfg := &client.Config{
		AllowedDomains: cfg.AllowedDomains,
		CachePath:      cfg.CachePath,
		StoragePath:    cfg.StoragePath,
		Delay:          cfg.Delay,
		RequestTimeout: cfg.RequestTimeout,
		MaxRetry:       cfg.MaxRetry,
		RandomDelay:    cfg.RandomDelay,
		ThreadCount:    cfg.ThreadCount,
	}
	if ignoreCache {
		log.Debug("running with no cache")
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

// RunAll runs the backfill workflow for all restaurants
func (s *Scraper) RunAll(ctx context.Context) error {
	restaurants, err := s.repository.ListRestaurants(ctx)
	if err != nil {
		return fmt.Errorf("failed to list restaurants: %w", err)
	}

	log.WithFields(log.Fields{
		"count": len(restaurants),
	}).Info("running backfill for restaurants")

	collector := s.client.GetCollector()
	detailCollector := s.client.GetDetailCollector()

	s.setupHandlers(collector, detailCollector)
	s.setupDetailHandlers(ctx, detailCollector)

	for _, r := range restaurants {
		api := "https://web.archive.org/cdx/search/cdx?url=" + r.URL + "&output=json&fl=timestamp,original"
		if err := s.client.EnqueueURL(api); err != nil {
			return err
		}
	}

	if err := s.client.RunQueue(collector); err != nil {
		return err
	}

	// TODO: add summary of results
	log.WithField("scraped", s.scraped.Load()).Info("completed backfill")
	return nil
}

// Run runs the backfill workflow for a single restaurant URL
func (s *Scraper) Run(ctx context.Context, url string) error {
	log.WithField("url", url).Debug("running backfill for restaurant")

	collector := s.client.GetCollector()
	detailCollector := s.client.GetDetailCollector()

	s.setupHandlers(collector, detailCollector)
	s.setupDetailHandlers(ctx, detailCollector)

	api := "https://web.archive.org/cdx/search/cdx?url=" + url + "&output=json&fl=timestamp,original"
	if err := collector.Visit(api); err != nil {
		log.WithError(err).WithField("url", url).Error("failed to visit restaurant URL")
		return err
	}

	log.WithField("url", url).Debug("completed backfill for one restaurant")
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
		_, cacheHit := s.client.IsCached(r.URL.String())

		r.Ctx.Put("cache_hit", cacheHit)

		log.WithFields(log.Fields{
			"attempt":   attempt,
			"cache_hit": cacheHit,
			"url":       r.URL,
		}).Debug("requesting cdx api")
	})

	collector.OnResponse(func(r *colly.Response) {
		url := r.Request.URL.Query().Get("url")

		var rows [][]string
		if err := json.Unmarshal(r.Body, &rows); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"url":         url,
				"status_code": r.StatusCode,
				"cdx_api":     r.Request.URL,
			}).Warn("failed to parse cdx api response")
			return
		}

		if len(rows) <= 1 {
			log.WithFields(log.Fields{
				"url":         url,
				"rows":        rows,
				"status_code": r.StatusCode,
				"cdx_api":     r.Request.URL,
			}).Debug("no snapshots found")
			// FIXME: this is currently cached because CDX api returns 200
			return
		}

		minTimestampLen := 14
		snapshot := 0
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
				log.WithError(err).WithFields(log.Fields{
					"url":         url,
					"wayback_url": snapshotURL,
				}).Debug("failed to visit snapshot URL")
				continue
			}
			snapshot++
		}

		log.WithFields(log.Fields{
			"cache_hit":   r.Ctx.GetAny("cache_hit"),
			"cdx_api":     r.Request.URL,
			"snapshot":    snapshot,
			"status_code": r.StatusCode,
			"url":         url,
		}).Debug("processing cdx api")
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
		_, cacheHit := s.client.IsCached(r.URL.String())
		r.Ctx.Put("cache_hit", cacheHit)

		log.WithFields(log.Fields{
			"attempt":   attempt,
			"cache_hit": cacheHit,
			"url":       r.URL,
		}).Debug("requesting wayback snapshot")
	})

	detailCollector.OnXML(xPathDetailRoot, func(e *colly.XMLElement) {
		err := handlers.Handle(ctx, e, s.repository)
		if err != nil {
			log.WithError(err).WithField("url", e.Request.URL).Error("failed to handle restaurant extraction")
			return
		}
		if n := s.scraped.Add(1); n%100 == 0 {
			log.WithField("scraped", n).Info("backfill in progress")
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

		// We don't retry 403 Forbidden errors, as they indicate restricted access and retries won't help.
		// In the Wayback Machine, a 403 typically means the site owner has blocked archiving.
		switch r.StatusCode {
		case http.StatusForbidden:
			log.WithError(err).WithFields(fields).Debug("request forbidden, skip retry")
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
