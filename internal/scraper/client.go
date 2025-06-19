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
)

// webClient provides HTTP client functionality for web scraping.
type webClient struct {
	collector *colly.Collector
	queue     *queue.Queue
	config    *config.Config
}

// newWebClient creates a new web client instance.
func newWebClient(cfg *config.Config) (*webClient, error) {
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

	return &webClient{
		collector: c,
		queue:     q,
		config:    cfg,
	}, nil
}

// getQueue returns the queue for managing URLs.
func (w *webClient) getQueue() *queue.Queue {
	return w.queue
}

// getCollector returns the colly collector for direct access.
func (w *webClient) getCollector() *colly.Collector {
	return w.collector
}

// createDetailCollector creates a cloned collector for detail page scraping.
func (w *webClient) createDetailCollector() *colly.Collector {
	dc := w.collector.Clone()
	extensions.RandomUserAgent(dc)
	extensions.Referer(dc)
	return dc
}

// clearCache removes the cache file for a given colly.Request.
// by default Colly cache responses that are not 200 OK, including those with error status codes.
func (w *webClient) clearCache(r *colly.Request) error {
	url := r.URL.String()
	sum := sha1.Sum([]byte(url))
	hash := hex.EncodeToString(sum[:])

	cacheDir := path.Join(w.config.Cache.Path, hash[:2])
	filename := path.Join(cacheDir, hash)

	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// addURL adds a URL to the scraping queue.
func (w *webClient) addURL(url string) {
	w.queue.AddURL(url)
}

// run starts the web scraping process.
func (w *webClient) run() {
	w.queue.Run(w.collector)
}
