package scraper

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/queue"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/config"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/models"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/parser"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/storage"
	log "github.com/sirupsen/logrus"
)

// Scraper orchestrates the web scraping process.
type Scraper struct {
	config       *config.Config
	client       *webClient
	repository   storage.RestaurantRepository
	michelinURLs []models.GuideURL
}

// Default creates a Scraper instance with default settings.
func Default() (*Scraper, error) {
	cfg := config.Default()

	client, err := newWebClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create web client: %w", err)
	}

	repository, err := storage.NewSQLiteRepository(cfg.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	return new(cfg, client, repository), nil
}

// new creates a new Scraper instance with the provided dependencies.
func new(cfg *config.Config, client *webClient, repository storage.RestaurantRepository) *Scraper {
	s := &Scraper{
		config:     cfg,
		client:     client,
		repository: repository,
	}
	s.initURLs()
	return s
}

// initURLs initializes the default start URLs for all award distinctions.
func (s *Scraper) initURLs() {
	allAwards := []string{
		models.ThreeStars,
		models.TwoStars,
		models.OneStar,
		models.BibGourmand,
		models.SelectedRestaurants,
	}

	for _, distinction := range allAwards {
		url, ok := models.DistinctionURL[distinction]
		if !ok {
			continue
		}

		michelinURL := models.GuideURL{
			Distinction: distinction,
			URL:         url,
		}
		s.michelinURLs = append(s.michelinURLs, michelinURL)
	}
}

// timeTrack tracks the time elapsed for a function call and logs the duration.
func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.WithFields(log.Fields{
		"name":    name,
		"elapsed": elapsed,
	}).Infof("function %s took %s", name, elapsed)
}

// Crawl crawls Michelin Guide Restaurants information from s.michelinURLs.
func (s *Scraper) Crawl(ctx context.Context) error {
	defer timeTrack(time.Now(), "crawl")

	queue := s.client.getQueue()
	collector := s.client.getCollector()
	detailCollector := s.client.createDetailCollector()

	s.setupMainCollectorHandlers(ctx, collector, queue, detailCollector)
	s.setupDetailCollectorHandlers(ctx, detailCollector, queue)

	// Add all URLs to the scraping queue
	for _, url := range s.michelinURLs {
		s.client.addURL(url.URL)
	}

	// Start scraping
	s.client.run()
	return nil
}

// setupMainCollectorHandlers sets up handlers for the main page collector.
func (s *Scraper) setupMainCollectorHandlers(ctx context.Context, collector *colly.Collector, q *queue.Queue, detailCollector *colly.Collector) {
	collector.OnRequest(func(r *colly.Request) {
		attempt := r.Ctx.GetAny("attempt")
		if attempt == nil {
			r.Ctx.Put("attempt", 1)
			attempt = 1
		}
		log.WithFields(log.Fields{
			"url":     r.URL,
			"attempt": attempt,
		}).Debug("→ fetching listing page")
	})

	collector.OnResponse(func(r *colly.Response) {
		log.WithFields(log.Fields{
			"url":    r.Request.URL,
			"status": r.StatusCode,
		}).Debug("→ processing listing page")
	})

	collector.OnScraped(func(r *colly.Response) {
		log.WithField("url", r.Request.URL).Info("✓ listing page parsed")
	})

	collector.OnError(s.createErrorHandler())

	// Extract restaurant URLs from the main page and visit them
	collector.OnXML(restaurantXPath, func(e *colly.XMLElement) {
		url := e.Request.AbsoluteURL(e.ChildAttr(restaurantDetailUrlXPath, "href"))

		location := e.ChildText(restaurantLocationXPath)
		longitude := e.Attr("data-lng")
		latitude := e.Attr("data-lat")

		e.Request.Ctx.Put("location", location)
		e.Request.Ctx.Put("longitude", longitude)
		e.Request.Ctx.Put("latitude", latitude)

		detailCollector.Request(e.Request.Method, url, nil, e.Request.Ctx, nil)
	})

	// Extract and visit next page links
	collector.OnXML(nextPageArrowButtonXPath, func(e *colly.XMLElement) {
		e.Request.Visit(e.Attr("href"))
	})
}

