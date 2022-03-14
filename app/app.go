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
	fName := filepath.Join("generated", "michelin_my_maps.csv")

	file, err := os.Create(fName)
	if err != nil {
		log.Fatalf("cannot create file %q: %s\n", fName, err)
	}

	writer := csv.NewWriter(file)

	csvHeader := model.GenerateFieldNameSlice(model.Restaurant{})

	if err := writer.Write(csvHeader); err != nil {
		log.Fatalf("cannot write header to csv %q: %s\n", fName, err)
	}

	// Initialize colly collectors
	urls := []string{
		"https://guide.michelin.com/en/restaurants/3-stars-michelin/",
		"https://guide.michelin.com/en/restaurants/2-stars-michelin/",
		"https://guide.michelin.com/en/restaurants/1-star-michelin/",
		"https://guide.michelin.com/en/restaurants/bib-gourmand",
	}

	cacheDir := filepath.Join("cache")

	collector := colly.NewCollector(
		colly.CacheDir(cacheDir),
		colly.AllowedDomains("guide.michelin.com", "michelin.com"),
	)

	collector.Limit(&colly.LimitRule{
		Parallelism: 5,
		Delay:       2 * time.Second,
		RandomDelay: 2 * time.Second,
	})

	detailCollector := collector.Clone()

	extensions.RandomUserAgent(collector)
	extensions.Referer(collector)

	return &App{
		collector,
		detailCollector,
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
	app.collector.OnXML("//div[@class='col-md-6 col-lg-6 col-xl-3']", func(e *colly.XMLElement) {
		url := e.Request.AbsoluteURL(e.ChildAttr("//a[@class='link']", "href"))
		location := e.ChildText("//div[@class='card__menu-footer--location flex-fill pl-text']/i/following-sibling::text()")

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
	app.collector.OnXML("//a[@class='btn btn-outline-secondary btn-sm']", func(e *colly.XMLElement) {
		e.Request.Visit(e.Attr("href"))
	})

	// Extract details of the restaurant
	app.detailCollector.OnXML("//div[@class='restaurant-details']", func(e *colly.XMLElement) {
		name := e.ChildText("//h2[@class='restaurant-details__heading--title']")

		address := e.ChildText("//ul[@class='restaurant-details__heading--list']/li")

		priceAndType := e.ChildText("//li[@class='restaurant-details__heading-price']")
		price, restaurantType := parser.SplitUnpack(priceAndType, "•")
		price = parser.TrimWhiteSpaces(price)

		minPrice, maxPrice, currency := parser.ExtractPrice(price)

		googleMapsUrl := e.ChildAttr("//div[@class='google-map__static']/iframe", "src")
		latitude, longitude := parser.ExtractCoordinates(googleMapsUrl)

		phoneNumberString := e.ChildText("//span[@class='flex-fill']")
		phoneNumber, _ := phonenumbers.Parse(phoneNumberString, "")

		websiteUrl := e.ChildAttr("//div[@class='collapse__block-item link-item']/a", "href")

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
