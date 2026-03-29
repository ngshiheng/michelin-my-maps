package client

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
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
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/publicsuffix"
)

const cookieConfigPath = ".mym/config.json"

type cookie struct {
	Name     string     `json:"name"`
	Value    string     `json:"value"`
	Domain   string     `json:"domain"`
	Path     string     `json:"path"`
	Expires  *time.Time `json:"expires,omitempty"`
	Secure   bool       `json:"secure"`
	HttpOnly bool       `json:"http_only"`
}

type cookieConfig struct {
	Cookies []cookie `json:"cookies"`
}

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
	opts := []colly.CollectorOption{}

	if cfg.CachePath != "" {
		opts = append(opts, colly.CacheDir(filepath.Join(cfg.CachePath)))
	}

	opts = append(opts, colly.AllowedDomains(cfg.AllowedDomains...))

	c := colly.NewCollector(opts...)

	if err := c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Delay:       cfg.Delay,
		RandomDelay: cfg.RandomDelay,
	}); err != nil {
		return nil, err
	}

	extensions.RandomUserAgent(c)
	extensions.Referer(c)

	if err := attachCookies(c); err != nil {
		return nil, err
	}

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

// attachCookies loads cookies from $HOME/.mym/config.json and injects them into
// the collector's cookie jar. Returns nil (no-op) if the file does not exist.
func attachCookies(c *colly.Collector) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	f, err := os.Open(filepath.Join(home, cookieConfigPath))
	if err != nil {
		return errors.New("login is required")
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			log.WithError(cerr).Warn("failed to close cookie file")
		}
	}()

	var cfg cookieConfig
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return err
	}
	if len(cfg.Cookies) == 0 {
		return errors.New("no cookies found")
	}

	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return err
	}

	// Group cookies by domain so a single SetCookies call covers each origin.
	byDomain := make(map[string][]*http.Cookie, len(cfg.Cookies))
	for _, ck := range cfg.Cookies {
		hc := &http.Cookie{
			Name:     ck.Name,
			Value:    strings.ReplaceAll(ck.Value, `"`, ""), // strip cookie values containing double-quote chars
			Domain:   ck.Domain,
			Path:     ck.Path,
			Secure:   ck.Secure,
			HttpOnly: ck.HttpOnly,
		}
		if ck.Expires != nil {
			hc.Expires = *ck.Expires
		}
		byDomain[ck.Domain] = append(byDomain[ck.Domain], hc)
	}

	for domain, cookies := range byDomain {
		u := &url.URL{Scheme: "https", Host: domain}
		jar.SetCookies(u, cookies)
	}

	c.SetCookieJar(jar)
	return nil
}

// GetCollector returns the colly collector for direct access
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

// EnqueueURL adds a URL to the queue for processing
func (w *Colly) EnqueueURL(url string) {
	if err := w.queue.AddURL(url); err != nil {
		log.WithFields(log.Fields{
			"url":   url,
			"error": err,
		}).Warn("failed to enqueue url")
	}
}

// Run starts the web scraping process
func (w *Colly) Run() {
	if err := w.queue.Run(w.collector); err != nil {
		log.WithError(err).Warn("queue run error")
	}
}
