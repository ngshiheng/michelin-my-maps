package config

import (
	"time"
)

// Config holds all configuration for the scraper application.
type Config struct {
	Database Database
	Scraper  Scraper
	Cache    Cache
}

// Database holds database-related configuration.
type Database struct {
	Path string
}

// Scraper holds scraper-related configuration.
type Scraper struct {
	AllowedDomain         string
	Delay                 time.Duration
	AdditionalRandomDelay time.Duration
	MaxRetry              int
	ThreadCount           int
	MaxURLs               int
}

// Cache holds cache-related configuration.
type Cache struct {
	Path string
}

// Default returns a configuration with default values.
func Default() *Config {
	return &Config{
		Database: Database{
			Path: "data/michelin.db",
		},
		Scraper: Scraper{
			AllowedDomain:         "guide.michelin.com",
			Delay:                 5 * time.Second,
			AdditionalRandomDelay: 5 * time.Second,
			MaxRetry:              3,
			ThreadCount:           1,
			MaxURLs:               30000, // There are currently ~17k restaurants on Michelin Guide as of Jun 2025
		},
		Cache: Cache{
			Path: "cache",
		},
	}
}
