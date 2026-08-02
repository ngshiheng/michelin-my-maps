package backfill

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/ngshiheng/michelin-my-maps/v4/internal/client"
	"github.com/ngshiheng/michelin-my-maps/v4/internal/handlers"
	"github.com/ngshiheng/michelin-my-maps/v4/internal/storage"
	"github.com/ngshiheng/michelin-my-maps/v4/internal/utils"
)

const xPathDetailRoot = "html"

func defaultConfig() *client.Config {
	return &client.Config{
		AllowedDomains: []string{"web.archive.org"},
		CachePath:      client.DefaultCacheWayback,
		DatabasePath:   client.DefaultDataPath,
		StoragePath:    client.DefaultStoragePath,
		// Wayback CDX guidance is < 60 requests/minute. 1.0-1.5s pacing
		// yields ~40-60 req/minute with jitter while remaining conservative.
		Delay:          1 * time.Second,
		MaxRetry:       3,
		RandomDelay:    500 * time.Millisecond,
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

// RunAll runs the backfill workflow for all restaurants
func (s *Scraper) RunAll(ctx context.Context) error {
	restaurants, err := s.repository.ListRestaurants(ctx)
	if err != nil {
		return fmt.Errorf("failed to list restaurants: %w", err)
	}

	slog.Info("running backfill for restaurants", "count", len(restaurants))

	collector := s.client.GetCollector()
	detailCollector := s.client.GetDetailCollector()

	s.setupHandlers(ctx, collector, detailCollector)
	s.setupDetailHandlers(ctx, detailCollector)

	for _, r := range restaurants {
		if err := ctx.Err(); err != nil {
			return err
		}
		api := "https://web.archive.org/cdx/search/cdx?url=" + r.URL + "&output=json&fl=timestamp,original"
		if err := s.client.EnqueueURL(api); err != nil {
			return err
		}
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.client.RunQueue(collector); err != nil {
		return err
	}

	// TODO: add summary of results
	slog.Info("completed backfill", "scraped", s.scraped.Load())
	return nil
}

// Run runs the backfill workflow for a single restaurant URL
func (s *Scraper) Run(ctx context.Context, url string) error {
	slog.Debug("running backfill for restaurant", "url", url)

	collector := s.client.GetCollector()
	detailCollector := s.client.GetDetailCollector()

	s.setupHandlers(ctx, collector, detailCollector)
	s.setupDetailHandlers(ctx, detailCollector)

	if err := ctx.Err(); err != nil {
		return err
	}
	api := "https://web.archive.org/cdx/search/cdx?url=" + url + "&output=json&fl=timestamp,original"
	if err := collector.Visit(api); err != nil {
		slog.Error("failed to visit restaurant URL", "error", err, "url", url)
		return err
	}

	slog.Debug("completed backfill for one restaurant", "url", url)
	return nil
}

func (s *Scraper) setupHandlers(ctx context.Context, collector *colly.Collector, detailCollector *colly.Collector) {
	collector.OnError(s.createErrorHandler(ctx))

	collector.OnRequest(func(r *colly.Request) {
		_, cacheHit := s.client.IsCached(r.URL.String())
		attempt, aborted := client.PrepareRequest(ctx, r, cacheHit)
		if aborted {
			return
		}

		slog.Debug("requesting cdx api", "attempt", attempt, "cache_hit", cacheHit, "url", r.URL)
	})

	collector.OnResponse(func(r *colly.Response) {
		if ctx.Err() != nil {
			return
		}

		url := r.Request.URL.Query().Get("url")

		var rows [][]string
		if err := json.Unmarshal(r.Body, &rows); err != nil {
			slog.Warn("failed to parse cdx api response", "error", err, "url", url, "status_code", r.StatusCode, "cdx_api", r.Request.URL)
			return
		}

		if len(rows) <= 1 {
			slog.Debug("no snapshots found", "url", url, "rows", rows, "status_code", r.StatusCode, "cdx_api", r.Request.URL)
			// FIXME: this is currently cached because CDX api returns 200
			return
		}

		minTimestampLen := 14
		snapshot := 0
		for i, row := range rows {
			if ctx.Err() != nil {
				return
			}
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
				slog.Debug("failed to visit snapshot URL", "error", err, "url", url, "wayback_url", snapshotURL)
				continue
			}
			snapshot++
		}

		slog.Debug("processing cdx api", "cache_hit", r.Ctx.GetAny("cache_hit"), "cdx_api", r.Request.URL, "snapshot", snapshot, "status_code", r.StatusCode, "url", url)
	})
}

func (s *Scraper) setupDetailHandlers(ctx context.Context, detailCollector *colly.Collector) {
	detailCollector.OnError(s.createErrorHandler(ctx))

	detailCollector.OnRequest(func(r *colly.Request) {
		_, cacheHit := s.client.IsCached(r.URL.String())
		attempt, aborted := client.PrepareRequest(ctx, r, cacheHit)
		if aborted {
			return
		}

		slog.Debug("requesting wayback snapshot", "attempt", attempt, "cache_hit", cacheHit, "url", r.URL)
	})

	detailCollector.OnXML(xPathDetailRoot, func(e *colly.XMLElement) {
		if ctx.Err() != nil {
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

		if errors.Is(err, context.Canceled) {
			slog.Debug("context canceled, skip retry", append(utils.FieldsToArgs(fields), "error", err)...)
			return
		}

		// We don't retry 403 Forbidden errors, as they indicate restricted access and retries won't help.
		// In the Wayback Machine, a 403 typically means the site owner has blocked archiving.
		switch r.StatusCode {
		case http.StatusForbidden:
			slog.Debug("request forbidden, skip retry", append(utils.FieldsToArgs(fields), "error", err)...)
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
