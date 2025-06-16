package scraper

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path"
	"path/filepath"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/gocolly/colly/v2/queue"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/config"
	log "github.com/sirupsen/logrus"
)

// WebClient provides HTTP client functionality for web scraping.
type WebClient struct {
	collector *colly.Collector
	queue     *queue.Queue
	config    *config.Config
}

// NewWebClient creates a new web client instance.
func NewWebClient(cfg *config.Config) (*WebClient, error) {
	cacheDir := filepath.Join(cfg.Cache.Path)

	c := colly.NewCollector(
		colly.CacheDir(cacheDir),
		colly.AllowedDomains(cfg.Scraper.AllowedDomain),
	)

	c.Limit(&colly.LimitRule{
		Delay:       cfg.Scraper.Delay,
		RandomDelay: cfg.Scraper.AdditionalRandomDelay,
	})

	extensions.RandomUserAgent(c)
	extensions.Referer(c)

	q, err := queue.New(
		cfg.Scraper.ThreadCount,
		&queue.InMemoryQueueStorage{MaxSize: cfg.Scraper.MaxURLs},
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

// GetCollector returns the colly collector for direct access.
func (w *WebClient) GetCollector() *colly.Collector {
	return w.collector
}

// GetQueue returns the queue for managing URLs.
func (w *WebClient) GetQueue() *queue.Queue {
	return w.queue
}

// ClearCache removes the cache file for a given colly.Request.
// by default Colly cache responses that are not 200 OK, including those with error status codes.
func (w *WebClient) ClearCache(r *colly.Request) error {
	url := r.URL.String()
	sum := sha1.Sum([]byte(url))
	hash := hex.EncodeToString(sum[:])

	cacheDir := path.Join(w.config.Cache.Path, hash[:2])
	filename := path.Join(cacheDir, hash)

	if err := os.Remove(filename); err != nil {
		log.WithFields(
			log.Fields{
				"error":    err,
				"cacheDir": cacheDir,
				"filename": filename,
				"url":      url,
			},
		).Error("failed to remove cache file")
		return err
	}

	log.WithField("url", url).Debug("cleared cache for URL")
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

// CreateDetailCollector creates a cloned collector for detail page scraping.
func (w *WebClient) CreateDetailCollector() *colly.Collector {
	dc := w.collector.Clone()
	extensions.RandomUserAgent(dc)
	extensions.Referer(dc)
	return dc
}
