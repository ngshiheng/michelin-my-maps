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
}

// parseGlobalFlags parses global flags and returns a Config
func parseGlobalFlags() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.LogLevel, "log", log.InfoLevel.String(),
		"log level (debug, info, warning, error, fatal, panic)")
	flag.BoolVar(&cfg.Help, "help", false, "show help message")
	flag.BoolVar(&cfg.Version, "version", false, "print version information")

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

// printUsage prints the custom usage message
func printUsage() {
	fmt.Printf("Usage: %s [global options] <command>\n\n", os.Args[0])
	fmt.Println("Commands:")
	fmt.Println("  run    start the scraper")
	fmt.Println("")
	fmt.Println("Global options:")
	flag.PrintDefaults()
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Printf("  %s run              # start the scraper\n", os.Args[0])
	fmt.Printf("  %s run --log debug  # start with debug logging\n", os.Args[0])
	fmt.Printf("  %s --version        # show version\n", os.Args[0])
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

// handleRunCommand handles the 'run' subcommand
func handleRunCommand(cfg *Config) error {
	// Setup logging
	if err := setupLogging(cfg.LogLevel); err != nil {
		return err
	}

	// Run the scraper
	ctx := context.Background()
	return runScraper(ctx)
}

// run contains the main application logic
func run() error {
	cfg := parseGlobalFlags()

	// Custom usage function
	flag.Usage = printUsage

	// Parse global flags first
	flag.Parse()

	// Handle version flag
	if cfg.Version {
		printVersion()
		return nil
	}

	// Handle help flag
	if cfg.Help {
		printUsage()
		return nil
	}

	// Get remaining arguments (subcommands)
	args := flag.Args()

	// If no subcommand provided, show usage
	if len(args) == 0 {
		printUsage()
		return nil
	}

	// Handle subcommands
	switch args[0] {
	case "run":
		return handleRunCommand(cfg)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", args[0])
		printUsage()
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
