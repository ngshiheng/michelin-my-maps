package client

import (
	"crypto/sha1"
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
	log "github.com/sirupsen/logrus"
	"github.com/velebak/colly-sqlite3-storage/colly/sqlite3"
)

const (
	DefaultCacheScrape  = "cache/scrape"
	DefaultCacheWayback = "cache/wayback"
	DefaultDataPath     = "data/michelin.db"
	DefaultStoragePath  = "data/colly.db"
)

// Config defines the minimal config needed for Colly
type Config struct {
	AllowedDomains  []string
	AllowURLRevisit bool
	CachePath       string
	DatabasePath    string
	StoragePath     string
	Delay           time.Duration
	MaxRetry        int
	RandomDelay     time.Duration
	RequestTimeout  time.Duration
	ThreadCount     int
}

// Colly provides HTTP client functionality for web scraping
type Colly struct {
	collector *colly.Collector
	queue     *queue.Queue
	storage   *sqlite3.Storage
	config    *Config
}

// New creates a new web client instance
func New(cfg *Config) (*Colly, error) {
	// We build collector options conditionally so cache can be disabled when CachePath is empty
	opts := []colly.CollectorOption{
		colly.Async(false),      // SQLite only support one write at a time
		colly.AllowURLRevisit(), // disables colly's internal URL dedup so that it won't mess with our cache
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

	return &Colly{
		collector: collector,
		queue:     queue,
		storage:   collyStorage,
		config:    cfg,
	}, nil
}

// GetCollector returns the colly collector for direct access.
func (w *Colly) GetCollector() *colly.Collector {
	return w.collector
}

// GetCookies returns a map of cookie name->value for the given URL as seen by
// the collector's cookie jar.
func (w *Colly) GetCookies(urlStr string) map[string]string {
	out := make(map[string]string)
	if w == nil || w.collector == nil {
		return out
	}
	cookies := w.collector.Cookies(urlStr)
	for _, c := range cookies {
		out[c.Name] = c.Value
	}
	return out
}

// GetDetailCollector creates a cloned collector for detail page scraping
func (w *Colly) GetDetailCollector() *colly.Collector {
	dc := w.collector.Clone()
	extensions.RandomUserAgent(dc)
	extensions.Referer(dc)
	return dc
}

// ClearCache removes the cache file for a given colly.Request
func (w *Colly) ClearCache(r *colly.Request) error {
	if w.config == nil || w.config.CachePath == "" {
		return nil
	}

	sum := sha1.Sum([]byte(r.URL.String()))
	hash := hex.EncodeToString(sum[:])
	filename := path.Join(w.config.CachePath, hash[:2], hash)

	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// IsCached reports whether cache is enabled and whether a URL exists in cache.
func (w *Colly) IsCached(urlStr string) (cacheEnabled bool, cacheHit bool) {
	if w == nil || w.config == nil || strings.TrimSpace(w.config.CachePath) == "" {
		return false, false
	}

	sum := sha1.Sum([]byte(urlStr))
	hash := hex.EncodeToString(sum[:])
	filename := path.Join(w.config.CachePath, hash[:2], hash)

	if _, err := os.Stat(filename); err == nil {
		return true, true
	}
	return true, false
}

// EnqueueURL adds a URL to the queue for processing
func (w *Colly) EnqueueURL(url string) error {
	if err := w.queue.AddURL(url); err != nil {
		log.WithError(err).WithField("url", url).Warn("failed to enqueue url")
		return err
	}
	return nil
}

// RunQueue drains the queue by dispatching each request to dc
func (w *Colly) RunQueue(dc *colly.Collector) error {
	if err := w.queue.Run(dc); err != nil {
		log.WithError(err).Warn("failed to run queue")
		return err
	}
	return nil
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
	return w.storage.AddRequest(data)
}
