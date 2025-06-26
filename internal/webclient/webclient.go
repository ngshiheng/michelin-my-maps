package webclient

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

// Config defines the minimal config needed for webClient.
type Config struct {
	CachePath      string
	AllowedDomains []string
	Delay          time.Duration
	RandomDelay    time.Duration
	ThreadCount    int
	MaxURLs        int
}

// webClient provides HTTP client functionality for web scraping.
type WebClient struct {
	collector *colly.Collector
	queue     *queue.Queue
	config    *Config
}

// New creates a new web client instance.
func New(cfg *Config) (*WebClient, error) {
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

	return &WebClient{
		collector: c,
		queue:     q,
		config:    cfg,
	}, nil
}

// GetQueue returns the queue for managing URLs.
func (w *WebClient) GetQueue() *queue.Queue {
	return w.queue
}

// GetCollector returns the colly collector for direct access.
func (w *WebClient) GetCollector() *colly.Collector {
	return w.collector
}

// CreateDetailCollector creates a cloned collector for detail page scraping.
func (w *WebClient) CreateDetailCollector() *colly.Collector {
	dc := w.collector.Clone()
	extensions.RandomUserAgent(dc)
	extensions.Referer(dc)
	return dc
}

// ClearCache removes the cache file for a given colly.Request.
func (w *WebClient) ClearCache(r *colly.Request) error {
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

// AddURL adds a URL to the scraping queue.
func (w *WebClient) AddURL(url string) {
	w.queue.AddURL(url)
}

// Run starts the web scraping process.
func (w *WebClient) Run() {
	w.queue.Run(w.collector)
}
