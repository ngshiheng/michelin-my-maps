package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ngshiheng/michelin-my-maps/v3/internal/backfill"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/models"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/scraper"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/storage"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/trip"
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
	commandTrip     = "trip"
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
	case commandTrip:
		return handleTrip(arg[2:])
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
	fmt.Println("  trip       manage travel restaurant maps.")
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

// handleTrip handles the 'trip' subcommand
func handleTrip(args []string) error {
	if len(args) == 0 {
		printTripUsage()
		return nil
	}

	subcommand := args[0]
	switch subcommand {
	case "create":
		return handleTripCreate(args[1:])
	case "list":
		return handleTripList(args[1:])
	case "delete":
		return handleTripDelete(args[1:])
	case "export":
		return handleTripExport(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown trip subcommand: \"%s\"\n\n", subcommand)
		printTripUsage()
		return nil
	}
}

// printTripUsage prints usage information for the trip command
func printTripUsage() {
	fmt.Printf("usage: %s trip <subcommand> [options]\n\n", os.Args[0])
	fmt.Println("<subcommand>")
	fmt.Println("  create  <name>  create a new trip with filters")
	fmt.Println("  list            list all saved trips")
	fmt.Println("  delete  <name>  delete a trip")
	fmt.Println("  export  <name>  export trip to map formats (geojson, kml, html)")
	fmt.Println("")
	fmt.Println("create options:")
	fmt.Println("  -cities <list>       comma-separated list of cities (e.g., 'Paris,Lyon')")
	fmt.Println("  -distinctions <list> comma-separated distinctions (e.g., '3 Stars,2 Stars,1 Star,Bib Gourmand')")
	fmt.Println("  -min-stars <n>       minimum star rating: 1, 2, or 3")
	fmt.Println("  -cuisines <list>     comma-separated cuisines (e.g., 'French,Italian')")
	fmt.Println("  -green-star          only include green star restaurants")
	fmt.Println("")
	fmt.Println("export options:")
	fmt.Println("  -format <fmt>        export format: geojson, kml, html, or all (default: all)")
	fmt.Println("  -output <dir>        output directory (default: current directory)")
	fmt.Println("")
}

// handleTripCreate creates a new trip
func handleTripCreate(args []string) error {
	createCmd := flag.NewFlagSet("create", flag.ExitOnError)
	cities := createCmd.String("cities", "", "comma-separated list of cities")
	distinctions := createCmd.String("distinctions", "", "comma-separated distinctions")
	minStars := createCmd.Int("min-stars", 0, "minimum star rating")
	cuisines := createCmd.String("cuisines", "", "comma-separated cuisines")
	greenStar := createCmd.Bool("green-star", false, "only green star restaurants")

	if err := createCmd.Parse(args); err != nil {
		return err
	}

	if createCmd.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "error: trip name is required")
		printTripUsage()
		return nil
	}

	tripName := createCmd.Arg(0)

	// Parse comma-separated lists
	var cityList, distinctionList, cuisineList []string
	if *cities != "" {
		cityList = strings.Split(*cities, ",")
		for i := range cityList {
			cityList[i] = strings.TrimSpace(cityList[i])
		}
	}
	if *distinctions != "" {
		distinctionList = strings.Split(*distinctions, ",")
		for i := range distinctionList {
			distinctionList[i] = strings.TrimSpace(distinctionList[i])
		}
	}
	if *cuisines != "" {
		cuisineList = strings.Split(*cuisines, ",")
		for i := range cuisineList {
			cuisineList[i] = strings.TrimSpace(cuisineList[i])
		}
	}

	newTrip := &trip.Trip{
		Name:         tripName,
		Cities:       cityList,
		Distinctions: distinctionList,
		MinStars:     *minStars,
		Cuisines:     cuisineList,
		GreenStarOnly: *greenStar,
	}

	manager, err := trip.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create trip manager: %w", err)
	}

	if err := manager.Create(newTrip); err != nil {
		return fmt.Errorf("failed to create trip: %w", err)
	}

	fmt.Printf("✓ Trip %q created successfully\n", tripName)
	return nil
}

