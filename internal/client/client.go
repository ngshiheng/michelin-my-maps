package client

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path"
	"path/filepath"

	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/gocolly/colly/v2/queue"
)

// Config defines the minimal config needed for Colly
type Config struct {
	AllowedDomains []string
	CachePath      string
	DatabasePath   string
	Delay          time.Duration
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
}

// New creates a new web client instance
func New(cfg *Config) (*Colly, error) {
	cacheDir := filepath.Join(cfg.CachePath)

	c := colly.NewCollector(
		colly.CacheDir(cacheDir),
		colly.AllowedDomains(cfg.AllowedDomains...),
	)

	c.Limit(&colly.LimitRule{
		Delay:       cfg.Delay,
		RandomDelay: cfg.RandomDelay,
	})

	extensions.RandomUserAgent(c)
	extensions.Referer(c)

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
	}, nil
}

// GetCollector returns the colly collector for direct access
func (w *Colly) GetCollector() *colly.Collector {
	return w.collector
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
	url := r.URL.String()
	sum := sha1.Sum([]byte(url))
	hash := hex.EncodeToString(sum[:])

	cacheDir := path.Join(w.config.CachePath, hash[:2])
	filename := path.Join(cacheDir, hash)

	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// EnqueueURL adds a URL to the queue for processing
func (w *Colly) EnqueueURL(url string) {
	w.queue.AddURL(url)
}

// Run starts the web scraping process
func (w *Colly) Run() {
	w.queue.Run(w.collector)
}
