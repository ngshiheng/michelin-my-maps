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

	fName := "generated/michelin_my_maps.csv"
	file, err := os.Create(fName)
	if err != nil {
		log.Fatalf("cannot create file %q: %s\n", fName, err)
		return
	}

	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	csvHeader := model.GenerateFieldNameSlice(model.Restaurant{})
	writer.Write(csvHeader)

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

	c.OnResponse(func(r *colly.Response) {
		log.Println("visited", r.Request.URL)
		r.Request.Visit(r.Ctx.Get("url"))
	})

	c.OnScraped(func(r *colly.Response) {
		log.Println("finished", r.Request.URL)
	})

	// Extract url of each restaurant and visit them
	c.OnXML("//div[@class='col-md-6 col-lg-6 col-xl-3']", func(e *colly.XMLElement) {
		url := e.Request.AbsoluteURL(e.ChildAttr("//a[@class='link']", "href"))
		location := e.ChildText("//div[@class='card__menu-footer--location flex-fill pl-text']/i/following-sibling::text()")

		e.Request.Ctx.Put("location", location)

		switch requestUrl := e.Request.URL.String(); requestUrl {
		case "https://guide.michelin.com/en/restaurants/3-stars-michelin/":
			e.Request.Ctx.Put("classification", "3 MICHELIN Stars")
		case "https://guide.michelin.com/en/restaurants/2-stars-michelin/":
			e.Request.Ctx.Put("classification", "2 MICHELIN Stars")
		case "https://guide.michelin.com/en/restaurants/1-star-michelin/":
			e.Request.Ctx.Put("classification", "1 MICHELIN Star")
		case "https://guide.michelin.com/en/restaurants/bib-gourmand":
			e.Request.Ctx.Put("classification", "Bib Gourmand")
		}
		detailCollector.Request(e.Request.Method, url, nil, e.Request.Ctx, nil)
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
		latitude, longitude := parser.ExtractCoordinates(googleMapsUrl)

		phoneNumber := e.ChildText("//span[@class='flex-fill']")
		phoneNumber = strings.ReplaceAll(phoneNumber, " ", "")

		websiteUrl := e.ChildAttr("//div[@class='collapse__block-item link-item']/a", "href")

		restaurant := model.Restaurant{
			Name:           name,
			Address:        address,
			Location:       e.Request.Ctx.Get("location"),
			Price:          price,
			Type:           restaurantType,
			Longitude:      longitude,
			Latitude:       latitude,
			PhoneNumber:    phoneNumber,
			Url:            e.Request.URL.String(),
			WebsiteUrl:     websiteUrl,
			Classification: e.Request.Ctx.Get("classification"),
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