// handleTripList lists all trips
func handleTripList(args []string) error {
	manager, err := trip.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create trip manager: %w", err)
	}

	trips, err := manager.List()
	if err != nil {
		return fmt.Errorf("failed to list trips: %w", err)
	}

	if len(trips) == 0 {
		fmt.Println("No trips found. Create one with: mym trip create <name>")
		return nil
	}

	fmt.Printf("Found %d trip(s):\n\n", len(trips))
	for _, t := range trips {
		fmt.Printf("  %s\n", t.Name)
		if len(t.Cities) > 0 {
			fmt.Printf("    Cities: %s\n", strings.Join(t.Cities, ", "))
		}
		if len(t.Distinctions) > 0 {
			fmt.Printf("    Distinctions: %s\n", strings.Join(t.Distinctions, ", "))
		}
		if t.MinStars > 0 {
			fmt.Printf("    Min Stars: %d\n", t.MinStars)
		}
		if len(t.Cuisines) > 0 {
			fmt.Printf("    Cuisines: %s\n", strings.Join(t.Cuisines, ", "))
		}
		if t.GreenStarOnly {
			fmt.Printf("    Green Star Only: yes\n")
		}
		fmt.Printf("    Created: %s\n", t.CreatedAt.Format("2006-01-02"))
		fmt.Println()
	}

	return nil
}

// handleTripDelete deletes a trip
func handleTripDelete(args []string) error {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "error: trip name is required")
		printTripUsage()
		return nil
	}

	tripName := args[0]

	manager, err := trip.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create trip manager: %w", err)
	}

	if err := manager.Delete(tripName); err != nil {
		return fmt.Errorf("failed to delete trip: %w", err)
	}

	fmt.Printf("✓ Trip %q deleted successfully\n", tripName)
	return nil
}

// handleTripExport exports a trip to various formats
func handleTripExport(args []string) error {
	exportCmd := flag.NewFlagSet("export", flag.ExitOnError)
	format := exportCmd.String("format", "all", "export format: geojson, kml, html, or all")
	outputDir := exportCmd.String("output", ".", "output directory")

	if err := exportCmd.Parse(args); err != nil {
		return err
	}

	if exportCmd.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "error: trip name is required")
		printTripUsage()
		return nil
	}

	tripName := exportCmd.Arg(0)

	// Load trip
	manager, err := trip.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create trip manager: %w", err)
	}

	tripData, err := manager.Get(tripName)
	if err != nil {
		return fmt.Errorf("failed to get trip: %w", err)
	}

	// Connect to database
	dbPath := filepath.Join("data", "michelin.db")
	repo, err := storage.NewSQLiteRepository(dbPath)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Query restaurants
	ctx := context.Background()
	restaurants, err := manager.QueryRestaurants(ctx, repo.GetDB(), tripData)
	if err != nil {
		return fmt.Errorf("failed to query restaurants: %w", err)
	}

	if len(restaurants) == 0 {
		fmt.Println("No restaurants match your trip filters.")
		return nil
	}

	fmt.Printf("Found %d restaurant(s) matching your trip filters\n", len(restaurants))

	// Ensure output directory exists
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Export to requested formats
	formats := []string{*format}
	if *format == "all" {
		formats = []string{"geojson", "kml", "html"}
	}

	for _, fmt := range formats {
		var outputPath string
		var exportErr error

		switch fmt {
		case "geojson":
			outputPath = filepath.Join(*outputDir, tripName+".geojson")
			exportErr = trip.ExportGeoJSON(restaurants, outputPath)
		case "kml":
			outputPath = filepath.Join(*outputDir, tripName+".kml")
			exportErr = trip.ExportKML(restaurants, outputPath, tripName)
		case "html":
			outputPath = filepath.Join(*outputDir, tripName+".html")
			exportErr = trip.ExportHTML(restaurants, outputPath, tripName)
		default:
			fmt.Fprintf(os.Stderr, "unknown format: %s\n", fmt)
			continue
		}

		if exportErr != nil {
			fmt.Fprintf(os.Stderr, "failed to export %s: %v\n", fmt, exportErr)
			continue
		}

		fmt.Printf("✓ Exported to %s\n", outputPath)
	}

	return nil
}

// main is the entry point for the mym CLI tool
func main() {
	os.Setenv("TZ", time.UTC.String())
	time.Local = time.UTC

	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
