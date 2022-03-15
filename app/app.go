package app

import (
	"encoding/csv"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
	"github.com/ngshiheng/michelin-my-maps/model"
	"github.com/ngshiheng/michelin-my-maps/util/logger"
	"github.com/ngshiheng/michelin-my-maps/util/parser"
	"github.com/nyaruka/phonenumbers"
)

type App struct {
	collector       *colly.Collector
	detailCollector *colly.Collector
	writer          *csv.Writer
	file            *os.File
	URLs            []string
}

func New() *App {
	// Initialize csv file and writer
	file, err := os.Create(filepath.Join(outputPath, outputFileName))
	if err != nil {
		log.Fatalf("cannot create file: %s\n", err)
	}

	writer := csv.NewWriter(file)

	csvHeader := model.GenerateFieldNameSlice(model.Restaurant{})
	if err := writer.Write(csvHeader); err != nil {
		log.Fatalf("cannot write header to file: %s\n", err)
	}

	// Initialize colly collectors
	cacheDir := filepath.Join(cachePath)

	c := colly.NewCollector(
		colly.CacheDir(cacheDir),
		colly.AllowedDomains(allowedDomain),
	)

	c.Limit(&colly.LimitRule{
		Parallelism: parallelism,
		Delay:       delay,
		RandomDelay: randomDelay,
	})

	dc := c.Clone()

	extensions.RandomUserAgent(c)
	extensions.Referer(c)

	return &App{
		c,
		dc,
		writer,
		file,
		urls,
	}
}

// Crawl Michelin Guide Restaurants information from app.URLs
func (app *App) Crawl() {
	defer logger.TimeTrack(time.Now(), "crawl")
	defer app.file.Close()
	defer app.writer.Flush()

	app.collector.OnResponse(func(r *colly.Response) {
		log.Println("visited", r.Request.URL)
		r.Request.Visit(r.Ctx.Get("url"))
	})

	app.collector.OnScraped(func(r *colly.Response) {
		log.Println("finished", r.Request.URL)
	})

	// Extract url of each restaurant and visit them
	app.collector.OnXML(restaurantXPath, func(e *colly.XMLElement) {
		url := e.Request.AbsoluteURL(e.ChildAttr(restaurantDetailUrlXPath, "href"))
		location := e.ChildText(restaurantLocationXPath)

		e.Request.Ctx.Put("location", location)

		switch requestUrl := e.Request.URL.String(); requestUrl {
		case "https://guide.michelin.com/en/restaurants/3-stars-michelin/":
			e.Request.Ctx.Put("award", "3 MICHELIN Stars")
		case "https://guide.michelin.com/en/restaurants/2-stars-michelin/":
			e.Request.Ctx.Put("award", "2 MICHELIN Stars")
		case "https://guide.michelin.com/en/restaurants/1-star-michelin/":
			e.Request.Ctx.Put("award", "1 MICHELIN Star")
		case "https://guide.michelin.com/en/restaurants/bib-gourmand":
			e.Request.Ctx.Put("award", "Bib Gourmand")
		}
		app.detailCollector.Request(e.Request.Method, url, nil, e.Request.Ctx, nil)
	})

	// Extract and visit next page links
	app.collector.OnXML(nextPageArrowButtonXPath, func(e *colly.XMLElement) {
		e.Request.Visit(e.Attr("href"))
	})

	// Extract details of the restaurant
	app.detailCollector.OnXML(restaurantDetailXPath, func(e *colly.XMLElement) {
		name := e.ChildText(restaurantNameXPath)

		address := e.ChildText(restaurantAddressXPath)

		priceAndCuisine := e.ChildText(restaurantpriceAndCuisineXPath)
		price, restaurantType := parser.SplitUnpack(priceAndCuisine, "•")
		price = parser.TrimWhiteSpaces(price)

		minPrice, maxPrice, currency := parser.ExtractPrice(price)

		googleMapsUrl := e.ChildAttr(restarauntGoogleMapsXPath, "src")
		latitude, longitude := parser.ExtractCoordinates(googleMapsUrl)

		phoneNumberString := e.ChildText(restarauntPhoneNumberXPath)
		phoneNumber, _ := phonenumbers.Parse(phoneNumberString, "")

		websiteUrl := e.ChildAttr(restarauntWebsiteUrlXPath, "href")

		restaurant := model.Restaurant{
			Name:        name,
			Address:     address,
			Location:    e.Request.Ctx.Get("location"),
			MinPrice:    minPrice,
			MaxPrice:    maxPrice,
			Currency:    currency,
			Cuisine:     restaurantType,
			Longitude:   longitude,
			Latitude:    latitude,
			PhoneNumber: phonenumbers.Format(phoneNumber, phonenumbers.E164),
			Url:         e.Request.URL.String(),
			WebsiteUrl:  websiteUrl,
			Award:       e.Request.Ctx.Get("award"),
		}

		log.Println(restaurant)

		if err := app.writer.Write(model.GenerateFieldValueSlice(restaurant)); err != nil {
			log.Fatalf("cannot write data %q: %s\n", restaurant, err)
		}
	})

	// Start scraping
	for _, url := range app.URLs {
		app.collector.Visit(url)
	}

	// Wait until threads are finished
	app.collector.Wait()
	app.detailCollector.Wait()
}
