package main

import (
	"github.com/ngshiheng/michelin-my-maps/app"
)

func main() {
	crawler := app.New()
	crawler.Crawl()
}
