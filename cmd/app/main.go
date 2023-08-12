package main

import (
	"flag"
	"os"

	"github.com/ngshiheng/michelin-my-maps/pkg/crawler"
	log "github.com/sirupsen/logrus"
)

func main() {
	var (
		logLevel = flag.String("log", log.InfoLevel.String(), "log level (debug, info, warning, error, fatal, panic)")
		helpFlag = flag.Bool("help", false, "show help message")
	)
	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Errorf("failed to parse log level: %v\n", err)
		os.Exit(1)
	}
	log.SetLevel(level)
	log.SetOutput(os.Stdout)

	app := crawler.Default()
	app.Crawl()
}
