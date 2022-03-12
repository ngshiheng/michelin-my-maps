package main

import (
	"log"
	"time"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
	"github.com/ngshiheng/michelin-my-maps/util/logger"
)

func main() {
	defer logger.TimeTrack(time.Now(), "main")

	c := colly.NewCollector(
		colly.CacheDir("./cache"),
		colly.AllowedDomains("guide.michelin.com", "michelin.com"),
	)

	c.Limit(&colly.LimitRule{
		Parallelism: 5,
	})

	detailCollector := c.Clone()

	extensions.RandomUserAgent(c)
	extensions.Referer(c)

	c.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	c.OnResponse(func(r *colly.Response) {
		log.Println("visited", r.Request.URL)
	})

	c.OnScraped(func(r *colly.Response) {
		log.Println("finished", r.Request.URL)
	})

	// Extract url of each restaurant
	c.OnXML("//a[@class='link']", func(e *colly.XMLElement) {
		restaurantUrl := e.Request.AbsoluteURL(e.Attr("href"))
		detailCollector.Visit(restaurantUrl)
	})

	// Extract details of the restaurant
	detailCollector.OnXML("//div[@class='restaurant-details']", func(e *colly.XMLElement) {
		name := e.ChildText("//h2[@class='restaurant-details__heading--title']")
		log.Println(name)
	})

	// Start scraping
	c.Visit("https://guide.michelin.com/en/restaurants")

	// Wait until threads are finished
	c.Wait()
	detailCollector.Wait()
}
