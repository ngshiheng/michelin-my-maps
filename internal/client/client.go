package client

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
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
	AllowedDomains []string
	CachePath      string
	DatabasePath   string
	StoragePath    string
	Delay          time.Duration
	RequestTimeout time.Duration
	MaxRetry       int
	MaxURLs        int
	RandomDelay    time.Duration
	ThreadCount    int
}

// Colly provides HTTP client functionality for web scraping
type Colly struct {
	collector *colly.Collector
	queue     *queue.Queue
	config    *Config
	storage   storage.Storage
}

// NewSQLiteStorage creates and initializes the sqlite storage backend used by Colly
func NewSQLiteStorage(storagePath string) (storage.Storage, error) {
	if strings.TrimSpace(storagePath) == "" {
		storagePath = DefaultStoragePath
	}

	dir := filepath.Dir(storagePath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, err
	}

	store := &sqlite3.Storage{Filename: storagePath}
	if err := store.Init(); err != nil {
		return nil, err
	}
	return store, nil
}

// SaveCookies persists cookies to the given storage under the specified host.
// Existing rows are cleared first since the sqlite3 backend uses plain INSERT (not upsert).
func SaveCookies(store storage.Storage, host string, cookies []*http.Cookie) error {
	if clearable, ok := store.(interface{ Clear() error }); ok {
		if err := clearable.Clear(); err != nil {
			log.WithError(err).Warn("failed to clear storage before seeding cookies")
		}
	}
	if err := store.Init(); err != nil {
		return fmt.Errorf("failed to reinitialize storage: %w", err)
	}
	u := &url.URL{Host: host}
	lines := make([]string, len(cookies))
	for i, c := range cookies {
		lines[i] = c.String()
	}
	store.SetCookies(u, strings.Join(lines, "\n"))
	return nil
}

// Storage returns the underlying storage backend.
func (w *Colly) Storage() storage.Storage {
	return w.storage
}

// New creates a new web client instance
func New(cfg *Config) (*Colly, error) {
	opts := []colly.CollectorOption{
		colly.AllowURLRevisit(), // disables colly's internal URL dedup
	}

	if cfg.CachePath != "" {
		opts = append(opts, colly.CacheDir(filepath.Join(cfg.CachePath)))
	}

	opts = append(opts, colly.AllowedDomains(cfg.AllowedDomains...))

	c := colly.NewCollector(opts...)
	if cfg.RequestTimeout > 0 {
		c.SetRequestTimeout(cfg.RequestTimeout)
	}

	if err := c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Delay:       cfg.Delay,
		RandomDelay: cfg.RandomDelay,
	}); err != nil {
		return nil, err
	}

	extensions.RandomUserAgent(c)
	extensions.Referer(c)

	store, err := NewSQLiteStorage(cfg.StoragePath)
	if err != nil {
		return nil, err
	}
	c.SetStorage(store)

	q, err := queue.New(
		cfg.ThreadCount,
		&queue.InMemoryQueueStorage{MaxSize: cfg.MaxURLs},
	)
	if err != nil {
		return nil, err
	}

	return &Colly{
		collector: c,
		queue:     q,
		config:    cfg,
		storage:   store,
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
		log.WithFields(log.Fields{
			"url":   url,
			"error": err,
		}).Warn("failed to enqueue url")
		return err
	}
	return nil
}

// Run starts the web scraping process
func (w *Colly) Run() error {
	if err := w.queue.Run(w.collector); err != nil {
		log.WithError(err).Warn("queue run error")
		return err
	}
	return nil
}
