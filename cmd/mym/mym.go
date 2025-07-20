package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/ngshiheng/michelin-my-maps/v3/internal/backfill"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/scraper"
	log "github.com/sirupsen/logrus"
)

const (
	versionLongFlag  = "--version"
	versionShortFlag = "-v"
	helpLongFlag     = "--help"
	helpShortFlag    = "-h"
)

// run contains the main application logic
func run() error {
	if len(os.Args) < 2 {
		printUsage()
		return nil
	}

	arg := os.Args[1]
	switch arg {
	case versionLongFlag, versionShortFlag:
		printVersion()
		return nil
	case helpLongFlag, helpShortFlag:
		printUsage()
		return nil
	default:
		return handleCommand(os.Args)
	}
}

// handleCommand processes the main command and its subcommands
func handleCommand(arg []string) error {
	command := arg[1]

	switch command {
	case "scrape":
		return handleScrape(arg[2:])
	case "backfill":
		return handleBackfill(arg[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: \"%s\"\n\n", command)
		printUsage()
		return nil
	}
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
	fmt.Printf("Usage: %s <command> [options]\n\n", os.Args[0])
	fmt.Println("<command>")
	fmt.Println("  scrape     scrape latest restaurant data.")
	fmt.Println("  backfill   backfill restaurant data.")
	fmt.Println("")
	fmt.Println("[options]")
	fmt.Println("  -log <level>   set log level. (default: info)")
	fmt.Println("  -help          show help.")
	fmt.Println("  -version       show version.")
	fmt.Println("")
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

// handleScrape handles the 'scrape' subcommand with its own flag set
func handleScrape(args []string) error {
	scrapeCmd := flag.NewFlagSet("scrape", flag.ExitOnError)
	logLevel := scrapeCmd.String("log", log.InfoLevel.String(), "log level (debug, info, warning, error, fatal, panic)")

	if err := scrapeCmd.Parse(args); err != nil {
		return err
	}

	if err := setupLogging(*logLevel); err != nil {
		return err
	}

	log.Info("starting scrape command")
	ctx := context.Background()
	app, err := scraper.New()
	if err != nil {
		return fmt.Errorf("failed to create scraper: %w", err)
	}

	if err := app.Run(ctx); err != nil {
		return fmt.Errorf("failed to run scraper: %w", err)
	}

	return nil
}

// handleBackfill handles the 'backfill' subcommand with its own flag set
func handleBackfill(args []string) error {
	backfillCmd := flag.NewFlagSet("backfill", flag.ExitOnError)
	logLevel := backfillCmd.String("log", log.InfoLevel.String(), "log level (debug, info, warning, error, fatal, panic)")

	// Parse flags, remaining args may include [url]
	if err := backfillCmd.Parse(args); err != nil {
		return err
	}
	if err := setupLogging(*logLevel); err != nil {
		return err
	}

	log.Info("starting backfill command")
	var urlFilter string
	rest := backfillCmd.Args()
	if len(rest) > 0 {
		urlFilter = rest[0]
	}

	ctx := context.Background()
	bs, err := backfill.New()
	if err != nil {
		return fmt.Errorf("failed to create backfill scraper: %w", err)
	}
	return bs.Run(ctx, urlFilter)
}

func main() {
	os.Setenv("TZ", "UTC")
	time.Local = time.UTC
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
