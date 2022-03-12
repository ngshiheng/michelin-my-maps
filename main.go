package main

import (
	"encoding/csv"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
	"github.com/ngshiheng/michelin-my-maps/model"
	"github.com/ngshiheng/michelin-my-maps/util/logger"
	"github.com/ngshiheng/michelin-my-maps/util/parser"
)

var urls = []string{
	"https://guide.michelin.com/en/restaurants/3-stars-michelin/",
	"https://guide.michelin.com/en/restaurants/2-stars-michelin/",
	"https://guide.michelin.com/en/restaurants/1-star-michelin/",
	"https://guide.michelin.com/en/restaurants/bib-gourmand",
}

func main() {
	defer logger.TimeTrack(time.Now(), "main")
	crawl()
}

func crawl() {
	defer logger.TimeTrack(time.Now(), "crawl")

	fName := "michelin-my-maps.csv"
	file, err := os.Create(fName)
	if err != nil {
		log.Fatalf("Cannot create file %q: %s\n", fName, err)
		return
	}

	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	csvHeader := model.GenerateFieldNameSlice(model.Restaurant{})
	writer.Write(csvHeader)

	c := colly.NewCollector(
		colly.Async(true),
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

	// Extract url of each restaurant and visit them
	c.OnXML("//a[@class='link']", func(e *colly.XMLElement) {
		restaurantUrl := e.Request.AbsoluteURL(e.Attr("href"))
		detailCollector.Visit(restaurantUrl)
	})

	// Extract and visit next page links
	c.OnXML("//a[@class='btn btn-outline-secondary btn-sm']", func(e *colly.XMLElement) {
		e.Request.Visit(e.Attr("href"))
	})

	// Extract details of the restaurant
	detailCollector.OnXML("//div[@class='restaurant-details']", func(e *colly.XMLElement) {
		name := e.ChildText("//h2[@class='restaurant-details__heading--title']")

		address := e.ChildText("//ul[@class='restaurant-details__heading--list']/li")

		priceAndType := e.ChildText("//li[@class='restaurant-details__heading-price']")
		price, restaurantType := parser.SplitUnpack(priceAndType, "•")
		price = parser.TrimWhiteSpaces(price)

		googleMapsUrl := e.ChildAttr("//div[@class='google-map__static']/iframe", "src")
		longitude, latitude := parser.ExtractCoordinates(googleMapsUrl)

		phoneNumber := e.ChildText("//span[@class='flex-fill']")
		phoneNumber = strings.ReplaceAll(phoneNumber, " ", "")

		websiteUrl := e.ChildAttr("//div[@class='collapse__block-item link-item']/a", "href")

		classification := e.ChildText("//ul[@class='restaurant-details__classification--list']/li")
		classification = parser.TrimWhiteSpaces(classification)

		restaurant := model.Restaurant{
			Name:           name,
			Address:        address,
			Price:          price,
			Type:           restaurantType,
			Longitude:      longitude,
			Latitude:       latitude,
			PhoneNumber:    phoneNumber,
			Url:            e.Request.URL.String(),
			WebsiteUrl:     websiteUrl,
			Classification: classification,
		}

		log.Println(restaurant)
		writer.Write(model.GenerateFieldValueSlice(restaurant))
	})

	// Start scraping
	for _, url := range urls {
		c.Visit(url)
	}

	// Wait until threads are finished
	c.Wait()
	detailCollector.Wait()
}
