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

// printVersion prints the application version
func printVersion() {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("Unable to determine version information.")
		return
	}

	if buildInfo.Main.Version != "" {
		fmt.Printf("Version: %s\n", buildInfo.Main.Version)
	} else {
		fmt.Println("Version: unknown")
	}
}

var (
	logLevel    = flag.String("log", log.InfoLevel.String(), "log level (debug, info, warning, error, fatal, panic)")
	helpFlag    = flag.Bool("help", false, "show help message")
	versionFlag = flag.Bool("version", false, "print version information")
)

func main() {
	flag.Parse()

	if *versionFlag {
		printVersion()
		return
	}

	if *helpFlag {
		flag.Usage()
		return
	}

	// Set log level
	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Errorf("failed to parse log level: %v\n", err)
		os.Exit(1)
	}
	log.SetLevel(level)
	log.SetOutput(os.Stdout)

	// Start crawling process
	app, err := scraper.Default()
	if err != nil {
		log.Fatalf("failed to create scraper: %v", err)
	}

	ctx := context.Background()
	if err := app.Crawl(ctx); err != nil {
		log.Fatalf("failed to crawl: %v", err)
	}
}
