package crawler

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path"
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
	maxRetry              = 3

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
	maxRetry     int
	michelinURLs []michelin.GuideURL
}

// Default creates an App instance with default settings.
func Default() *App {
	a := &App{maxRetry: maxRetry}
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
		maxRetry:     maxRetry,
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

	// Automigrate the Restaurant and RestaurantAward models to ensure tables and indexes are created
	db.AutoMigrate(&michelin.Restaurant{}, &michelin.RestaurantAward{})

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

// clearCache removes the cache file for a given colly.Request.
// by default Colly cache responses that are not 200 OK, including those with error status codes.
func (a *App) clearCache(r *colly.Request) {
	url := r.URL.String()
	sum := sha1.Sum([]byte(url))
	hash := hex.EncodeToString(sum[:])

	cacheDir := path.Join(cachePath, hash[:2])
	filename := path.Join(cacheDir, hash)

	if err := os.Remove(filename); err != nil {
		log.WithFields(
			log.Fields{
				"error":    err,
				"cacheDir": cacheDir,
				"filename": filename,
			},
		).Fatal("failed to remove cache file")
	}
}

// upsertRestaurantAward creates or updates a restaurant and its award for the current year.
func (a *App) upsertRestaurantAward(restaurantData RestaurantData) error {
	currentYear := time.Now().Year()

	// Find or create restaurant
	var restaurant michelin.Restaurant
	result := a.database.Where("url = ?", restaurantData.URL).First(&restaurant)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Create new restaurant
			restaurant = michelin.Restaurant{
				URL:                   restaurantData.URL,
				Name:                  restaurantData.Name,
				Description:           restaurantData.Description,
				Address:               restaurantData.Address,
				Location:              restaurantData.Location,
				Latitude:              restaurantData.Latitude,
				Longitude:             restaurantData.Longitude,
				Cuisine:               restaurantData.Cuisine,
				FacilitiesAndServices: restaurantData.FacilitiesAndServices,
				PhoneNumber:           restaurantData.PhoneNumber,
				WebsiteURL:            restaurantData.WebsiteURL,
			}
			if err := a.database.Create(&restaurant).Error; err != nil {
				return err
			}
		} else {
			return result.Error
		}
	} else {
		// Update existing restaurant's basic info
		restaurant.Name = restaurantData.Name
		restaurant.Description = restaurantData.Description
		restaurant.Address = restaurantData.Address
		restaurant.Location = restaurantData.Location
		restaurant.Latitude = restaurantData.Latitude
		restaurant.Longitude = restaurantData.Longitude
		restaurant.Cuisine = restaurantData.Cuisine
		restaurant.FacilitiesAndServices = restaurantData.FacilitiesAndServices
		restaurant.PhoneNumber = restaurantData.PhoneNumber
		restaurant.WebsiteURL = restaurantData.WebsiteURL
		if err := a.database.Save(&restaurant).Error; err != nil {
			return err
		}
	}

	// Find or create award for current year
	var award michelin.RestaurantAward
	result = a.database.Where("restaurant_id = ? AND year = ?", restaurant.ID, currentYear).First(&award)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Create new award
			award = michelin.RestaurantAward{
				RestaurantID: restaurant.ID,
				Year:         currentYear,
				Distinction:  restaurantData.Distinction,
				Price:        restaurantData.Price,
				GreenStar:    restaurantData.GreenStar,
			}
			if err := a.database.Create(&award).Error; err != nil {
				return err
			}
		} else {
			return result.Error
		}
	} else {
		// Update existing award for current year
		award.Distinction = restaurantData.Distinction
		award.Price = restaurantData.Price
		award.GreenStar = restaurantData.GreenStar
		if err := a.database.Save(&award).Error; err != nil {
			return err
		}
	}

	return nil
}

// RestaurantData holds the scraped restaurant information.
type RestaurantData struct {
	URL                   string
	Name                  string
	Address               string
	Location              string
	Latitude              string
	Longitude             string
	Cuisine               string
	PhoneNumber           string
	WebsiteURL            string
	Distinction           string
	Description           string
	Price                 string
	FacilitiesAndServices string
	GreenStar             bool
}

