package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/ngshiheng/michelin-my-maps/v3/internal/scraper"
	log "github.com/sirupsen/logrus"
)

// Config holds all command-line configuration
type Config struct {
	LogLevel string
	Help     bool
	Version  bool
	Run      bool
}

// parseFlags parses command-line flags and returns a Config
func parseFlags() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.LogLevel, "log", log.InfoLevel.String(), "log level (debug, info, warning, error, fatal, panic)")
	flag.BoolVar(&cfg.Help, "help", false, "show help message")
	flag.BoolVar(&cfg.Version, "version", false, "print version information")
	flag.BoolVar(&cfg.Run, "run", false, "start the scraper")

	flag.Parse()
	return cfg
}

// printVersion prints the application version information
func printVersion() {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("Unable to determine version information.")
		return
	}

	version := "unknown"
	if buildInfo.Main.Version != "" {
		version = buildInfo.Main.Version
	}

	fmt.Printf("Version: %s\n", version)
}

// setupLogging configures the logging level and output
func setupLogging(levelStr string) error {
	level, err := log.ParseLevel(levelStr)
	if err != nil {
		return fmt.Errorf("invalid log level %q: %w", levelStr, err)
	}

	log.SetLevel(level)
	log.SetOutput(os.Stdout)
	return nil
}

// runScraper initializes and runs the scraper
func runScraper(ctx context.Context) error {
	app, err := scraper.Default()
	if err != nil {
		return fmt.Errorf("failed to create scraper: %w", err)
	}

	if err := app.Crawl(ctx); err != nil {
		return fmt.Errorf("failed to crawl: %w", err)
	}

	return nil
}

// run contains the main application logic
func run() error {
	cfg := parseFlags()

	// Handle version flag
	if cfg.Version {
		printVersion()
		return nil
	}

	// Handle help flag
	if cfg.Help {
		flag.Usage()
		return nil
	}

	// Setup logging
	if err := setupLogging(cfg.LogLevel); err != nil {
		return err
	}

	// Handle run flag or show help by default
	if cfg.Run {
		ctx := context.Background()
		return runScraper(ctx)
	}

	// Show help by default when no action flags are provided
	flag.Usage()
	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
