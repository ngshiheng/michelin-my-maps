package client

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/gocolly/colly/v2/queue"
	"github.com/gocolly/colly/v2/storage"
	"github.com/velebak/colly-sqlite3-storage/colly/sqlite3"
)

const (
	DefaultCacheScrape  = "cache/scrape"
	DefaultCacheWayback = "cache/wayback"
	DefaultDataPath     = "data/michelin.db"
	DefaultStoragePath  = "data/colly.db"
	acceptLanguage      = "en-SG,en;q=0.9"
)

// Config defines the minimal config needed for Colly
type Config struct {
	AllowedDomains []string
	CachePath      string
	DatabasePath   string
	StoragePath    string
	Delay          time.Duration
	MaxRetry       int
	RandomDelay    time.Duration
	RequestTimeout time.Duration
	ThreadCount    int
}

// Colly provides HTTP client functionality for web scraping
type Colly struct {
	Collector *colly.Collector
	Config    *Config
	Database  *sql.DB
	Queue     *queue.Queue
	Storage   *sqlite3.Storage
}

// New creates a new web client instance
func New(cfg *Config) (*Colly, error) {
	// NOTE: build collector options conditionally so cache can be disabled when CachePath is empty
	opts := []colly.CollectorOption{
		colly.Async(false), // SQLite WAL only supports one write at a time
	}

	if cfg.CachePath != "" {
		opts = append(opts, colly.CacheDir(filepath.Join(cfg.CachePath)))
	}

	opts = append(opts, colly.AllowedDomains(cfg.AllowedDomains...))

	collector := colly.NewCollector(opts...)
	if cfg.RequestTimeout > 0 {
		collector.SetRequestTimeout(cfg.RequestTimeout)
	}

	if err := collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Delay:       cfg.Delay,
		RandomDelay: cfg.RandomDelay,
	}); err != nil {
		return nil, err
	}

	extensions.RandomUserAgent(collector)
	extensions.Referer(collector)

	collyStorage := &sqlite3.Storage{Filename: cfg.StoragePath}

	err := collector.SetStorage(collyStorage)
	if err != nil {
		return nil, err
	}

	// colly-sqlite3-storage.SetCookies uses plain INSERT (not UPSERT), so
	// Set-Cookie responses accumulate stale rows; reads always return the
	// oldest row.
	// The fix here is to seed an in-memory jar from sqlite once at startup,
	// then let Go's standard jar handle all subsequent Set-Cookie updates.
	// sqlite storage continues serving visited-URL dedup and the queue.
	memJar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	for _, domain := range cfg.AllowedDomains {
		u := &url.URL{Scheme: "https", Host: domain}
		if raw := collyStorage.Cookies(u); raw != "" {
			memJar.SetCookies(u, storage.UnstringifyCookies(raw))
		}
	}
	collector.SetCookieJar(memJar)

	queue, err := queue.New(
		cfg.ThreadCount,
		collyStorage,
	)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", cfg.StoragePath)
	if err != nil {
		return nil, err
	}

	return &Colly{
		Collector: collector,
		Queue:     queue,
		Storage:   collyStorage,
		Config:    cfg,
		Database:  db,
	}, nil
}

// GetCollector returns the colly collector for direct access.
func (w *Colly) GetCollector() *colly.Collector {
	return w.Collector
}

// GetCookies returns a map of cookie name->value for the given URL as seen by
// the collector's cookie jar.
func (w *Colly) GetCookies(urlStr string) map[string]string {
	out := make(map[string]string)
	if w == nil || w.Collector == nil {
		return out
	}
	cookies := w.Collector.Cookies(urlStr)
	for _, c := range cookies {
		out[c.Name] = c.Value
	}
	return out
}

// GetDetailCollector creates a cloned collector for detail page scraping
func (w *Colly) GetDetailCollector() *colly.Collector {
	dc := w.Collector.Clone()
	extensions.RandomUserAgent(dc)
	extensions.Referer(dc)
	return dc
}

// ClearCache removes the cache file for a given colly.Request
func (w *Colly) ClearCache(r *colly.Request) error {
	if w.Config == nil || w.Config.CachePath == "" {
		return nil
	}

	sum := sha1.Sum([]byte(r.URL.String()))
	hash := hex.EncodeToString(sum[:])
	filename := path.Join(w.Config.CachePath, hash[:2], hash)

	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// IsCached reports whether cache is enabled and whether a URL exists in cache.
func (w *Colly) IsCached(urlStr string) (cacheEnabled bool, cacheHit bool) {
	if w == nil || w.Config == nil || strings.TrimSpace(w.Config.CachePath) == "" {
		return false, false
	}

	sum := sha1.Sum([]byte(urlStr))
	hash := hex.EncodeToString(sum[:])
	filename := path.Join(w.Config.CachePath, hash[:2], hash)

	if _, err := os.Stat(filename); err == nil {
		return true, true
	}
	return true, false
}

// ClearVisited removes all rows from the visited table so that a fresh Phase 1
// run can re-visit seed listing pages that were marked visited in a prior completed run.
func (w *Colly) ClearVisited() error {
	db, err := sql.Open("sqlite3", w.Config.StoragePath)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec("DELETE FROM visited")
	return err
}

// ClearCookies removes persisted cookies without touching queue/visited state.
// If host is empty, all cookie rows are removed.
func (w *Colly) ClearCookies(host string) error {
	db, err := sql.Open("sqlite3", w.Config.StoragePath)
	if err != nil {
		return err
	}
	defer db.Close()

	if host == "" {
		_, err = db.Exec("DELETE FROM cookies")
		return err
	}

	_, err = db.Exec("DELETE FROM cookies WHERE host = ?", host)
	return err
}

// EnqueueURLWithContext enqueues a GET request that carries a colly.Context
// (e.g. location) through the queue boundary into the next phase.
// queue.AddURL cannot be used here because it always creates a bare request
// with no context; serializing a Request directly is the only way to preserve
// the extra fields across the SQLite queue.
func (w *Colly) EnqueueURLWithContext(rawURL, location string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	ctx := colly.NewContext()
	ctx.Put("location", location)
	r := &colly.Request{
		URL:    u,
		Method: "GET",
		Ctx:    ctx,
	}
	data, err := r.Marshal()
	if err != nil {
		return err
	}
	return w.Storage.AddRequest(data)
}

// PrepareRequest applies shared request context fields for scraper collectors.
func PrepareRequest(ctx context.Context, r *colly.Request, cacheHit bool) (attempt any, aborted bool) {
	if ctx.Err() != nil {
		r.Abort()
		return nil, true
	}

	r.Headers.Set("Accept-Language", acceptLanguage)

	attempt = r.Ctx.GetAny("attempt")
	if attempt == nil {
		r.Ctx.Put("attempt", 1)
		attempt = 1
	}

	r.Ctx.Put("cache_hit", cacheHit)
	return attempt, false
}
