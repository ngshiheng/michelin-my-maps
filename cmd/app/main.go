package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ngshiheng/michelin-my-maps/pkg/crawler"
	"github.com/sirupsen/logrus"
)

func main() {
	var (
		logLevel = flag.String("log", logrus.WarnLevel.String(), "Log level (debug, info, warning, error, fatal, panic).")
		helpFlag = flag.Bool("help", false, "Show help message.")
	)
	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	level, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		fmt.Printf("Failed to parse log level: %v\n", err)
		os.Exit(1)
	}
	logrus.SetLevel(level)
	logrus.SetOutput(os.Stdout)

	app := crawler.Default()
	app.Crawl()

}
