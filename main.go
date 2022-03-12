package main

import (
	"log"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
	"github.com/ngshiheng/michelin-my-maps/model"
	"github.com/ngshiheng/michelin-my-maps/util/logger"
	"github.com/ngshiheng/michelin-my-maps/util/parser"
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

		address := e.ChildText("//ul[@class='restaurant-details__heading--list']/li")

		priceAndType := e.ChildText("//li[@class='restaurant-details__heading-price']")
		price, restaurantType := parser.SplitUnpack(priceAndType, "•")
		price = parser.TrimWhiteSpaces(price)

		phoneNumber := e.ChildText("//span[@class='flex-fill']")
		phoneNumber = strings.ReplaceAll(phoneNumber, " ", "")

		classification := e.ChildText("//ul[@class='restaurant-details__classification--list']/li")
		classification = parser.TrimWhiteSpaces(classification)

		michelinUrl := e.ChildAttr("//div[@class='collapse__block-item link-item']/a", "href")
		michelinUrl = parser.TrimWhiteSpaces(michelinUrl)

		restaurant := model.Restaurant{
			Name:           name,
			Address:        address,
			Price:          price,
			Type:           restaurantType,
			Latitude:       0.00,
			Longitude:      0.00,
			PhoneNumber:    phoneNumber,
			MichelinUrl:    michelinUrl,
			WebsiteUrl:     e.Request.URL.String(),
			Classification: classification,
		}

		log.Println(restaurant)

	})

	// Start scraping
	c.Visit("https://guide.michelin.com/en/restaurants")

	// Wait until threads are finished
	c.Wait()
	detailCollector.Wait()
}
