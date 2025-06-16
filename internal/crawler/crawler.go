package crawler

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/gocolly/colly/v2/queue"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/michelin"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/models"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/parser"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
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
	sqlitePath = "data/michelin.db"
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
		models.ThreeStars,
		models.TwoStars,
		models.OneStar,
		models.BibGourmand,
		models.SelectedRestaurants,
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
	db, err := gorm.Open(sqlite.Open(sqlitePath), &gorm.Config{
		PrepareStmt: true,
		Logger:      logger.Default.LogMode(logger.Silent), // Disable GORM logging
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
	db.AutoMigrate(&models.Restaurant{}, &models.RestaurantAward{})

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

// upsertRestaurantAward creates or updates a restaurant and its award with simplified change detection.
func (a *App) upsertRestaurantAward(restaurantData RestaurantData) error {
	currentYear := time.Now().Year()

	// Upsert restaurant data
	restaurant := models.Restaurant{
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

	if err := a.database.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "url"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name", "description", "address", "location",
			"latitude", "longitude", "cuisine",
			"facilities_and_services", "phone_number", "website_url",
		}),
	}).Create(&restaurant).Error; err != nil {
		return err
	}

	// Award handling with simplified change detection
	// check if this restaurant have any awards first
	// if if no -> create it. end
	// if yes -> check if award for
	var existingAward models.RestaurantAward
	result := a.database.Where("restaurant_id = ? AND year = ?", restaurant.ID, currentYear).First(&existingAward)

	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return result.Error
	}

	noExistingAwardForCurrentYear := result.Error == gorm.ErrRecordNotFound
	if noExistingAwardForCurrentYear {
		newAward := models.RestaurantAward{
			RestaurantID: restaurant.ID,
			Year:         currentYear,
			Distinction:  restaurantData.Distinction,
			Price:        restaurantData.Price,
			GreenStar:    restaurantData.GreenStar,
		}

		log.WithFields(log.Fields{
			"restaurant_id": restaurant.ID,
			"year":          currentYear,
			"distinction":   restaurantData.Distinction,
		}).Info("creating new award")

		return a.database.Create(&newAward).Error
	}

	// Existing award found - check for changes
	foundExistingAwardForCurrentYear := existingAward.Distinction != restaurantData.Distinction ||
		existingAward.Price != restaurantData.Price ||
		existingAward.GreenStar != restaurantData.GreenStar
	if foundExistingAwardForCurrentYear {
		// Change detected! Use simple time-based logic
		timeSinceUpdate := time.Since(existingAward.UpdatedAt)
		const changeThreshold = 24 * time.Hour

		if timeSinceUpdate > changeThreshold {
			// Significant time has passed - likely a real award change
			// Backdate existing award to previous year and create new one
			previousYear := currentYear - 1

			// Check if previous year already exists to avoid conflicts
			var conflictAward models.RestaurantAward
			conflictResult := a.database.Where("restaurant_id = ? AND year = ?", restaurant.ID, previousYear).First(&conflictAward)

			if conflictResult.Error == gorm.ErrRecordNotFound {
				// Safe to backdate - update existing award to previous year
				existingAward.Year = previousYear
				if err := a.database.Save(&existingAward).Error; err != nil {
					return err
				}

				log.WithFields(log.Fields{
					"restaurant_id":   restaurant.ID,
					"old_distinction": existingAward.Distinction,
					"new_distinction": restaurantData.Distinction,
					"backdated_year":  previousYear,
					"current_year":    currentYear,
				}).Info("backdated existing award and creating new one")

				// Create new award for current year
				newAward := models.RestaurantAward{
					RestaurantID: restaurant.ID,
					Year:         currentYear,
					Distinction:  restaurantData.Distinction,
					Price:        restaurantData.Price,
					GreenStar:    restaurantData.GreenStar,
				}

				return a.database.Create(&newAward).Error
			} else {
				// Conflict exists - just update the current year award
				log.WithFields(log.Fields{
					"restaurant_id": restaurant.ID,
					"conflict_year": previousYear,
					"current_year":  currentYear,
				}).Warn("cannot backdate due to year conflict, updating current award")
			}
		} else {
			// Recent change - likely a correction
			log.WithFields(log.Fields{
				"restaurant_id":   restaurant.ID,
				"old_distinction": existingAward.Distinction,
				"new_distinction": restaurantData.Distinction,
				"hours_since":     timeSinceUpdate.Hours(),
			}).Info("recent change detected, updating existing award")
		}

		// Update existing award with new data
		existingAward.Distinction = restaurantData.Distinction
		existingAward.Price = restaurantData.Price
		existingAward.GreenStar = restaurantData.GreenStar
		return a.database.Save(&existingAward).Error
	}

	// No changes detected - award stays the same
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

// TimeTrack tracks the time elapsed for a function call and logs the duration.
func TimeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.WithFields(log.Fields{
		"name":    name,
		"elapsed": elapsed,
	}).Infof("function %s took %s", name, elapsed)
}

// Crawl crawls Michelin Guide Restaurants information from a.michelinURLs.
func (a *App) Crawl() {
	defer TimeTrack(time.Now(), "crawl")

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
