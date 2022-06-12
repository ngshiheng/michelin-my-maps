package main

import (
	"os"

	"github.com/ngshiheng/michelin-my-maps/app"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	log.SetOutput(os.Stdout)
	log.SetLevel(log.WarnLevel)
}

func main() {
	crawler := app.New()
	crawler.Crawl()
}
