package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/ngshiheng/michelin-my-maps/v3/internal/scraper"
	log "github.com/sirupsen/logrus"
)

// run contains the main application logic
func run() error {
	// Handle global flags first (before subcommands)
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-version":
			printVersion()
			return nil
		case "--help", "-help":
			printUsage()
			return nil
		}
	}

	// Check if we have at least one argument (the subcommand)
	if len(os.Args) < 2 {
		printUsage()
		return nil
	}

	// Get the subcommand
	command := os.Args[1]
	commandArgs := os.Args[2:]

	// Handle subcommands
	switch command {
	case "run":
		return handleRunCommand(commandArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
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
	fmt.Println("Commands:")
	fmt.Println("  run    start the scraper")
	fmt.Println("")
	fmt.Println("Global options:")
	fmt.Println("  -help     show help message")
	fmt.Println("  -version  print version information")
	fmt.Println("")
	fmt.Println("Run command options:")
	fmt.Println("  -log <level> (\"debug\", \"info\", \"warning\", \"error\", \"fatal\", \"panic\") (default: \"info\")")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Printf("  %s run             # start the scraper\n", os.Args[0])
	fmt.Printf("  %s run -log debug  # start with debug logging\n", os.Args[0])
}

// handleRunCommand handles the 'run' subcommand with its own flag set
func handleRunCommand(args []string) error {
	// Create a new flag set for the run command
	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	logLevel := runCmd.String("log", log.InfoLevel.String(), "log level (debug, info, warning, error, fatal, panic)")

	// Parse the run command flags
	if err := runCmd.Parse(args); err != nil {
		return err
	}

	// Setup logging
	if err := setupLogging(*logLevel); err != nil {
		return err
	}

	// Run the scraper
	ctx := context.Background()
	return runScraper(ctx)
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

func main() {
	os.Setenv("TZ", "UTC")
	time.Local = time.UTC
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
