package crawler

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/gocolly/colly/v2/queue"
	"github.com/ngshiheng/michelin-my-maps/v2/pkg/logger"
	"github.com/ngshiheng/michelin-my-maps/v2/pkg/michelin"
	"github.com/ngshiheng/michelin-my-maps/v2/pkg/parser"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	// Colly collector settings
	allowedDomain         = "guide.michelin.com"
	cachePath             = "cache"
	delay                 = 5 * time.Second
	additionalRandomDelay = 5 * time.Second

	// Colly queue settings
	threadCount = 1
	urlCount    = 30_000 // There are currently ~17k restaurants on Michelin Guide as of Jun 2024

	// SQLite database settings
	sqlitePath = "michelin.db"
)

// App contains the necessary components for the crawler.
type App struct {
	collector    *colly.Collector
	database     *gorm.DB
	queue        *queue.Queue
	michelinURLs []michelin.GuideURL
}

// Default creates an App instance with default settings.
func Default() *App {
	a := &App{}
	a.initDefaultURLs()
	a.initDefaultCollector()
	a.initDefaultDatabase()
	a.initDefaultQueue()
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
	a.initDefaultQueue()
	return a
}

// Initialize default start URLs.
func (a *App) initDefaultURLs() {
	allAwards := []string{
		michelin.ThreeStars,
		michelin.TwoStars,
		michelin.OneStar,
		michelin.BibGourmand,
		michelin.SelectedRestaurants,
	}

	for _, distinction := range allAwards {
		url, ok := michelin.DistinctionURL[distinction]
		if !ok {
			continue
		}

		michelinURL := michelin.GuideURL{
			Distinction: distinction,
			URL:         url,
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
		Delay:       delay,
		RandomDelay: additionalRandomDelay,
	})

	extensions.RandomUserAgent(c)
	extensions.Referer(c)

	a.collector = c
}

// Initialize the default database.
func (a *App) initDefaultDatabase() {
	db, err := gorm.Open(sqlite.Open("michelin.db"), &gorm.Config{
		PrepareStmt: true,
	})

	if err != nil {
		log.Fatal("failed to connect to database")
	}

	// Get the generic database object sql.DB to use its functions
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("failed to get database object:", err)
	}

	// Set PRAGMA statements
	_, err = sqlDB.Exec("PRAGMA journal_mode = WAL;")
	if err != nil {
		log.Fatal("failed to set journal_mode:", err)
	}

	_, err = sqlDB.Exec("PRAGMA synchronous = NORMAL;")
	if err != nil {
		log.Fatal("failed to set synchronous:", err)
	}

	_, err = sqlDB.Exec("PRAGMA cache_size = 10000;")
	if err != nil {
		log.Fatal("failed to set cache_size:", err)
	}

	_, err = sqlDB.Exec("PRAGMA temp_store = MEMORY;")
	if err != nil {
		log.Fatal("failed to set temp_store:", err)
	}

	// Automigrate the Restaurant model to ensure tables and indexes are created
	db.AutoMigrate(&michelin.Restaurant{})

	// Assign the database to the App struct
	a.database = db
}

// Initialize the default queue.
func (a *App) initDefaultQueue() {
	q, err := queue.New(
		threadCount,
		&queue.InMemoryQueueStorage{MaxSize: urlCount},
	)
	if err != nil {
		log.Fatal("failed to create queue:", err)
	}
	a.queue = q
}

