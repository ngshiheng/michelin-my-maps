package app

import (
	"encoding/csv"
	"net/http"

	"os"
	"path/filepath"
	"time"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
	"github.com/ngshiheng/michelin-my-maps/model"
	"github.com/ngshiheng/michelin-my-maps/util/logger"
	"github.com/ngshiheng/michelin-my-maps/util/parser"

	log "github.com/sirupsen/logrus"
)

type App struct {
	collector       *colly.Collector
	detailCollector *colly.Collector
	writer          *csv.Writer
	file            *os.File
	startUrls       []startUrl
}

func New() *App {
	// Initialize csv file and writer
	file, err := os.Create(filepath.Join(outputPath, outputFileName))
	if err != nil {
		log.WithFields(log.Fields{"file": file}).Fatal("cannot create file")
	}

	writer := csv.NewWriter(file)

	csvHeader := model.GenerateFieldNameSlice(model.Restaurant{})
	if err := writer.Write(csvHeader); err != nil {
		log.WithFields(log.Fields{
			"file":      file,
			"csvHeader": csvHeader,
		}).Fatal("cannot write header to file")
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

	extensions.RandomUserAgent(c)
	extensions.Referer(c)

	dc := c.Clone()

	return &App{
		c,
		dc,
		writer,
		file,
		urls,
	}
}

// Crawl Michelin Guide Restaurants information from app.startUrls
func (app *App) Crawl() {
	defer logger.TimeTrack(time.Now(), "crawl")
	defer app.file.Close()
	defer app.writer.Flush()

	app.collector.OnResponse(func(r *colly.Response) {
		log.Info("visited ", r.Request.URL)
		r.Request.Visit(r.Ctx.Get("url"))
	})

	app.collector.OnScraped(func(r *colly.Response) {
		log.Info("finished ", r.Request.URL)
	})

	// Extract url of each restaurant from the main page and visit them
	app.collector.OnXML(restaurantXPath, func(e *colly.XMLElement) {
		url := e.Request.AbsoluteURL(e.ChildAttr(restaurantDetailUrlXPath, "href"))

		location := e.ChildText(restaurantLocationXPath)
		longitude := e.ChildAttr(restaurantXPath, "data-lng")
		latitude := e.ChildAttr(restaurantXPath, "data-lat")

		e.Request.Ctx.Put("location", location)
		e.Request.Ctx.Put("longitude", longitude)
		e.Request.Ctx.Put("latitude", latitude)

		app.detailCollector.Request(e.Request.Method, url, nil, e.Request.Ctx, nil)
	})

	// Extract and visit next page links
	app.collector.OnXML(nextPageArrowButtonXPath, func(e *colly.XMLElement) {
		e.Request.Visit(e.Attr("href"))
	})

	// Extract details of each restaurant and write to csv file
	app.detailCollector.OnXML(restaurantDetailXPath, func(e *colly.XMLElement) {
		url := e.Request.URL.String()
		websiteUrl := e.ChildAttr(restaurantWebsiteUrlXPath, "href")

		name := e.ChildText(restaurantNameXPath)

		address := e.ChildText(restaurantAddressXPath)

		priceAndCuisine := e.ChildText(restaurantPriceAndCuisineXPath)
		price, cuisine := parser.SplitUnpack(priceAndCuisine, "•")
		price = parser.TrimWhiteSpaces(price)

		minPrice, maxPrice, currency := parser.ParsePrice(price)

		phoneNumber := e.ChildAttr(restaurantPhoneNumberXPath, "href")
		formattedPhoneNumber := parser.ParsePhoneNumber(phoneNumber)

		restaurant := model.Restaurant{
			Name:        name,
			Address:     address,
			Location:    e.Request.Ctx.Get("location"),
			MinPrice:    minPrice,
			MaxPrice:    maxPrice,
			Currency:    currency,
			Cuisine:     cuisine,
			Longitude:   e.Request.Ctx.Get("longitude"),
			Latitude:    e.Request.Ctx.Get("latitude"),
			PhoneNumber: formattedPhoneNumber,
			Url:         url,
			WebsiteUrl:  websiteUrl,
			Award:       e.Request.Ctx.Get("award"),
		}

		log.Debug(restaurant)

		if err := app.writer.Write(model.GenerateFieldValueSlice(restaurant)); err != nil {
			log.Fatalf("cannot write data %q: %s\n", restaurant, err)
		}
	})

	// Start scraping
	for _, url := range app.startUrls {
		ctx := colly.NewContext()
		ctx.Put("award", url.Award)
		app.collector.Request(http.MethodGet, url.Url, nil, ctx, nil)
	}

	// Wait until threads are finished
	app.collector.Wait()
	app.detailCollector.Wait()
}
