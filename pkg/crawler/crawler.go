package crawler

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/ngshiheng/michelin-my-maps/v2/pkg/logger"
	"github.com/ngshiheng/michelin-my-maps/v2/pkg/michelin"
	"github.com/ngshiheng/michelin-my-maps/v2/pkg/parser"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	allowedDomain = "guide.michelin.com"
	cachePath     = "cache"
	delay         = 2 * time.Second
	parallelism   = 5
	randomDelay   = 2 * time.Second
	sqlitePath    = "michelin_my_maps.db"
)

// App contains the necessary components for the crawler.
type App struct {
	collector    *colly.Collector
	database     *gorm.DB
	michelinURLs []michelin.GuideURL
}

// Default creates an App instance with default settings.
func Default() *App {
	a := &App{}
	a.initDefaultURLs()
	a.initDefaultCollector()
	a.initDefaultDatabase()
	return a
}

// New creates an App instance with custom distinction and database.
func New(distinction string, db *gorm.DB) *App {
	url := michelin.GuideURL{
		Distinction: distinction,
		URL:         michelin.DistinctionURL[distinction],
	}

	a := &App{
		database:     db,
		michelinURLs: []michelin.GuideURL{url},
	}
	a.initDefaultCollector()
	return a
}

// Initialize default start URLs.
func (a *App) initDefaultURLs() {
	allAwards := []string{michelin.ThreeStars, michelin.TwoStars, michelin.OneStar, michelin.BibGourmand, michelin.GreenStar}

	for _, distinction := range allAwards {
		michelinURL := michelin.GuideURL{
			Distinction: distinction,
			URL:         michelin.DistinctionURL[distinction],
		}
		a.michelinURLs = append(a.michelinURLs, michelinURL)
	}
}

// Initialize the default collector.
func (a *App) initDefaultCollector() {
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

	a.collector = c
}

// Initialize the default database.
func (a *App) initDefaultDatabase() {
	db, err := gorm.Open(sqlite.Open(sqlitePath), &gorm.Config{})
	if err != nil {
		panic("failed to connect to database")
	}
	db.AutoMigrate(&michelin.Restaurant{})
	a.database = db
}

// Crawl crawls Michelin Guide Restaurants information from a.michelinURLs.
func (a *App) Crawl() {
	defer logger.TimeTrack(time.Now(), "crawl")

	dc := a.collector.Clone()

	a.collector.OnResponse(func(r *colly.Response) {
		log.Info("visited: ", r.Request.URL)
		r.Request.Visit(r.Ctx.Get("url"))
	})

	a.collector.OnScraped(func(r *colly.Response) {
		log.Debug("finished: ", r.Request.URL)
	})

	// Extract url of each restaurant from the main page and visit them
	a.collector.OnXML(restaurantXPath, func(e *colly.XMLElement) {
		url := e.Request.AbsoluteURL(e.ChildAttr(restaurantDetailUrlXPath, "href"))

		location := e.ChildText(restaurantLocationXPath)
		longitude := e.ChildAttr(restaurantXPath, "data-lng")
		latitude := e.ChildAttr(restaurantXPath, "data-lat")

		e.Request.Ctx.Put("location", location)
		e.Request.Ctx.Put("longitude", longitude)
		e.Request.Ctx.Put("latitude", latitude)

		dc.Request(e.Request.Method, url, nil, e.Request.Ctx, nil)
	})

	// Extract and visit next page links
	a.collector.OnXML(nextPageArrowButtonXPath, func(e *colly.XMLElement) {
		e.Request.Visit(e.Attr("href"))
	})

	// Extract details of each restaurant and write to csv file
	dc.OnXML(restaurantDetailXPath, func(e *colly.XMLElement) {
		url := e.Request.URL.String()
		websiteUrl := e.ChildAttr(restaurantWebsiteUrlXPath, "href")

		name := e.ChildText(restaurantNameXPath)

		address := e.ChildText(restaurantAddressXPath)
		address = strings.Replace(address, "\n", " ", -1)

		description := e.ChildText(restaurantDescriptionXPath)

		distinctions := []string{}
		distinctionSlice := e.ChildTexts(restaurantDistinctionXPath)
		for _, d := range distinctionSlice {
			distinction := parser.ParseDistinction(d)
			if distinction != "" {
				distinctions = append(distinctions, distinction)
			}
		}

		priceAndCuisine := e.ChildText(restaurantPriceAndCuisineXPath)
		price, cuisine := parser.SplitUnpack(priceAndCuisine, "·")

		phoneNumber := e.ChildAttr(restaurantPhoneNumberXPath, "href")
		formattedPhoneNumber := parser.ParsePhoneNumber(phoneNumber)
		if formattedPhoneNumber == "" {
			log.WithFields(
				log.Fields{
					"url":                  url,
					"phoneNumber":          phoneNumber,
					"formattedPhoneNumber": formattedPhoneNumber,
				},
			).Warn("phone number is not available")
		}

		facilitiesAndServicesSlice := e.ChildTexts(restaurantFacilitiesAndServicesXPath)

		restaurant := michelin.Restaurant{
			Address:               address,
			Cuisine:               cuisine,
			Description:           parser.TrimWhiteSpaces(description),
			Distinction:           strings.Join(distinctions, ","),
			FacilitiesAndServices: strings.Join(facilitiesAndServicesSlice, ","),
			Latitude:              e.Request.Ctx.Get("latitude"),
			Location:              e.Request.Ctx.Get("location"),
			Longitude:             e.Request.Ctx.Get("longitude"),
			Name:                  name,
			PhoneNumber:           formattedPhoneNumber,
			Price:                 price,
			URL:                   url,
			WebsiteURL:            websiteUrl,
		}

		log.Debug(restaurant)
		a.database.Create(restaurant)
	})

	// Start scraping
	for _, url := range a.michelinURLs {
		ctx := colly.NewContext()
		a.collector.Request(http.MethodGet, url.URL, nil, ctx, nil)
	}

	a.collector.Wait()
	dc.Wait()
}