// Crawl crawls Michelin Guide Restaurants information from a.michelinURLs.
func (a *App) Crawl() {
	defer logger.TimeTrack(time.Now(), "crawl")

	dc := a.collector.Clone()
	extensions.RandomUserAgent(dc)
	extensions.Referer(dc)

	a.collector.OnRequest(func(r *colly.Request) {
		log.WithField("url", r.URL).Debug("visiting")
		a.queue.AddRequest(r)
	})

	a.collector.OnResponse(func(r *colly.Response) {
		log.WithFields(
			log.Fields{
				"status_code": r.StatusCode,
				"url":         r.Request.URL,
			},
		).Info("visited")
		r.Request.Visit(r.Ctx.Get("url"))
	})

	a.collector.OnScraped(func(r *colly.Response) {
		log.WithField("url", r.Request.URL).Debug("finished")
	})

	a.collector.OnError(func(r *colly.Response, err error) {
		log.WithFields(
			log.Fields{
				"error":       err,
				"headers":     r.Request.Headers,
				"status_code": r.StatusCode,
				"url":         r.Request.URL,
			},
		).Error("error")
	})

	dc.OnRequest(func(r *colly.Request) {
		log.Debug("visiting: ", r.URL)
		a.queue.AddRequest(r)
	})

	dc.OnError(func(r *colly.Response, err error) {
		log.WithFields(
			log.Fields{
				"error":       err,
				"headers":     r.Request.Headers,
				"status_code": r.StatusCode,
				"url":         r.Request.URL,
			},
		).Error("error")
	})

	// Extract url of each restaurant from the main page and visit them
	a.collector.OnXML(restaurantXPath, func(e *colly.XMLElement) {
		url := e.Request.AbsoluteURL(e.ChildAttr(restaurantDetailUrlXPath, "href"))

		location := e.ChildText(restaurantLocationXPath)
		longitude := e.ChildAttrs(restaurantXPath, "data-lng")[0]
		latitude := e.ChildAttrs(restaurantXPath, "data-lat")[0]

		e.Request.Ctx.Put("location", location)
		e.Request.Ctx.Put("longitude", longitude)
		e.Request.Ctx.Put("latitude", latitude)

		dc.Request(e.Request.Method, url, nil, e.Request.Ctx, nil)
	})

	// Extract and visit next page links
	a.collector.OnXML(nextPageArrowButtonXPath, func(e *colly.XMLElement) {
		e.Request.Visit(e.Attr("href"))
	})

	// Extract details of each restaurant and write to sqlite database
	dc.OnXML(restaurantDetailXPath, func(e *colly.XMLElement) {
		url := e.Request.URL.String()
		websiteUrl := e.ChildAttr(restaurantWebsiteUrlXPath, "href")

		name := e.ChildText(restaurantNameXPath)

		address := e.ChildText(restaurantAddressXPath)
		address = strings.Replace(address, "\n", " ", -1)

		description := e.ChildText(restaurantDescriptionXPath)

		distinctions := e.ChildTexts(restaurantDistinctionXPath)
		distinction := michelin.SelectedRestaurants
		if len(distinctions) > 0 {
			distinction = parser.ParseDistinction(distinctions[0])
		}

		greenStar := false
		if len(distinctions) > 1 {
			greenStar = parser.ParseGreenStar(distinctions[len(distinctions)-1])
		}

		priceAndCuisine := e.ChildText(restaurantPriceAndCuisineXPath)
		price, cuisine := parser.SplitUnpack(priceAndCuisine, "·")

		phoneNumber := e.ChildAttr(restaurantPhoneNumberXPath, "href")
		formattedPhoneNumber := parser.ParsePhoneNumber(phoneNumber)
		if formattedPhoneNumber == "" {
			log.WithFields(
				log.Fields{
					"url":          url,
					"phone_number": phoneNumber,
				},
			).Warn("invalid phone number")
		}

		facilitiesAndServices := e.ChildTexts(restaurantFacilitiesAndServicesXPath)

		restaurant := michelin.Restaurant{
			Address:               address,
			Cuisine:               cuisine,
			Description:           parser.TrimWhiteSpaces(description),
			Distinction:           distinction,
			FacilitiesAndServices: strings.Join(facilitiesAndServices, ","),
			GreenStar:             greenStar,
			Location:              e.Request.Ctx.Get("location"),
			Latitude:              e.Request.Ctx.Get("latitude"),
			Longitude:             e.Request.Ctx.Get("longitude"),
			Name:                  name,
			PhoneNumber:           formattedPhoneNumber,
			Price:                 price,
			URL:                   url,
			WebsiteURL:            websiteUrl,
			UpdatedOn:             time.Now(),
		}

		log.Debug(restaurant)
		a.database.Create(&restaurant)
	})

	for _, url := range a.michelinURLs {
		a.queue.AddURL(url.URL)
	}

	// Start scraping
	a.queue.Run(a.collector)
}
