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
	helpLongFlag     = "--help"
	helpShortFlag    = "-h"
	versionLongFlag  = "--version"
	versionShortFlag = "-v"
)

const (
	commandBackfill = "backfill"
	commandScrape   = "scrape"
)

// run contains the main application logic of the CLI tool
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
	case commandScrape:
		return handleScrape(arg[2:])
	case commandBackfill:
		return handleBackfill(arg[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: \"%s\"\n\n", command)
		printUsage()
		return nil
	}
}

// printVersion prints the application version information
func printVersion() {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("unable to determine build information.")
		return
	}

	version := "development"
	if buildInfo.Main.Version != "" {
		version = buildInfo.Main.Version
	}

	fmt.Printf("version: %s\n", version)
}

// printUsage prints the custom usage message
func printUsage() {
	fmt.Printf("usage: %s <command> [options]\n\n", os.Args[0])
	fmt.Println("<command>")
	fmt.Println("  scrape     scrape latest restaurant data or a single restaurant if <url> is provided.")
	fmt.Println("  backfill   backfill restaurant data or a single restaurant if <url> is provided.")
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

// handleScrape handles the 'scrape' subcommand
func handleScrape(args []string) error {
	scrapeCmd := flag.NewFlagSet(commandScrape, flag.ExitOnError)
	logLevel := scrapeCmd.String("log", log.InfoLevel.String(), "log level (debug, info, warning, error, fatal, panic)")

	if err := scrapeCmd.Parse(args); err != nil {
		return err
	}

	if err := setupLogging(*logLevel); err != nil {
		return err
	}

	urlArg := scrapeCmd.Arg(0)

	app, err := scraper.New()
	if err != nil {
		return fmt.Errorf("failed to create live scraper: %w", err)
	}

	log.Info("starting scrape command")
	ctx := context.Background()
	if urlArg != "" {
		return app.Run(ctx, urlArg)
	}
	return app.RunAll(ctx)
}

// handleBackfill handles the 'backfill' subcommand
func handleBackfill(args []string) error {
	backfillCmd := flag.NewFlagSet(commandBackfill, flag.ExitOnError)
	logLevel := backfillCmd.String("log", log.InfoLevel.String(), "log level (debug, info, warning, error, fatal, panic)")

	if err := backfillCmd.Parse(args); err != nil {
		return err
	}
	if err := setupLogging(*logLevel); err != nil {
		return err
	}

	urlArg := backfillCmd.Arg(0)

	app, err := backfill.New()
	if err != nil {
		return fmt.Errorf("failed to create backfill scraper: %w", err)
	}

	log.Info("starting backfill command")
	ctx := context.Background()
	if urlArg != "" {
		return app.Run(ctx, urlArg)
	}
	return app.RunAll(ctx)
}

// main is the entry point for the mym CLI tool
func main() {
	os.Setenv("TZ", time.UTC.String())
	time.Local = time.UTC

	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
