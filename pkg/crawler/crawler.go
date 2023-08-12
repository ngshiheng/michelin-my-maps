package crawler

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/ngshiheng/michelin-my-maps/pkg/logger"
	"github.com/ngshiheng/michelin-my-maps/pkg/michelin"
	"github.com/ngshiheng/michelin-my-maps/pkg/parser"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type App struct {
	collector    *colly.Collector
	database     *gorm.DB
	michelinURLs []michelin.GuideURL
}

func Default() *App {
	a := &App{}
	a.initDefaultStartUrls()
	a.initDefaultCollector()
	a.initDefaultDatabase()
	return a
}

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

func (a *App) initDefaultStartUrls() {
	allAwards := []string{michelin.ThreeStars, michelin.TwoStars, michelin.OneStar, michelin.BibGourmand, michelin.GreenStar}

	for _, distinction := range allAwards {
		michelinURL := michelin.GuideURL{
			Distinction: distinction,
			URL:         michelin.DistinctionURL[distinction],
		}
		a.michelinURLs = append(a.michelinURLs, michelinURL)
	}
}

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

func (a *App) initDefaultDatabase() {
	db, err := gorm.Open(sqlite.Open("michelin.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to database")
	}
	db.AutoMigrate(&michelin.Restaurant{})
	a.database = db
}

// Crawl crawls Michelin Guide Restaurants information from a.startUrls
func (a *App) Crawl() {
	defer logger.TimeTrack(time.Now(), "crawl")

	dc := a.collector.Clone()

	a.collector.OnResponse(func(r *colly.Response) {
		log.Info("visited ", r.Request.URL)
		r.Request.Visit(r.Ctx.Get("url"))
	})

	a.collector.OnScraped(func(r *colly.Response) {
		log.Info("finished ", r.Request.URL)
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
		facilitiesAndServices := strings.Join(facilitiesAndServicesSlice, ",")

		restaurant := michelin.Restaurant{
			Name:                  name,
			Address:               address,
			Location:              e.Request.Ctx.Get("location"),
			Price:                 price,
			Cuisine:               cuisine,
			Longitude:             e.Request.Ctx.Get("longitude"),
			Latitude:              e.Request.Ctx.Get("latitude"),
			PhoneNumber:           formattedPhoneNumber,
			Url:                   url,
			WebsiteUrl:            websiteUrl,
			Award:                 e.Request.Ctx.Get("distinction"),
			FacilitiesAndServices: facilitiesAndServices,
		}

		log.Debug(restaurant)
		a.database.Create(restaurant)
	})

	// Start scraping
	for _, url := range a.michelinURLs {
		ctx := colly.NewContext()
		ctx.Put("distinction", url.Distinction)
		a.collector.Request(http.MethodGet, url.URL, nil, ctx, nil)
	}

	// Wait until threads are finished
	a.collector.Wait()
	dc.Wait()
}