// Crawl crawls Michelin Guide Restaurants information from a.michelinURLs.
func (a *App) Crawl() {
	defer logger.TimeTrack(time.Now(), "crawl")

	dc := a.collector.Clone()
	extensions.RandomUserAgent(dc)
	extensions.Referer(dc)

	a.collector.OnRequest(func(r *colly.Request) {
		attempt := r.Ctx.GetAny("attempt")
		if attempt == nil {
			r.Ctx.Put("attempt", 1)
		}

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
		attempt := r.Ctx.GetAny("attempt").(int)

		shouldRetry := r.StatusCode >= 300 && attempt <= a.maxRetry
		if shouldRetry {
			a.clearCache(r.Request)
			log.WithFields(
				log.Fields{
					"attempt":     attempt,
					"error":       err,
					"status_code": r.StatusCode,
					"url":         r.Request.URL,
				},
			).Warnf("retrying request in %v", delay)
			r.Ctx.Put("attempt", attempt+1)
			time.Sleep(delay)
			r.Request.Retry()
		} else {
			log.WithFields(
				log.Fields{
					"error":       err,
					"headers":     r.Request.Headers,
					"status_code": r.StatusCode,
					"url":         r.Request.URL,
				},
			).Error("error")
		}
	})

	dc.OnRequest(func(r *colly.Request) {
		attempt := r.Ctx.GetAny("attempt")
		if attempt == nil {
			r.Ctx.Put("attempt", 1)
		}

		log.WithField("url", r.URL).Debug("visiting")
		a.queue.AddRequest(r)
	})

	dc.OnError(func(r *colly.Response, err error) {
		attempt := r.Ctx.GetAny("attempt").(int)

		shouldRetry := r.StatusCode >= 300 && attempt <= a.maxRetry
		if shouldRetry {
			a.clearCache(r.Request)
			log.WithFields(
				log.Fields{
					"attempt":     attempt,
					"error":       err,
					"status_code": r.StatusCode,
					"url":         r.Request.URL,
				},
			).Warnf("retrying request in %v", delay)
			r.Ctx.Put("attempt", attempt+1)
			time.Sleep(delay)
			r.Request.Retry()
		} else {
			log.WithFields(
				log.Fields{
					"error":       err,
					"headers":     r.Request.Headers,
					"status_code": r.StatusCode,
					"url":         r.Request.URL,
				},
			).Error("error")
		}
	})

	// Extract url of each restaurant from the main page and visit them
	a.collector.OnXML(restaurantXPath, func(e *colly.XMLElement) {
		url := e.Request.AbsoluteURL(e.ChildAttr(restaurantDetailUrlXPath, "href"))

		location := e.ChildText(restaurantLocationXPath)
		longitude := e.Attr("data-lng")
		latitude := e.Attr("data-lat")

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
		distinction := e.ChildText(restaurantDistinctionXPath)

		greenStar := e.ChildText(restaurantGreenStarXPath)

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
			).Debug("invalid phone number")
		}

		facilitiesAndServices := e.ChildTexts(restaurantFacilitiesAndServicesXPath)

		fmt.Println(description)
		fmt.Println(parser.TrimWhiteSpaces(description))

		restaurantData := RestaurantData{
			URL:                   url,
			Name:                  name,
			Address:               address,
			Location:              e.Request.Ctx.Get("location"),
			Latitude:              e.Request.Ctx.Get("latitude"),
			Longitude:             e.Request.Ctx.Get("longitude"),
			Cuisine:               cuisine,
			PhoneNumber:           formattedPhoneNumber,
			WebsiteURL:            websiteUrl,
			Distinction:           parser.ParseDistinction(distinction),
			Description:           parser.TrimWhiteSpaces(description),
			Price:                 price,
			FacilitiesAndServices: strings.Join(facilitiesAndServices, ","),
			GreenStar:             parser.ParseGreenStar(greenStar),
		}

		log.Debug(restaurantData)
		if err := a.upsertRestaurantAward(restaurantData); err != nil {
			log.WithFields(
				log.Fields{
					"url":   url,
					"error": err,
				},
			).Error("failed to upsert restaurant award")
		}
	})

	for _, url := range a.michelinURLs {
		a.queue.AddURL(url.URL)
	}

	// Start scraping
	a.queue.Run(a.collector)
}