// setupDetailCollectorHandlers sets up handlers for the detail page collector.
func (s *Scraper) setupDetailCollectorHandlers(ctx context.Context, detailCollector *colly.Collector, q *queue.Queue) {
	detailCollector.OnRequest(func(r *colly.Request) {
		attempt := r.Ctx.GetAny("attempt")
		if attempt == nil {
			r.Ctx.Put("attempt", 1)
			attempt = 1
		}
		log.WithFields(log.Fields{
			"attempt": attempt,
			"url":     r.URL,
		}).Debug("→ fetching restaurant detail")
	})

	detailCollector.OnResponse(func(r *colly.Response) {
		log.WithFields(log.Fields{
			"status": r.StatusCode,
			"url":    r.Request.URL,
		}).Debug("← restaurant detail received")
	})

	detailCollector.OnError(s.createErrorHandler())

	// Extract details of each restaurant and save to database
	detailCollector.OnXML(restaurantDetailXPath, func(e *colly.XMLElement) {
		data := s.extractRestaurantData(e)

		log.WithFields(log.Fields{
			"name":        data.Name,
			"distinction": data.Distinction,
			"url":         data.URL,
		}).Debug("✓ restaurant detail extracted")

		if err := s.repository.UpsertRestaurantWithAward(ctx, data); err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"url":   data.URL,
			}).Error("✗ failed to upsert restaurant award")
		}
	})
}

// createErrorHandler creates a reusable error handler for collectors.
func (s *Scraper) createErrorHandler() func(*colly.Response, error) {
	return func(r *colly.Response, err error) {
		attempt := r.Ctx.GetAny("attempt").(int)

		log.WithFields(log.Fields{
			"attempt":     attempt,
			"error":       err,
			"status_code": r.StatusCode,
			"url":         r.Request.URL,
		}).Warnf("✗ request failed on attempt %d", attempt)

		shouldRetry := attempt < s.config.Scraper.MaxRetry
		if shouldRetry {
			if err := s.client.clearCache(r.Request); err != nil {
				log.WithField("error", err).Error("✗ failed to clear cache for ")
			}

			// Exponential backoff for retries
			backoff := time.Duration(attempt) * s.config.Scraper.Delay

			log.WithFields(log.Fields{
				"attempt":     attempt,
				"error":       err,
				"status_code": r.StatusCode,
				"url":         r.Request.URL,
				"backoff":     backoff,
			}).Warnf("⚠ retrying request after %v", backoff)

			time.Sleep(backoff)
			r.Ctx.Put("attempt", attempt+1)
			r.Request.Retry()
		} else {
			log.WithFields(log.Fields{
				"error":         err,
				"status_code":   r.StatusCode,
				"url":           r.Request.URL,
				"final_attempt": attempt,
			}).Error("✗ giving up after max retries")
		}
	}
}

// extractRestaurantData extracts restaurant data from the XML element.
func (s *Scraper) extractRestaurantData(e *colly.XMLElement) storage.RestaurantData {
	url := e.Request.URL.String()
	websiteUrl := e.ChildAttr(restaurantWebsiteUrlXPath, "href")
	name := e.ChildText(restaurantNameXPath)

	address := e.ChildText(restaurantAddressXPath)
	address = strings.ReplaceAll(address, "\n", " ")

	description := e.ChildText(restaurantDescriptionXPath)
	distinction := e.ChildText(restaurantDistinctionXPath)
	greenStar := e.ChildText(restaurantGreenStarXPath)

	priceAndCuisine := e.ChildText(restaurantPriceAndCuisineXPath)
	price, cuisine := parser.SplitUnpack(priceAndCuisine, "·")

	phoneNumber := e.ChildAttr(restaurantPhoneNumberXPath, "href")
	formattedPhoneNumber := parser.ParsePhoneNumber(phoneNumber)
	if formattedPhoneNumber == "" {
		log.WithFields(log.Fields{
			"phone_number": phoneNumber,
			"url":          url,
		}).Debug("invalid phone number")
	}

	facilitiesAndServices := e.ChildTexts(restaurantFacilitiesAndServicesXPath)

	return storage.RestaurantData{
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
}
