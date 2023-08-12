package main

import (
	"os"

	"github.com/ngshiheng/michelin-my-maps/pkg/crawler"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.WarnLevel)
}

func main() {
	c := crawler.Default()
	c.Crawl()
}
